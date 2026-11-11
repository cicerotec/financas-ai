// Monta app-android/www a partir de web/ para um ambiente (dev ou prod). Chamado por scripts/app-android.ps1; testado em web/app-android.test.js.
//   node app-android/montar-www.js dev|prod
// Regras (.ai/specs/app-android/spec.md, R1, R2 e R4): www/ e gerado e nunca editado nem versionado; o config.js do pacote e o do ambiente pedido
// (web/config.dev.js no dev, web/config.js no prod) e nunca o do outro; sem o config do ambiente, termina com erro e nao toca em www/.
const fs = require('fs');
const path = require('path');

const FORA = /(^|\/)(config\.example\.js|config\.js|config\.dev\.js)$|\.test\.js$/; // config.js entra depois, pelo ambiente

function montar({raiz, ambiente}){
  if(ambiente !== 'dev' && ambiente !== 'prod') throw new Error('Ambiente invalido: use dev ou prod.');
  const web = path.join(raiz, 'web');
  const www = path.join(raiz, 'app-android', 'www');
  const nomeConfig = ambiente === 'dev' ? 'config.dev.js' : 'config.js';
  const origem = path.join(web, nomeConfig);
  if(!fs.existsSync(origem)) throw new Error('web/' + nomeConfig + ' nao existe. Copie web/config.example.js para web/' + nomeConfig + ' e preencha com os Outputs do deploy.');
  const config = fs.readFileSync(origem, 'utf8');
  const dev = /ambiente\s*:\s*["']dev["']/.test(config);
  if(ambiente === 'dev' && !dev) throw new Error('web/config.dev.js precisa ter ambiente: "dev" (e a faixa AMBIENTE DE TESTE do app dev).');
  if(ambiente === 'prod' && dev) throw new Error('web/config.js tem ambiente: "dev": o pacote prod nao pode ser o do dev.');

  fs.rmSync(www, {recursive: true, force: true});
  (function copiar(dir){
    for(const e of fs.readdirSync(path.join(web, dir), {withFileTypes: true})){
      const rel = dir ? dir + '/' + e.name : e.name;
      if(FORA.test(rel)) continue;
      if(e.isDirectory()){ fs.mkdirSync(path.join(www, rel), {recursive: true}); copiar(rel); }
      else { fs.mkdirSync(path.join(www, path.dirname(rel)), {recursive: true}); fs.copyFileSync(path.join(web, rel), path.join(www, rel)); }
    }
  })('');
  fs.writeFileSync(path.join(www, 'config.js'), config);
  return www;
}

module.exports = montar;
if(require.main === module){
  try{ console.log('www montado em ' + montar({raiz: path.resolve(__dirname, '..'), ambiente: process.argv[2]})); }
  catch(e){ console.error(e.message); process.exit(1); }
}
