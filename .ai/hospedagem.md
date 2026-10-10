# Hospedagem (S3 + CloudFront) e decisão sobre o `/api/*`

**Decidido: opção B**, a API continua sendo chamada direto na Function URL (com CORS), e o CloudFront serve só o front.
Motivos: o token do Cognito já protege os dados; rotear a API pelo CloudFront (OAC) exige `AuthType: AWS_IAM`, hash do
corpo em `x-amz-content-sha256` nos `POST`/`PUT`, token em outro cabeçalho (o OAC sobrescreve o `Authorization`) e
complica o desenvolvimento em `localhost`, e o ganho de segurança é pequeno (o CloudFront continua público, então um
anônimo ainda chega à Lambda por ele; limitar abuso de verdade pede WAF, que custa). Reavaliar com domínio próprio,
WAF ou mais usuários; a troca é localizada (URL base em `config.js`, cabeçalho em `api.js` e `auth.go`, template).

Pronto (PR #5, no ar):
- `template.yaml`: bucket S3 privado, OAC, distribuição (HTTPS, cache gerenciado, cabeçalhos de segurança), política do
  bucket; o CORS da Lambda e as URLs de retorno/saída do Cognito aceitam o CloudFront **e** `localhost`; parâmetro
  `ReservedConcurrency` (teto de execuções simultâneas da Lambda), **desligado (0)**: a conta tem limite de 10 execuções
  simultâneas no total (`aws lambda get-account-settings`) e a AWS exige manter 10 sem reserva, então nenhuma reserva
  é possível; esse limite de 10 da conta já funciona como teto natural. Para reservar de verdade, pedir aumento de cota
  em Service Quotas. Outputs novos: `FrontUrl`, `FrontBucket`, `DistributionId`.
- `scripts/publicar-front.ps1`: envia `web/` ao bucket (inclui `config.js`, que fica fora do git) e invalida o cache.
- Passos: sessão MFA, `.\scripts\deploy.ps1 <codigo>` (cria bucket e distribuição; leva alguns minutos), depois
  `.\scripts\publicar-front.ps1` (use `-Simular` antes) e abrir o `FrontUrl`.

## Decisão registrada
- Hospedagem: **S3 + CloudFront** (GitHub Pages descartado; `localhost` não serve para ela: sem HTTPS o PKCE e o
  Cognito não funcionam).
