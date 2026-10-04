#!/usr/bin/env python3
"""Apaga os lancamentos (TX#) e as configuracoes compartilhadas (CFG#) de um espaco.

Mantem sempre: USER# (membros e aparencia pessoal), SPACE# (vinculos) e META.

Uso (com a sessao do MFA ativa no terminal):
  python scripts/limpar-espaco.py <space-id>               # simulacao: so conta
  python scripts/limpar-espaco.py <space-id> --confirmar   # apaga de verdade
"""
import json
import shutil
import subprocess
import sys
import time

TABELA = "FinancasApp"
AWS = shutil.which("aws") or "aws"


def aws(*args):
    r = subprocess.run([AWS, *args, "--output", "json"], capture_output=True, text=True, encoding="utf-8")
    if r.returncode != 0:
        sys.exit("Erro do aws: " + r.stderr.strip())
    return json.loads(r.stdout) if r.stdout.strip() else {}


def listar(space):
    valores = json.dumps({":p": {"S": "SPACE#" + space}})
    res = aws("dynamodb", "query", "--table-name", TABELA,
              "--key-condition-expression", "PK = :p",
              "--expression-attribute-values", valores,
              "--projection-expression", "PK, SK")
    return [i for i in res.get("Items", [])]


def apagavel(sk):
    return sk.startswith("TX#") or sk.startswith("CFG#")


def main():
    args = [a for a in sys.argv[1:] if not a.startswith("--")]
    if not args:
        sys.exit(__doc__)
    space = args[0].removeprefix("SPACE#")
    confirmar = "--confirmar" in sys.argv

    itens = listar(space)
    alvo = [i for i in itens if apagavel(i["SK"]["S"])]
    mantidos = len(itens) - len(alvo)
    tx = sum(1 for i in alvo if i["SK"]["S"].startswith("TX#"))
    print(f"Espaco {space}: {len(itens)} itens; apagaria {len(alvo)} ({tx} lancamentos, {len(alvo)-tx} configuracoes); mantem {mantidos}.")
    if not alvo:
        return
    if not confirmar:
        print("Simulacao. Rode de novo com --confirmar para apagar.")
        return

    apagados = 0
    for i in range(0, len(alvo), 25):
        pedidos = [{"DeleteRequest": {"Key": {"PK": it["PK"], "SK": it["SK"]}}} for it in alvo[i:i + 25]]
        espera = 0.5
        while pedidos:
            res = aws("dynamodb", "batch-write-item", "--request-items", json.dumps({TABELA: pedidos}))
            sobra = res.get("UnprocessedItems", {}).get(TABELA, [])
            apagados += len(pedidos) - len(sobra)
            pedidos = sobra
            if pedidos:
                time.sleep(espera)
                espera = min(espera * 2, 5)
        print(f"  {apagados}/{len(alvo)} apagados", end="\r")
    print(f"\nPronto: {apagados} itens apagados.")


if __name__ == "__main__":
    main()
