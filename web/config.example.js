// Copie para config.js e preencha com os Outputs do "sam deploy" (config.js nao vai para o git).
window.FINANCAS_CONFIG = {
  apiUrl: "https://<id>.lambda-url.<regiao>.on.aws/",
  clientId: "<ClientId>",
  loginDomain: "<prefixo>.auth.<regiao>.amazoncognito.com"
  // ambiente: "dev",   // so no config.dev.js: mostra a faixa laranja "AMBIENTE DE TESTE" no topo
};
