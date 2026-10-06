#!/usr/bin/env python3
"""Carrega dados FICTICIOS no ambiente de teste (tabela FinancasApp-dev). Nunca toca na tabela de producao.

Cria as listas (status e bancos), centenas de tags, combos e uns 2 mil lancamentos de 2025 ate hoje, mais eventos
futuros (PREVISTO, TRANSFERINDO, CREDITANDO) e transferencias entre bancos, num volume parecido com o uso real.
Os ids sao fixos (gerador com semente), entao rodar de novo sobrescreve os mesmos registros em vez de duplicar.
Para recomecar do zero: python scripts/limpar-espaco.py <space-id> --dev --confirmar

Uso (com a sessao do MFA ativa no terminal; o espaco vem do scripts/seed.sh financas-dev <email-dono>):
  python scripts/dados-ficticios.py <space-id>               # simulacao: so conta o que seria gravado
  python scripts/dados-ficticios.py <space-id> --confirmar   # grava de verdade
"""
import json
import random
import shutil
import subprocess
import sys
import time
from datetime import date, datetime, timedelta, timezone

TABELA = "FinancasApp-dev"   # fixo de proposito: este script nao tem como escrever na producao
AWS = shutil.which("aws") or "aws"
ALFA = "abcdefghijklmnopqrstuvwxyz0123456789"
INICIO = date(2025, 1, 1)
HOJE = date.today()
FIM_FUTURO = HOJE + timedelta(days=150)

STATUS = ["PAGO", "PREVISTO", "CREDITO IN", "CREDITO EX", "CONTAS", "TRANSFERINDO", "CREDITANDO"]
BANCOS = ["Nubank", "Itau", "Bradesco", "Inter", "C6", "Carteira"]
FUTUROS = {"PREVISTO": True, "TRANSFERINDO": True, "CREDITANDO": True}

CATEGORIAS = {   # categoria: (descricoes, faixa de valor, tipo)
    "mercado": (["Mercado", "Supermercado", "Hortifruti", "Padaria", "Acougue"], (25, 480), "saida"),
    "transporte": (["Uber", "Combustivel", "Estacionamento", "Pedagio", "Onibus"], (8, 260), "saida"),
    "casa": (["Aluguel", "Condominio", "Luz", "Agua", "Internet", "Gas"], (60, 2400), "saida"),
    "saude": (["Farmacia", "Consulta", "Exame", "Plano de saude", "Dentista"], (20, 650), "saida"),
    "lazer": (["Cinema", "Restaurante", "Streaming", "Viagem", "Show", "Livraria"], (15, 700), "saida"),
    "educacao": (["Curso", "Escola", "Material escolar", "Livro"], (30, 900), "saida"),
    "compras": (["Roupa", "Eletronico", "Presente", "Casa e decoracao", "Pet shop"], (30, 1200), "saida"),
    "renda": (["Salario", "Freela", "Reembolso", "Rendimento", "Venda"], (150, 7500), "entrada"),
}
ADJ = ["mensal", "extra", "urgente", "fixo", "parcelado", "viagem", "familia", "trabalho", "casa", "criancas",
       "natal", "aniversario", "ferias", "reforma", "carro", "pet", "festa", "estudo", "saude", "online"]


def aws(*args):
    r = subprocess.run([AWS, *args, "--output", "json"], capture_output=True, text=True, encoding="utf-8")
    if r.returncode != 0:
        sys.exit("Erro do aws: " + r.stderr.strip())
    return json.loads(r.stdout) if r.stdout.strip() else {}


def av(v):
    """Python -> formato de atributo do DynamoDB."""
    if isinstance(v, bool):
        return {"BOOL": v}
    if isinstance(v, (int, float)):
        return {"N": str(v)}
    if isinstance(v, str):
        return {"S": v}
    if isinstance(v, list):
        return {"L": [av(x) for x in v]}
    if isinstance(v, dict):
        return {"M": {k: av(x) for k, x in v.items()}}
    raise TypeError(type(v))


def item(d):
    return {k: av(v) for k, v in d.items()}


def gerar(space):
    rng = random.Random(20260506)   # semente fixa: mesmos ids e mesmos dados a cada execucao
    pk = "SPACE#" + space

    def novo_id():
        return "".join(rng.choice(ALFA) for _ in range(20))

    def iso(d, h=None, m=None):
        h = rng.randint(7, 22) if h is None else h
        m = rng.randint(0, 59) if m is None else m
        return datetime(d.year, d.month, d.day, h, m, tzinfo=timezone.utc).strftime("%Y-%m-%dT%H:%M:00.000Z")

    # centenas de tags: categoria + qualificador, mais algumas soltas
    tags = set(CATEGORIAS)
    for cat in CATEGORIAS:
        for a in ADJ:
            tags.add(f"{cat}-{a}")
    tags = sorted(tags)
    tags_da = {c: [t for t in tags if t == c or t.startswith(c + "-")] for c in CATEGORIAS}

    itens = []
    itens.append({"PK": pk, "SK": "CFG#LISTAS", "version": 1, "status": STATUS, "banco": BANCOS,
                  "statusReal": {"PREVISTO": False, "TRANSFERINDO": False, "CREDITANDO": False},
                  "afetaSaldoBanco": {"PAGO": True, "CONTAS": True}, "statusFuturo": FUTUROS})
    itens.append({"PK": pk, "SK": "CFG#TAGS", "version": 1, "usadas": tags})
    combos = [{"tags": [c, f"{c}-{a}"]} for c in ("mercado", "transporte", "casa") for a in ("mensal", "extra", "fixo")]
    itens.append({"PK": pk, "SK": "CFG#COMBOS", "version": 1, "lista": combos})

    def tx(dia, status, tipo, valor, desc, banco, tg, extra=None):
        i = novo_id()
        d = {"PK": pk, "SK": f"TX#{iso(dia)}#{i}", "id": i, "status": status, "tipo": tipo,
             "valor": round(valor, 2), "descricao": desc, "nota": "", "banco": banco, "tags": tg,
             "criadoEm": iso(dia), "criadoPor": "dados-ficticios",
             "excluirDoTotal": status in ("CONTAS", "TRANSFERINDO")}
        d["dataEvento"] = d["SK"].split("#")[1]
        if status.startswith("CREDITO"):
            d["faturaAjuste"], d["valorEstimado"] = 0, False
        if extra:
            d.update(extra)
        itens.append(d)

    dia = INICIO
    while dia <= FIM_FUTURO:
        futuro = dia > HOJE
        # salario no dia 5, aluguel e condominio perto do dia 10
        if dia.day == 5 and not futuro:
            tx(dia, "PAGO", "entrada", rng.uniform(6500, 7500), "Salario", "Itau", ["renda", "renda-mensal"])
        if dia.day == 10:
            st = "PREVISTO" if futuro else "PAGO"
            tx(dia, st, "saida", 2200, "Aluguel", "Itau", ["casa", "casa-mensal", "casa-fixo"])
            tx(dia, st, "saida", rng.uniform(480, 560), "Condominio", "Itau", ["casa", "casa-mensal"])
        # lancamentos do dia: media de ~3 por dia, ate 6 em dias cheios
        for _ in range(rng.choice([0, 1, 2, 2, 3, 3, 4, 5, 6])):
            cat = rng.choices(list(CATEGORIAS), weights=[22, 14, 8, 8, 12, 4, 10, 3])[0]
            descs, (lo, hi), tipo = CATEGORIAS[cat]
            tg = rng.sample(tags_da[cat], rng.randint(1, 3))
            banco = rng.choice(BANCOS)
            if futuro:
                st = rng.choices(["PREVISTO", "CREDITANDO", "TRANSFERINDO"], weights=[8, 1, 1])[0]
            elif tipo == "entrada":
                st = rng.choice(["PAGO", "CREDITO IN"])
            else:
                st = rng.choices(["PAGO", "CREDITO EX", "CONTAS"], weights=[9, 5, 1])[0]
            tx(dia, st, tipo, rng.uniform(lo, hi), rng.choice(descs), banco, tg)
        # uma transferencia entre bancos por semana, em par (saida + entrada)
        if dia.weekday() == 2 and not futuro:
            orig, dest = rng.sample(BANCOS, 2)
            v, par = round(rng.uniform(100, 1500), 2), novo_id()
            tx(dia, "CONTAS", "saida", v, f"Transferencia para {dest}", orig, [], {"transferParId": par})
            tx(dia, "CONTAS", "entrada", v, f"Transferencia de {orig}", dest, [], {"transferParId": par})
        dia += timedelta(days=1)

    seq = {"PK": pk, "SK": "SEQ", "seq": len(itens)}
    return itens, seq


def gravar(itens):
    gravados = 0
    for i in range(0, len(itens), 25):
        pedidos = [{"PutRequest": {"Item": item(d)}} for d in itens[i:i + 25]]
        espera = 0.5
        while pedidos:
            res = aws("dynamodb", "batch-write-item", "--request-items", json.dumps({TABELA: pedidos}))
            sobra = res.get("UnprocessedItems", {}).get(TABELA, [])
            gravados += len(pedidos) - len(sobra)
            pedidos = sobra
            if pedidos:
                time.sleep(espera)
                espera = min(espera * 2, 5)
        print(f"  {gravados}/{len(itens)} gravados", end="\r")
    print()


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    if not args:
        sys.exit(__doc__)
    space = args[0].removeprefix("SPACE#")

    # o espaco precisa existir na tabela de teste (criado pelo seed.sh), senao os dados ficariam orfaos
    meta = aws("dynamodb", "get-item", "--table-name", TABELA,
               "--key", json.dumps({"PK": {"S": "SPACE#" + space}, "SK": {"S": "META"}}))
    if not meta.get("Item"):
        sys.exit(f"O espaco {space} nao existe na tabela {TABELA}. Rode antes: scripts/seed.sh financas-dev <email>")

    itens, seq = gerar(space)
    tx = [d for d in itens if d["SK"].startswith("TX#")]
    futuros = sum(1 for d in tx if d["status"] in FUTUROS)
    print(f"Espaco {space} em {TABELA}: {len(tx)} lancamentos ({futuros} futuros), "
          f"{len(itens) - len(tx)} configuracoes, 1 contador seq.")
    if "--confirmar" not in sys.argv:
        print("Simulacao. Rode de novo com --confirmar para gravar.")
        return
    gravar(itens + [seq])
    print("Pronto. Recarregue o app do dev (pode ser preciso sair e entrar de novo).")


if __name__ == "__main__":
    main()
