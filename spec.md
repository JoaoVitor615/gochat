# GoChat — especificação funcional e técnica

> Documento de referência do estado atual do projeto. Descreve o comportamento presente no código e separa explicitamente as decisões/funções ainda planejadas. Não deve ser interpretado como garantia de conectividade entre quaisquer duas redes.

## 1. Visão do produto

GoChat é um aplicativo de conversa entre peers, operado inicialmente pelo terminal. A aplicação usa uma API de Discovery para trocar metadados de conexão e iniciar convites, mas o tráfego de mensagens deve ir diretamente de um peer ao outro por libp2p/QUIC.

O Discovery não é um servidor de chat: não recebe, persiste nem retransmite conteúdo de mensagens. O Redis do Discovery guarda somente presença/endereço temporário e convites de curta duração. As mensagens, contatos, identidade e fila de saída ficam no dispositivo do usuário.

### Objetivos atuais

- Criar uma identidade libp2p persistente por instalação/usuário do sistema operacional.
- Permitir que usuários troquem convites para aprender o PeerID e os endereços candidatos um do outro.
- Anunciar endereços candidatos através de heartbeats para o Discovery.
- Descobrir, por um observer libp2p hospedado junto ao Discovery, o endereço UDP/QUIC visto externamente.
- Abrir conexões P2P diretas e trocar mensagens com confirmação de persistência no destinatário.
- Persistir contatos, histórico e mensagens ainda não entregues localmente em bbolt.
- Exibir convites, contatos, histórico e envio no TUI.

### Não objetivos do estado atual

- Não existe relay de mensagens, relay circuit ou fallback por servidor.
- Não existe hole punching/DCUtR implementado explicitamente pelo GoChat.
- Não há garantia de que uma conexão direta atravesse NAT, CGNAT, firewall ou rede corporativa.
- Não há criptografia de conteúdo implementada no envelope da aplicação, gestão de chaves por conversa, verificação de identidade pelo usuário, grupos, anexos ou sincronização entre dispositivos.
- Não há servidor remoto de histórico nem sincronização da base local.

## 2. Vocabulário e entidades

| Termo | Significado no projeto |
|---|---|
| Peer | Instância GoChat identificada por um PeerID derivado de chave Ed25519. |
| PeerID | Identificador libp2p estável derivado da identidade privada; não é o endereço IP. |
| Multiaddr | Endereço estruturado libp2p, por exemplo `/ip4/198.51.100.20/udp/4005/quic-v1/p2p/<PeerID>`. |
| Discovery | API HTTP e armazenamento Redis para heartbeats, convites, resolução de convites e descoberta do observer. |
| Address observer | Host libp2p auxiliar que observa IP/porta UDP de origem de uma conexão. Só troca metadados de endereço; não participa da conversa. |
| Contato | Peer que o usuário salvou localmente. Endereços são um cache e podem ficar obsoletos. |
| Envelope | Estrutura de mensagem de aplicação enviada dentro de um frame P2P. |
| ACK | Confirmação P2P de que o destinatário persistiu a mensagem localmente. Não significa leitura pelo usuário. |
| Outbox | Mensagens locais aguardando envio ou confirmação de entrega. |

## 3. Regras de negócio

### 3.1 Identidade

1. Na primeira execução, o GoChat gera um par de chaves Ed25519 com o gerador criptográfico do sistema.
2. A chave privada é serializada em PKCS#8 e armazenada em Base64 sem padding em `~/.gochat/identity.key`.
3. O diretório é criado com permissões `0700`; o arquivo, com `0600`.
4. Nas execuções seguintes, o mesmo arquivo é carregado. Portanto, a identidade/PeerID persiste enquanto esse arquivo for preservado.
5. O PeerID é derivado da chave privada pela libp2p. A chave pública e privada da identidade não são enviadas ao Discovery.
6. Apagar ou trocar `identity.key` cria outra identidade. Contatos existentes ainda apontam para o PeerID antigo.

### 3.2 Contatos e convites

1. Um peer online solicita ao Discovery um convite associado ao seu PeerID.
2. O código tem nove caracteres alfanuméricos, formatados como `XXX-XXX-XXX`; é gerado usando `crypto/rand`.
3. O convite expira após 10 minutos e só pode ser aceito uma vez. O Discovery consome o convite e registra o evento de aceitação atomicamente no Redis.
4. Para aceitar um convite, tanto o peer que o criou quanto o peer que o aceita precisam ter presença válida no Discovery. Se um deles estiver offline, a operação falha sem consumir o convite.
5. A aceitação retorna ao peer que digitou o código o PeerID e os endereços do criador; também deixa um evento temporário com o PeerID/endereço do aceitante para o criador.
6. O TUI do criador consulta aceitações a cada 2 segundos, somente até 10 minutos após a criação do convite. Após receber e persistir o contato, confirma o evento ao Discovery.
7. Os dois lados salvam o contato localmente, registram os multiaddrs no peerstore e tentam estabelecer uma conexão direta libp2p. Ambos abrem o chat do peer aceito; falha de conexão não encaminha mensagens pelo Discovery e é exibida no chat.
8. Um contato salvo permanece localmente mesmo que fique offline. Excluir um contato não apaga o histórico de mensagens.
9. Convites e contatos são locais a cada instalação. A aceitação de um único convite agora cria o contato em ambos os dispositivos, sem exigir que cada lado gere e digite um segundo convite.
10. O caminho `gochat invite` cria e imprime o convite, mas encerra o comando ao terminar. A presença fica online apenas enquanto o processo/host estiver ativo. Para o convite ser aceito e o peer continuar alcançável, o GoChat do criador precisa permanecer em execução; o TUI mantém o host ativo enquanto está aberto.
11. A opção de adicionar no TUI normaliza o código para maiúsculas e remove hífens. A API aceita código com ou sem hífens.

### 3.3 Presença e endereços

1. Ao iniciar o modo de conversa ou o comando de convite, o GoChat sobe um host libp2p QUIC/UDP e envia um heartbeat imediatamente.
2. A aplicação escolhe a primeira porta UDP disponível no intervalo `4001–4010` e escuta em `0.0.0.0` (IPv4), com protocolo QUIC v1. Também habilita AutoNATv2 da libp2p.
3. O heartbeat anuncia uma lista de até 16 multiaddrs candidatas, com o PeerID anexado.
4. Os grupos de candidatos considerados são: endereço observado pelo observer, endereços públicos confirmados por AutoNATv2, demais candidatos públicos e endereços locais conhecidos. Loopback e endereço não especificado são descartados; endereços privados podem ser publicados como candidatos locais.
5. O IP externo exibido por um site não basta para formar um multiaddr: o observer precisa observar também a porta UDP de origem. O resultado observado é apenas candidato, não prova de que outro peer conseguirá conectar.
6. O Discovery substitui a lista de endereços associada ao peer em cada heartbeat. O cliente tenta renovar a observação e o heartbeat a cada 2 minutos.
7. O Discovery considera a presença expirável: o registro de peer no Redis tem TTL de 5 minutos. Se os heartbeats cessarem, convites podem deixar de resolver o peer como online.
8. Endereços de contatos já salvos localmente não são atualizados automaticamente pela API a cada envio. Eles podem ficar obsoletos; re-adicionar/resolver o peer atualiza o cache local.

### 3.4 Envio e recebimento de mensagens

1. O remetente cria um ID aleatório de 128 bits (32 caracteres hexadecimais), timestamps UTC e preenche remetente, destinatário e versão do protocolo.
2. Antes de tentar a rede, salva a mensagem no histórico local e cria a entrada correspondente na outbox, em uma transação bbolt.
3. O conteúdo deve ser não vazio e ter no máximo 60 KiB. O frame JSON total não pode exceder 64 KiB.
4. O remetente abre um stream libp2p para o PeerID de destino usando `/gochat/2.0.0`, envia um frame de mensagem e aguarda um ACK com o mesmo ID.
5. O destinatário valida versão, PeerID remoto, destinatário, ID e timestamp. Persiste a mensagem uma única vez antes de emitir o ACK.
6. Retransmissões idênticas com o mesmo ID são tratadas como duplicata idempotente e podem receber novo ACK. Reutilização do mesmo ID com outro conteúdo/conversa é rejeitada.
7. O ACK significa “persistido pelo destinatário”, não “lido”, “visto” ou “aceito pelo usuário”.
8. Se a conexão falhar antes de o frame completo ser escrito, a mensagem permanece na outbox como `pending`. Se o frame foi escrito mas o ACK não chegou, o estado local vira `sent`. Com ACK válido, vira `delivered` e a entrada sai da outbox.
9. Erro de rede não apaga a mensagem local. A API `Send` devolve resultado que pode conter `DeliveryError` com a mensagem já enfileirada.
10. O app tenta novamente a outbox na inicialização e quando uma conexão com um peer é estabelecida. Um novo envio também tenta mensagens pendentes daquele peer. Não há um worker periódico independente de retries.
11. O timeout máximo para esperar ACK é 15 segundos, limitado também pelo deadline do contexto.
12. A conversa é ordenada por `CreatedAt`, com ID como desempate. O TUI carrega páginas de 50 mensagens e consulta histórico periodicamente enquanto está no fim da conversa.
13. Status e outbox são metadados locais; status de entrega não é transmitido no envelope P2P.

### 3.5 TUI

- A tela inicial lista contatos salvos localmente; presença visual não é uma verificação de conectividade em tempo real.
- `↑/↓` ou `j/k`: navegar contatos; `Enter`: abrir conversa.
- `a`: informar código e adicionar peer; `i`: criar convite; `q`: sair da tela inicial.
- Na conversa, digitar texto e pressionar `Enter` envia; `Backspace` apaga caractere; `Ctrl+U` limpa o campo; `PgUp/PgDn` navega histórico; `Esc` retorna à lista.
- A interface mostra autor, horário local e, para mensagens enviadas, `na fila`, `enviado · aguardando confirmação` ou `entregue`.
- Mensagens recebidas são exibidas como recebidas. Não existe estado de leitura.
- O histórico é atualizado a cada 2 segundos quando o usuário está no fim da conversa. Ao rolar para trás, a atualização de topo não substitui a página que está sendo lida.

## 4. Arquitetura lógica

```text
┌───────────────────── Peer A ─────────────────────┐
│ CLI/TUI → App → Messaging → libp2p QUIC/UDP       │
│                 ↘ bbolt local                    │
└────────────────────────┬─────────────────────────┘
                         │ heartbeat / invite / resolve / GET /observer (HTTP)
                         ▼
┌──────────────────────── Discovery ─────────────────────────┐
│ HTTP handlers → services → Redis                            │
│   Redis: peer:<PeerID> (endereços, TTL 5m)                   │
│          invite:<code> (PeerID, TTL 10m, consumo único)       │
│ Address observer: host libp2p auxiliar, somente metadados     │
└─────────────────────────────────────────────────────────────┘
                         ▲                          │
                         │ GET /observer + consulta │ endereço UDP visto
                         │                          ▼
┌───────────────────── Peer B ─────────────────────┐
│ CLI/TUI → App → Messaging → libp2p QUIC/UDP       │
│                 ↘ bbolt local                    │
└──────────────────────────────────────────────────┘

Mensagens: Peer A ───────── conexão P2P direta ─────────► Peer B
           Discovery e observer não recebem nem retransmitem conteúdo.
```

### Componentes Go

| Caminho | Responsabilidade |
|---|---|
| `cmd/gochat` | Carrega `.env.chat`, inicializa App, fecha o armazenamento e executa a CLI. |
| `internal/chat/cli` | Comandos, ciclo de vida do host, heartbeat, consulta do observer e ligação TUI/serviços. |
| `internal/chat/identity` | Criação, serialização e persistência da identidade Ed25519. |
| `internal/chat/client` | Cliente HTTP do Discovery, contratos JSON e tratamento de erros HTTP. |
| `internal/chat/p2p` | Host, multiaddrs, streams, frames, validação de mensagem e ACK. |
| `internal/chat/message` | Envelope, frame, ACK, versão e estados locais. |
| `internal/chat/messaging` | Outbox, envio, retries e atualização de status. |
| `internal/chat/storage` | Interface de repositório e implementação bbolt. |
| `internal/chat/tui` | Interface terminal e coordenação de contatos, histórico e envio. |
| `internal/discovery/handler` | Rotas HTTP, validação de entradas e respostas. |
| `internal/discovery/service` | Casos de uso de heartbeat, criação e resolução de convites. |
| `internal/discovery/repository` | Interface e implementação Redis. |
| `internal/discovery/observation` | Host libp2p observer e protocolo restrito de observação UDP/QUIC. |

## 5. Contrato HTTP do Discovery

Todas as rotas são montadas sem prefixo. Erros JSON têm `{ "status": <HTTP>, "message": "..." }`. Os handlers limitam corpos a 64 KiB, rejeitam campos JSON desconhecidos e rejeitam valores JSON adicionais após o objeto esperado.

| Rota | Método | Entrada / saída | Regras principais |
|---|---|---|---|
| `/heartbeat` | POST | Entrada `{ "peer_id": string, "addresses": [string] }`; sucesso `204 No Content`. | PeerID válido; de 1 a 16 multiaddrs sintaticamente válidos. Substitui a presença/endereço no Redis por 5 min. |
| `/invite` | POST | Entrada `{ "peer_id": string }`; sucesso `{ "code": "XXX-XXX-XXX" }`. | PeerID válido; código aleatório guardado por 10 min. |
| `/resolve` | POST | Entrada `{ "code": string, "peer_id": string }`; sucesso `{ "peer_id": string, "addresses": [...] }`. | Aceita código com ou sem hífens; consome o convite e cria o evento de aceitação atomicamente. Convite inválido/expirado ou algum peer offline retorna 404. |
| `/acceptances?peer_id=<PeerID>` | GET | Sucesso `{ "acceptances": [{ "invite_id": string, "peer_id": string, "addresses": [...], "accepted_at": time }] }`. | Retorna eventos ainda não confirmados; metadados expiram em até 10 min. O TUI consulta a cada 2 s até 10 min após emitir convite. |
| `/acceptances/ack` | POST | Entrada `{ "peer_id": string, "invite_ids": [string] }`; sucesso `204 No Content`. | Remove os eventos já persistidos localmente pelo criador; limite de 100 IDs por chamada. |
| `/observer` | GET | Sucesso `{ "address": "/ip4/.../udp/4001/quic-v1/p2p/<PeerID>" }`. | `503` se observer não estiver habilitado; não transporta mensagens. |
| Qualquer rota desconhecida | Qualquer | Erro JSON `404`. | — |

Método incorreto retorna `405`. Falhas internas de persistência/serviço retornam `500`. O endpoint `/observer` expõe metadados de conectividade e atualmente não exige autenticação.

### Persistência do Discovery em Redis

- `peer:<PeerID>` → JSON array de endereços, TTL 5 minutos.
- `invite:<código>` → PeerID, TTL 10 minutos; aceitação consome o convite e publica o evento em uma operação Lua atômica.
- `acceptances:<PeerID>` → hash temporário de aceitações contendo apenas ID do convite, PeerID, endereços e timestamp; eventos são removidos após confirmação do cliente e expiram em 10 minutos.
- Discovery não guarda histórico, payloads P2P nem conteúdo de chat; a fila de aceitação é exclusivamente sinalização de metadados.
- A implementação atual usa Redis sem autenticação no Compose. A porta 6379 não deve ser liberada publicamente por security group/firewall.

## 6. Protocolo P2P

### Transporte e endereçamento

- Libp2p com transporte QUIC v1 sobre UDP.
- O host GoChat escuta no primeiro UDP livre de `4001–4010`, em IPv4 `0.0.0.0`.
- Protocolo de mensagens: `/gochat/2.0.0`.
- O contato é identificado por PeerID. Multiaddrs resolvidos do convite são validados para carregar o mesmo PeerID e registrados no peerstore com TTL permanente até serem substituídos/limpos.
- A conexão QUIC/libp2p autentica a identidade da conexão e protege o transporte. Isso não substitui uma camada de segurança de aplicação com criptografia de mensagem, verificação/TOFU, rotação e gestão de chaves por conversa.

### Frames atuais

Os frames são JSON delimitado por newline, um frame por stream. O parser limita cada frame a 64 KiB, rejeita campos desconhecidos, JSON incompleto, valores extras e tipos não suportados.

```json
{"type":"message","message":{"id":"...","sender_peer_id":"...","recipient_peer_id":"...","created_at":"...","Content":"...","protocol_version":2}}
```

```json
{"type":"ack","ack":{"message_id":"..."}}
```

Nota de compatibilidade: o campo Go `Content` não tem tag JSON explícita no código atual; por isso o `encoding/json` usa o nome `Content` no wire format. Alterar esse nome exige considerar compatibilidade entre versões.

### Entrega

```text
Remetente: bbolt(message + outbox) → QUIC frame(message)
Destinatário: validar → bbolt(SaveMessageOnce) → QUIC frame(ACK)
Remetente: validar ACK/ID → bbolt(status=delivered, remover outbox)
```

Se persistir no destinatário falhar, nenhum ACK de sucesso é emitido. Se o ACK se perder depois da gravação, a retransmissão é deduplicada por ID e pode ser confirmada novamente.

## 7. Persistência local

### Localização e segurança básica

- Diretório compartilhado: `~/.gochat/` (`internal/chat/paths.go`).
- Identidade: `~/.gochat/identity.key`, arquivo `0600`.
- Banco: `~/.gochat/gochat.db`, diretório `0700`, arquivo `0600`.
- Banco bbolt local, sem servidor e sem sincronização entre computadores.
- Versão de schema atual: `1`. Versão mais nova que a suportada falha ao abrir; versão antiga exige migração explícita, que ainda não foi implementada.

### Buckets bbolt

| Bucket | Conteúdo |
|---|---|
| `peers` | PeerID, nome opcional, data de adição, último horário registrado e endereços cacheados. |
| `messages` | Mensagens por conversa, ordenadas por timestamp e ID; inclui envelope e status local. |
| `message_ids` | Índice global ID → chave da mensagem, para leitura rápida e deduplicação. |
| `outbox` | Mensagens ainda não confirmadas como entregues, ordenadas por timestamp/ID. |
| `metadata` | Versão do schema. |

`DeletePeer` apaga apenas o contato; histórico e outbox são mantidos. Mensagens não têm limpeza/expiração implementada.

## 8. Configuração

### Cliente GoChat

O executável exige `.env.chat` no diretório de trabalho porque `godotenv.Load(".env.chat")` falha se o arquivo não existir. O exemplo versionado documenta somente:

```env
DISCOVERY_URL=http://localhost:8080
```

Em produção, trocar pelo URL HTTP alcançável do Discovery. O multiaddr do observer **não** fica no `.env.chat`: o cliente o consulta por `GET /observer` através do `DISCOVERY_URL`.

### Serviço Discovery

`.env.discovery.example` documenta:

```env
DISCOVERY_PORT=8080
REDIS_ADDR=redis:6379
DISCOVERY_OBSERVER_ADVERTISE_ADDR=/ip4/<IP_PUBLICO_OU_DNS>/udp/4001/quic-v1
```

O endereço configurado para o observer não inclui `/p2p`; o servidor acrescenta seu PeerID persistente. `DISCOVERY_OBSERVER_ADVERTISE_ADDR` precisa usar IP4 ou DNS4 público e UDP 4001. Se estiver vazio, o serviço HTTP pode iniciar sem observer, mas `GET /observer` responde 503 e clientes não obtêm a observação externa.

O volume Docker `observer_data` contém a identidade privada do observer. Não removê-lo em deploys normais; removê-lo troca o PeerID e o endereço publicado.

### Portas e firewall

- Clientes GoChat → Discovery HTTP: TCP 8080 no deploy Compose atual (ou HTTPS se houver proxy configurado externamente).
- Cliente → address observer: UDP 4001 na EC2.
- Peer ↔ peer: UDP 4001–4010 nos hosts/roteadores envolvidos para tentar conexões diretas. NAT/firewall podem exigir encaminhamento/configuração local.
- Redis TCP 6379 deve ficar acessível somente pela rede interna da EC2/Compose, nunca aberto à internet.
- O projeto não solicita permissão ao sistema operacional para abrir portas e não configura automaticamente firewall/port forwarding do usuário.

## 9. Execução e deploy

### Desenvolvimento

1. Criar `.env.chat` a partir de `.env.chat.example` e configurar `DISCOVERY_URL`.
2. Garantir que o Discovery/Redis estejam acessíveis.
3. Executar `go run ./cmd/gochat` para abrir o TUI ou `go run ./cmd/gochat invite` para gerar código.
4. O TUI precisa ficar aberto para manter o host e heartbeat ativos.

Dependências de build Go estão em `go.mod` (Go 1.27.1 e módulos listados no arquivo). A versão do host depende do ambiente Go instalado. O Gochat standalone ainda depende da existência de `.env.chat`.

### Discovery com Docker Compose

- `docker-compose.yml` sobe `redis` e `discovery`.
- O Dockerfile de Discovery é multi-stage: compila somente `cmd/discovery` e `internal/discovery` com Go 1.27; a imagem de runtime é Debian slim. Não compila o binário do chat.
- Compose publica TCP 8080 e UDP 4001. Redis está mapeado em 6379 no host, então o firewall/security group deve restringir essa porta.
- Discovery depende do serviço Redis, mas `depends_on` não garante prontidão lógica; a aplicação/infra deve tolerar ou tratar Redis indisponível.

### Workflow AWS

O workflow `.github/workflows/deploy-discovery.yml` dispara em push para `master`, usa credenciais AWS por OIDC e SSM na EC2. Espera variáveis de repositório:

- `AWS_ROLE_TO_ASSUME`
- `AWS_REGION`
- `EC2_INSTANCE_ID`
- `EC2_APP_DIR`
- `DISCOVERY_OBSERVER_ADVERTISE_ADDR`

Faz preflight de Git, Docker, Docker Compose, Buildx (mínimo 0.17.0), builder `gochat-builder` e diretório/repositório; atualiza checkout de `master`, cria `.env.discovery` com permissões restritas, valida Compose, constrói somente `discovery`, recria o serviço sem rebuild no `up` e verifica `REDIS_ADDR`/`PONG` do Redis. O workflow não instala as dependências da EC2; pré-requisitos faltantes interrompem o deploy.

## 10. Erros e comportamento de falha

- Falha ao abrir/validar `.env.chat`, identidade, client Discovery ou bbolt encerra a inicialização.
- Falha na consulta/conexão do observer é logada; o GoChat continua com candidatos restantes. A ausência do endereço observado pode reduzir a chance de conexão externa.
- Falha no heartbeat inicial impede iniciar presença; heartbeats subsequentes falhos são logados e o processo continua.
- Falha ao resolver convite apresenta erro; convites expirados/usados e peer sem presença resultam em 404 na API.
- Endereços inválidos ou com PeerID diferente do contato são rejeitados antes de salvar contato/registrar no peerstore.
- Falha de conexão/envio deixa mensagem na outbox. UI mostra que está na fila ou aguardando entrega.
- Fechar a aplicação fecha o host e interrompe presença. Os dados locais permanecem.
- Se o peer estiver offline, a entrega só é tentada novamente em eventos de retry atuais (inicialização, nova conexão ou novo envio); não há promessa de retry contínuo após qualquer mudança de rede.

## 11. Segurança, privacidade e confiança

### Garantias atuais

- Identidade local Ed25519 persistente e com permissões restritas no filesystem.
- Conexões libp2p/QUIC autenticam peers e criptografam o transporte.
- Validação do PeerID remoto contra o remetente declarado no envelope e do destinatário contra a identidade local.
- Limites de tamanho, versão de protocolo, deduplicação e persistência antes de ACK.
- Discovery/observer não recebem conteúdo de mensagens.

### Lacunas conhecidas

- Não há criptografia de ponta a ponta de nível de aplicação, rotação de chaves, gestão de dispositivos, UI de confiança/identidade nem proteção criptográfica do histórico em repouso.
- API Discovery não tem autenticação de conta. Endpoints de invite/heartbeat aceitam PeerIDs declarados pelo cliente; validar formato não prova posse da chave privada.
- Convite é um segredo bearer de curta duração e seu uso não depende de uma conta autenticada.
- `GET /observer` e API usam HTTP no deploy atual, sem TLS configurado no próprio serviço. O endereço e PeerID do observer são dados públicos de conectividade.
- Endereços IP/multiaddrs são publicados no Discovery e podem expor metadados de rede.
- O observer é um ponto público de metadados; não é relay, mas depende de UDP 4001, endereço público estável e volume de identidade persistente.
- Contatos persistem endereços em cache sem validade/atualização automática. Endereços antigos podem causar falha ou atraso de discagem.
- Redis no Compose não configura senha/TLS; exposição deve ser bloqueada na rede.

## 12. Limitações técnicas e decisões futuras

1. **Conexão direta apenas:** a tentativa usa endereços no peerstore e `host.NewStream`. Se NAT/firewall impedir UDP entrante, a mensagem pode não entregar.
2. **Hole punching ainda não implementado:** AutoNATv2/observação ajudam a diagnosticar e descobrir candidatos, mas não equivalem a hole punching.
3. **Sem relay deliberadamente:** se a futura tentativa direta e hole punching falhar, a decisão de produto vigente é devolver falha; não encaminhar conteúdo pelo Discovery nem adicionar relay como fallback.
4. **Discovery só para controle:** não introduzir endpoint, bucket Redis ou protocolo de mensagem no Discovery para encaminhar chat.
5. **Sem sincronização:** identidade e histórico são locais. Copiar apenas `.env.chat` para outro computador não compartilha PeerID; compartilhar `identity.key` mudaria o modelo de segurança e não é recomendado sem decisão explícita.
6. **Presença e convite:** o comando `invite` encerra seu processo; avaliar um comando de presença persistente ou recomendar TUI aberto para garantir alcançabilidade enquanto o convite é usado.
7. **Atualização de endereços:** avaliar refresh de endereços de contatos já conhecidos e expiração/cache no peerstore, pois o endereço pode mudar.
8. **Interface da CLI:** `add`, `friends` e `chat` ainda são comandos-placeholder; o fluxo funcional de convite/adicionar/conversar está no TUI.
9. **Estados do TUI:** o horário mostrado como “visto via Discovery” ao adicionar é o horário local da resolução, não uma confirmação contínua de online.
10. **Cobertura:** testes automatizados abrangentes para storage, API, protocolo e integração entre dois peers continuam sendo trabalho futuro.

## 13. Critérios de aceitação funcionais (alvo)

Os critérios abaixo descrevem como validar o comportamento pretendido. A passagem deles depende de conectividade de rede real e não é inferida apenas pelo sucesso de build.

1. Duas instalações geram PeerIDs distintos; reiniciar cada instalação preserva seu próprio PeerID.
2. Com ambos online, convite ainda válido é aceito uma única vez, devolve PeerID + multiaddrs do criador e cria um evento para o criador.
3. O peer que aceita e o peer que criou o convite salvam um ao outro localmente, tentam uma conexão direta e abrem o chat; o TUI do criador descobre a aceitação em até aproximadamente 2 s enquanto o convite estiver válido.
4. O TUI para de consultar aceitações após 10 minutos da criação do convite; eventos persistem temporariamente até confirmação ou expiração.
5. Uma mensagem válida é gravada localmente antes do envio, chega ao destinatário, aparece no histórico e só é considerada `delivered` no remetente após ACK.
6. Retransmitir o mesmo ID não duplica a mensagem; ACK só é enviado depois da persistência do destinatário.
7. Uma falha de conexão mantém a mensagem local na outbox e apresenta estado/erro compreensível; não envia conteúdo ao Discovery.
8. `GET /observer` retorna o multiaddr completo do observer quando configurado e 503 quando não configurado.
9. Mudança de rede, porta bloqueada, NAT incompatível ou observer indisponível deve ser comunicada como condição de rede, sem alegar garantia de conexão.

## 14. Fontes de verdade no repositório

- `cmd/gochat/main.go`, `internal/chat/cli/`
- `internal/chat/identity/`, `internal/chat/paths.go`
- `internal/chat/client/`, `internal/chat/message/`
- `internal/chat/p2p/`, `internal/chat/messaging/`
- `internal/chat/storage/`, `internal/chat/tui/`
- `internal/discovery/`, `cmd/discovery/`
- `.env.chat.example`, `.env.discovery.example`, `docker-compose.yml`
- `.github/workflows/deploy-discovery.yml`
- `docs/public-address-observer.md`
