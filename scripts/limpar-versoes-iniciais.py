#!/usr/bin/env python3
"""Apaga as versoes 1 de lancamentos que NUNCA foram alterados (sobras de criacao e importacao) em HIST#<espaco>.

Regra (ver .ai/specs/historico/plan.md, decisao 8): a versao so nasce na alteracao. As versoes 1 gravadas na criacao ou na
importacao tem acao=insert e porSub preenchido, e sao a UNICA versao do lancamento. Elas nao protegem nada.

NUNCA apaga:
  - nenhuma versao de um lancamento que tem versao 2 ou mais (inclusive a versao 1 dele, tenha porSub vazio ou preenchido);
  - versao 1 com porSub vazio (e o "estado antes da primeira alteracao", que nasce junto da versao 2);
  - versao 1 cuja acao nao seja insert.

Seguranca: o padrao e SIMULACAO (so le e conta). Antes de apagar cada item, o script le de novo as versoes daquele
lancamento (se apareceu versao 2, pula), grava o item inteiro num arquivo de backup (.jsonl, fora do git) e so entao
apaga, com condicao no proprio DynamoDB. Apaga devagar (--ritmo) e no maximo --limite itens por execucao.

Uso (com a sessao do MFA ativa no terminal; a role da Lambda tem Deny em HIST#, entao rode com o seu acesso):
  python scripts/limpar-versoes-iniciais.py <space-id> --dev                 # simulacao no ambiente de teste
  python scripts/limpar-versoes-iniciais.py <space-id> --dev --confirmar     # apaga de verdade (ate 500 itens)
  python scripts/limpar-versoes-iniciais.py <space-id> --confirmar --limite 200 --ritmo 1
Opcoes:
  --dev            tabela FinancasApp-dev (sem isso, a de producao FinancasApp)
  --confirmar      apaga de verdade (pede para digitar APAGAR)
  --limite N       apaga no maximo N itens por execucao (padrao 500; 0 = sem limite)
  --ritmo X        apagamentos por segundo (padrao 2; a tabela tem 5 WCU)
  --leitura X      unidades de leitura por segundo ao ler a particao (padrao 2; a tabela tem 5 RCU)
O arquivo de backup (backup-versoes-*.jsonl) tem dados reais: nao commite.
"""
import json
import os
import shutil
import subprocess
import sys
import time
from datetime import datetime

TABELA = "FinancasApp-dev" if "--dev" in sys.argv else "FinancasApp"
AWS = shutil.which("aws") or "aws"


def opcao(nome, padrao):
    if nome in sys.argv:
        i = sys.argv.index(nome)
        try:
            return float(sys.argv[i + 1])
        except (IndexError, ValueError):
            sys.exit("Falta o valor de " + nome)
    return padrao


class SemCapacidade(Exception):
    pass


class CondicaoFalhou(Exception):
    pass


def decodificar(b):
    """Bytes -> texto: UTF-8 e, se nao for, cp1252 (console do Windows)."""
    b = b or b""
    try:
        return b.decode("utf-8")
    except UnicodeDecodeError:
        return b.decode("cp1252", errors="replace")


def aws(*args):
    """Chama o AWS CLI. Levanta SemCapacidade se a tabela recusar por capacidade e CondicaoFalhou se a condicao nao valer."""
    # O AWS CLI no Windows pode devolver o texto na codificacao do console (cp1252) quando o item tem acento: le os bytes
    # e decodifica aqui, sem deixar o Python adivinhar.
    env = dict(os.environ, PYTHONUTF8="1", PYTHONIOENCODING="utf-8")
    r = subprocess.run([AWS, *args, "--output", "json"], capture_output=True, env=env)
    saida, erro = decodificar(r.stdout), decodificar(r.stderr)
    if r.returncode != 0:
        if "ProvisionedThroughputExceeded" in erro or "ThrottlingException" in erro:
            raise SemCapacidade()
        if "ConditionalCheckFailed" in erro:
            raise CondicaoFalhou()
        sys.exit("Erro do aws: " + erro.strip())
    return json.loads(saida) if saida.strip() else {}


def com_espera(*args):
    """Repete com espera crescente enquanto a tabela recusa por capacidade."""
    espera = 3.0
    for _ in range(8):
        try:
            return aws(*args)
        except SemCapacidade:
            print(f"  tabela sem capacidade agora; esperando {espera:.0f}s...")
            time.sleep(espera)
            espera = min(espera * 2, 60)
    sys.exit("A tabela continuou sem capacidade. Pare, espere alguns minutos e rode de novo.")


def id_e_versao(sk):
    """'TX#<id>#00000001' -> ('<id>', 1). None se a SK nao tiver esse formato."""
    if not sk.startswith("TX#"):
        return None
    meio, _, num = sk[3:].rpartition("#")
    if not meio or not num.isdigit():
        return None
    return meio, int(num)


def classificar(itens):
    """Recebe os itens lidos (com SK, versao, acao, porSub) e separa o que pode apagar.

    Devolve (apagaveis, resumo). apagaveis = lista de (id, SK).
    """
    por_id = {}
    for it in itens:
        sk = it["SK"]["S"]
        iv = id_e_versao(sk)
        if iv is None:
            continue
        por_id.setdefault(iv[0], {})[iv[1]] = it
    apagaveis, alterados, vazios, outras = [], 0, 0, 0
    for lid, vs in por_id.items():
        if max(vs) >= 2:
            alterados += 1  # tem versao 2 ou mais: fica tudo
            continue
        v1 = vs.get(1)
        if v1 is None:
            outras += 1
            continue
        acao = v1.get("acao", {}).get("S", "")
        por = v1.get("porSub", {}).get("S", "")
        if acao != "insert":
            outras += 1
        elif por == "":
            vazios += 1  # v1 "estado antes" sem versao 2: estranho, fica e e relatado
        else:
            apagaveis.append((lid, v1["SK"]["S"]))
    resumo = {"lancamentos": len(por_id), "alterados": alterados, "v1_sem_autor_sozinha": vazios, "outras": outras,
              "apagaveis": len(apagaveis)}
    return apagaveis, resumo


def ler_particao(space, leitura):
    """Le HIST#<space> inteira, devagar: depois de cada pagina espera o que o consumo pede para ficar em 'leitura' RCU/s."""
    itens, chave = [], None
    valores = json.dumps({":p": {"S": "HIST#" + space}, ":s": {"S": "TX#"}})
    while True:
        cmd = ["dynamodb", "query", "--table-name", TABELA, "--no-paginate", "--limit", "300",
               "--key-condition-expression", "PK = :p AND begins_with(SK, :s)",
               "--expression-attribute-values", valores,
               "--projection-expression", "SK, versao, acao, porSub",
               "--return-consumed-capacity", "TOTAL"]
        if chave:
            cmd += ["--exclusive-start-key", json.dumps(chave)]
        t0 = time.time()
        res = com_espera(*cmd)
        itens += res.get("Items", [])
        gasto = res.get("ConsumedCapacity", {}).get("CapacityUnits", 0)
        print(f"  lidos {len(itens)} itens", end="\r")
        chave = res.get("LastEvaluatedKey")
        if not chave:
            break
        time.sleep(max(0.0, gasto / max(leitura, 0.1) - (time.time() - t0)))
    print()
    return itens


def versoes_do_lancamento(space, lid):
    valores = json.dumps({":p": {"S": "HIST#" + space}, ":s": {"S": "TX#" + lid + "#"}})
    res = com_espera("dynamodb", "query", "--table-name", TABELA,
                     "--key-condition-expression", "PK = :p AND begins_with(SK, :s)",
                     "--expression-attribute-values", valores)
    return res.get("Items", [])


def main():
    args = []
    pular = False
    for a in sys.argv[1:]:
        if pular:
            pular = False
        elif a in ("--limite", "--ritmo", "--leitura"):
            pular = True
        elif not a.startswith("--"):
            args.append(a)
    if not args:
        sys.exit(__doc__)
    space = args[0].removeprefix("HIST#").removeprefix("SPACE#")
    confirmar = "--confirmar" in sys.argv
    limite = int(opcao("--limite", 500))
    ritmo = opcao("--ritmo", 2.0)
    leitura = opcao("--leitura", 2.0)

    print(f"Tabela {TABELA}, espaco {space}. Lendo HIST#{space} (devagar)...")
    itens = ler_particao(space, leitura)
    apagaveis, r = classificar(itens)
    print(f"{len(itens)} versoes em {r['lancamentos']} lancamentos.")
    print(f"  mantidos por terem versao 2 ou mais: {r['alterados']} lancamentos (nenhuma versao deles sera tocada)")
    print(f"  versao 1 sem autor e sem versao 2 (estranho, mantida): {r['v1_sem_autor_sozinha']}")
    print(f"  outras (acao diferente de insert etc., mantidas): {r['outras']}")
    print(f"  APAGAVEIS (versao 1 unica, insert, com autor): {r['apagaveis']}")
    if not apagaveis:
        return
    alvo = apagaveis if limite == 0 else apagaveis[:limite]
    print(f"Esta execucao apagaria {len(alvo)} itens (limite {limite or 'sem limite'}).")
    if not confirmar:
        print("Simulacao: nada foi apagado. Rode de novo com --confirmar.")
        return
    if input(f"Vai apagar {len(alvo)} itens de {TABELA}. Digite APAGAR para continuar: ").strip() != "APAGAR":
        sys.exit("Cancelado.")

    arq = "backup-versoes-" + datetime.now().strftime("%Y%m%d-%H%M%S") + ".jsonl"
    apagados = pulados = 0
    intervalo = 1.0 / max(ritmo, 0.1)
    bk = None  # o backup so e criado quando houver o que gravar
    try:
        for lid, sk in alvo:
            t0 = time.time()
            atuais = versoes_do_lancamento(space, lid)
            nums = [id_e_versao(i["SK"]["S"])[1] for i in atuais if id_e_versao(i["SK"]["S"])]
            v1 = next((i for i in atuais if i["SK"]["S"] == sk), None)
            if v1 is None or nums != [1] or v1.get("acao", {}).get("S") != "insert" or v1.get("porSub", {}).get("S", "") == "":
                pulados += 1  # mudou desde a leitura: nao toca
                continue
            if bk is None:
                bk = open(arq, "a", encoding="utf-8")
            bk.write(json.dumps(v1, ensure_ascii=False) + "\n")
            bk.flush()
            try:
                com_espera("dynamodb", "delete-item", "--table-name", TABELA,
                           "--key", json.dumps({"PK": {"S": "HIST#" + space}, "SK": {"S": sk}}),
                           "--condition-expression", "attribute_exists(SK) AND acao = :i AND versao = :u",
                           "--expression-attribute-values", json.dumps({":i": {"S": "insert"}, ":u": {"N": "1"}}))
            except CondicaoFalhou:
                pulados += 1  # o item mudou ou ja nao existe: nao apaga
                continue
            apagados += 1
            print(f"  {apagados}/{len(alvo)} apagados", end="\r")
            time.sleep(max(0.0, intervalo - (time.time() - t0)))
    finally:
        if bk is not None:
            bk.close()
    print(f"\nPronto: {apagados} apagados, {pulados} pulados (mudaram desde a leitura). Backup em {arq} (nao commite).")


if __name__ == "__main__":
    main()
