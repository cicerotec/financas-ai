# Intent: o app Android, fora do navegador

Este documento diz **o quê e por quê**; o como está em [`plan.md`](plan.md) e o roteiro verificável, em [`spec.md`](spec.md).

**Estado:** rascunho (2026-11-01). Fase 1 implementada em 2026-11-11 (ver `plan.md` e `spec.md`); `applicationId` decidido: `br.com.cicerotec.financas`.

## O problema

O app é uma página web (`web/`, JavaScript puro, sem build) que o dono usa no Android pelo navegador ou como PWA instalada. Isso limita o que ele pode fazer:

- O token do Cognito fica em `localStorage`, que qualquer XSS lê (ponto em aberto do item 18 de [`security.md`](../../security.md)).
- Não há como pedir digital ou rosto ao abrir, receber notificação do próprio app (os avisos vão só pelo Telegram) nem guardar dado no aparelho de forma
  confiável: o navegador pode apagar o `localStorage` e o IndexedDB sem avisar (ver [`offline-e-sincronizacao.md`](../../offline-e-sincronizacao.md), seção 10).
- Uma PWA não aparece na Play Store como um app de verdade, e o dono quer distribuir por lá, primeiro para duas pessoas e depois para outras.

## O que queremos

1. **Um app Android instalável**, com ícone, janela e botão voltar próprios, que abre **sem o navegador** e roda as mesmas telas de hoje.
2. **Capacidades nativas**, uma por vez e sem pressa: token no armazenamento seguro do Android (Keystore), trava por biometria, notificação push como
   canal de aviso e, no futuro, uso sem internet com a projeção do saldo no aparelho.
3. **Os mesmos dois ambientes de hoje**, `dev` (dados fictícios, faixa laranja) e `prod`, podendo ficar instalados lado a lado no mesmo aparelho.
4. **Publicação na Play Store**, começando por teste interno (duas pessoas) e chegando à produção aberta quando o app multiusuário estiver pronto.

## Decisões já tomadas (conversa de 2026-11-01)

- A tecnologia é o **Capacitor**: o `web/` atual vai **empacotado dentro do app** (não carrega uma URL) e os recursos nativos entram por plugins. Não se reescreve
  nenhuma tela.
- Foram descartados: **TWA/Bubblewrap** (continua dependendo do Chrome e não dá acesso nativo) e **app nativo em Kotlin ou Flutter** (dois produtos para manter, sem
  ganho enquanto a tela web resolve).
- O objetivo é **capacidade nativa** (não só tirar a barra de endereço), e o dono tem Android Studio e um aparelho para testar.
- **Backend e regras não mudam por causa do app.** O app é outro cliente da mesma API: o servidor continua sendo a verdade, e o front continua sem decidir regra
  (`arquitetura.md`). O que muda no servidor é infraestrutura (origem permitida no CORS, URL de retorno do Cognito) e, mais tarde, o canal de push.

## Limites (o que fica de fora)

- **iOS.** O Capacitor permitiria, mas não é desejo agora.
- **Uso sem internet.** Continua de futuro, como decidiu o dono em 2026-10-20. O app só abre o caminho (o IndexedDB passa a ser confiável fora do navegador). Sem
  rede, o app abre, porque as telas são locais, e as chamadas falham como falham hoje.
- **Telas novas** que não sejam necessárias para uma capacidade nativa. A tela de bloqueio da biometria é nova e **pede demo aprovada antes do código**
  (`guardrails.md`, "Telas").
- **Cadastro aberto, MFA, limite de taxa, cifragem dos dados e visibilidade por banco.** São de [`../app-multiusuario/`](../app-multiusuario/intent.md),
  [`privacidade-e-cifragem.md`](../../privacidade-e-cifragem.md) e do item 27, e **precedem a abertura na Play**, não a publicação para duas pessoas.
- **Segurança por biometria não é cofre.** A trava protege contra quem pega o aparelho desbloqueado; não protege contra aparelho com root nem substitui a senha
  e o MFA do Cognito.

## Custo

- Conta de desenvolvedor da Play: taxa única (confirmar o valor atual no Play Console).
- Push (FCM) e Keystore não têm custo. O build é local; build no CI é opcional e fica para depois.
- Cada mudança de tela passa a exigir **uma nova versão na Play**, porque o `web/` fica dentro do app. Isso é custo de processo, não de dinheiro: ver o fluxo
  de release no plan.

## Pronto quando

- O app instala no aparelho do dono, abre sem o navegador, entra pelo Cognito (Hosted UI no navegador do sistema), mostra as mesmas abas e grava e lê lançamentos
  no `dev`.
- O token **não está** em `localStorage`: fica no armazenamento seguro, e sair do app limpa tudo.
- O botão voltar do Android se comporta como a demo `demo-voltar-navegador.html` descreve.
- O app `dev` e o `prod` coexistem no mesmo aparelho, cada um com o seu nome, ícone e dados.
- A biometria trava a abertura quando a pessoa a liga, com demo aprovada.
- Um build assinado está no teste interno da Play, instalado pelas duas pessoas.
- `security.md`, `hospedagem.md`, `README.md` e `BACKLOG.md` ficam em dia.

## Para o dono decidir

- **Identificador do app** (decidido em 2026-11-11: `br.com.cicerotec.financas`; `applicationId`, ex.: `br.com.<algo>.financas`). É **permanente** depois do primeiro envio à Play: escolher com cuidado. O `dev` usa o
  mesmo com sufixo `.dev`.
- **Conta da Play:** pessoal ou de organização. Contas pessoais novas costumam exigir um teste fechado com um número mínimo de testadores durante algumas semanas
  antes da produção (confirmar a regra vigente); duas pessoas não bastam. Organização exige verificação da empresa.
- **Retorno do login:** esquema próprio (`br.com.<algo>.financas://auth`, mais simples) ou App Link por `https` no CloudFront (mais difícil de sequestrar, mas
  exige publicar um arquivo de verificação e mexer no front). **Decidido em 2026-11-11:** esquema próprio com PKCE e `state` na fase 2; o App Link volta na fase 5, com a chave de release.
- **Armazenamento seguro** (decidido em 2026-11-11): plugin nosso em Java sobre o Android Keystore, sem plugin de terceiros.
- **Se o acesso aos plugins sem bundler não funcionar**, o front ganharia um passo mínimo de build. Hoje o front não tem build; isso só se decide se for preciso.
- **Biometria:** opcional e ligada pela pessoa (proposta) ou obrigatória.
- **Distribuição inicial:** APK de release assinado, direto entre as duas pessoas, enquanto a conta da Play não estiver pronta, ou esperar o teste interno. Na Play o
  formato é sempre o AAB; o APK serve ao desenvolvimento e a esta distribuição direta.
