# Privacidade dos dados financeiros: cifragem no servidor (ideia para o futuro)

Registro de uma conversa de 2026-10-20. **Nada aqui foi decidido ou implementado.** É memória do que foi discutido, para
retomar depois. O documento continua aberto: novas perguntas e conclusões entram aqui.

Relacionado: `offline-e-sincronizacao.md` (uso sem internet e core independente da Lambda; estará em `.ai/` quando o PR
dele for mergeado), [`arquitetura.md`](arquitetura.md) (camadas, histórico imutável, "Saldo e totais calculados no
servidor"), [`security.md`](security.md).

## 1. O que o dono quer

Em uma aplicação financeira, **nenhum ser humano** deve conseguir ver quanto uma pessoa tem ao abrir o DynamoDB (console,
`select`, backup, export, dump vazado). O `service` continua trabalhando com os valores normalmente: o dado fica cifrado
no banco, e o serviço o decifra com chave própria e com a autorização do usuário. É o que se faz em ambientes de produção
profissionais: quem consulta o banco vê texto cifrado; comportamento se estuda em dev e homologação, com dados fictícios.

**Isto não é cifragem de ponta a ponta.** O servidor enxerga os valores quando precisa. Por isso **preserva** a decisão de
`arquitetura.md` de que o servidor calcula saldo, cascata, fechamento, fatura e totais, e não obriga o aparelho a
calcular. (Uma versão anterior da conversa tratou, por engano, de cifragem de ponta a ponta; ela foi descartada como
ponto de partida. Se um dia for desejada, é outro modelo, que desfaz o cálculo no servidor.)

## 2. Modelo de ameaça

| Protege contra | Não protege contra |
|---|---|
| Abrir o console e ler a tabela; rodar `select`; ler backup, export ou PITR; vazamento de dump; olhar por engano; um acesso ao banco sem acesso à chave | Quem consegue alterar o código da Lambda ou assumir a role dela: essa pessoa faz o serviço decifrar |

A proteção real depende de **separação de funções** (quem lê o banco não pode usar a chave; quem faz deploy não administra
a chave). Num projeto de uma pessoa só, o mesmo humano pode ser dono do código, do IAM e da chave: a cifragem protege de
vazamento e de olhar por engano, mas contra má-fé a defesa é **auditar** (alarme em `Decrypt` fora da Lambda), não impedir.

## 3. Desenho em duas formas

**Forma 1: chave gerenciada do KMS no próprio DynamoDB.**
- Quase não muda código. Quem não tem permissão na chave não consegue ler a tabela (confirmar na documentação da AWS).
- É tudo ou nada: sem a chave nem a estrutura aparece, o que dificulta depurar produção.

**Forma 2: cifragem por campo (envelope encryption), com chave por espaço (preferida).**
- Uma chave de dados por espaço, guardada **embrulhada** pelo KMS. A Lambda a decifra uma vez e a mantém em memória no
  ciclo de vida do container (menos chamadas ao KMS, menos custo).
- Cada campo sensível é cifrado com AES-GCM. A chave do registro (`PK`/`SK`) entra como dado autenticado, para ninguém
  trocar um valor de lugar.
- `PK`, `SK` e metadados ficam em claro: um `select` mostra estrutura e datas, mas nenhum valor.
- O **contexto de criptografia** do KMS carrega o ID do espaço: a chave só abre com o contexto certo (amarra a criptografia
  à autorização) e o CloudTrail registra qual espaço foi decifrado e por quem.
- **Apagar a chave de um espaço** torna os dados irrecuperáveis (serve a um pedido de exclusão).
- A AWS tem uma biblioteca de cifragem de campos para o DynamoDB (AWS Database Encryption SDK); verificar o suporte a Go
  antes de decidir entre ela e uma implementação própria.

**Campos candidatos (a confirmar):** valor, descrição, nota, saldo inicial, valores de fatura e de fechamento e, talvez,
tags e banco. Tags como "médico" também dizem muito.

## 4. Onde encaixa nas camadas

A cifragem é detalhe do adaptador de banco: uma interface `Cifrador` declarada pelo `service` e implementada com KMS; o
`repository` cifra ao gravar e decifra ao ler. `domain`, `service` e `controller` nunca veem texto cifrado, e só o
`repository` conhece as chaves do banco, como já diz `arquitetura.md`. Qualquer entrada futura (fila, worker, outro
executável) herda a proteção porque passa pelo mesmo `repository`.

## 5. Armadilhas específicas do projeto

- **Histórico imutável.** A arquitetura prevê `Deny` de atualizar e apagar o histórico para a role da Lambda. As versões
  antigas já gravadas estão em claro e cifrá-las exige reescrever. Opções: migração única com uma role excepcional,
  auditada, ou deixar o antigo em claro e cifrar só dali para a frente. **Decisão pendente.**
- **Princípio 3 de `arquitetura.md`.** Cifrar adiciona campos (versão da chave, nonce, marcador de campo cifrado): nenhum
  campo novo no DynamoDB sem discutir antes.
- **Dados em claro hoje.** Dev, homologação e os scripts de `scripts/` podem ter cópias de dados reais. Dev e homol devem
  usar dados fictícios e outra chave; nunca copiar produção.
- **Logs.** Os do CloudWatch não podem imprimir valores.
- **Avisos no Telegram.** O servidor monta a mensagem com valor e ela passa pelo Telegram em claro. Cifrar no banco não
  muda isso; é decisão à parte (aviso com ou sem valor).
- **Filtros no banco.** Campo cifrado não pode ser filtrado nem ordenado pelo DynamoDB. O serviço já carrega o mês e
  calcula em memória, então é aceitável.
- **Rotação.** Rotacionar a chave principal do KMS é gratuito e não reescreve dados. Rotacionar a chave de dados exige
  reescrever tudo e o histórico imutável complica: evitar.
- **Detecção de possível duplicado** (ver o documento de offline) precisa comparar valores: acontece no servidor, depois de
  decifrar, sem índice em claro.

## 6. Como tornar a separação real

- **Conta AWS separada para produção**, onde humanos leem o DynamoDB mas **não** têm `kms:Decrypt`.
- **Alarme** no CloudTrail para qualquer `Decrypt` feito por quem não seja a role da Lambda.
- Acesso de emergência ("break-glass") com MFA e alerta.
- Quem faz o deploy não administra a chave (só se cumpre em parte num projeto de uma pessoa).

## 7. Custo

Um par de chaves do KMS e algumas milhares de chamadas por mês custam, nessa escala, centavos a poucos dólares (**confirmar
o preço atual**). Com a chave de dados em cache na Lambda, as chamadas são poucas. A complexidade está em desenhar bem.

## 8. Quando

Hoje são duas pessoas e o dono é uma delas, então o impacto prático é pequeno. A urgência vem **antes de abrir o app a
outros usuários** (mesmo gatilho do uso offline): migrar depois, com dado de terceiros já gravado em claro, é mais caro
(reescrever dados e histórico). Offline, fila e core independente da Lambda não mudam por causa disto; basta que todo novo
ponto de entrada use o mesmo adaptador de banco.

## Em aberto
- Proteger só do dono e de acessos acidentais, ou também de quem tiver acesso à conta AWS no futuro (equipe)?
- Cifrar só valores e notas, ou também descrição, tags e banco?
- Histórico antigo: deixar em claro e cifrar daqui para a frente, ou migração única com role excepcional?
- O aviso do Telegram pode seguir com valor?
- Forma 1 (tabela inteira com chave do KMS) ou forma 2 (por campo, com chave por espaço)? Suporte a Go da biblioteca da AWS.
- Estratégia de separação de funções: conta separada para produção, alarmes e acesso de emergência.
- Revisar `scripts/` e as stacks dev/homol quanto a dados reais.
