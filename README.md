# Infraestrutura de Comunicação - (GRUPO 1)

Relatório e manual de utilização do projeto da disciplina de Infraestrutura de
Comunicação. O sistema implementa, na camada de aplicação, um transporte
confiável de dados entre cliente e servidor sobre sockets TCP da biblioteca padrão
do Go. A confiabilidade é simulada no protocolo de aplicação para permitir a
observação dos mecanismos estudados em transporte confiável: soma de
verificação, temporizador, número de sequência, ACK, NAK, janela e paralelismo.

## 1. Resumo

O projeto consiste em uma aplicação cliente-servidor capaz de enviar mensagens de
texto de forma fragmentada. Cada fragmento da camada de aplicação carrega no
máximo 4 caracteres de carga útil. O cliente escolhe o modo de operação
(`gbn` para Go-Back-N ou `sr` para Repetição Seletiva) e o tamanho máximo da
mensagem no início da comunicação. O servidor define o tamanho da janela, entre 1
e 5, com valor padrão 5.

Na entrega final, o sistema também permite simular, de forma determinística no
cliente, perdas e falhas de integridade em segmentos. O servidor verifica a soma de
verificação, envia confirmações positivas (`ACK`) para segmentos válidos e
confirmações negativas (`NAK`) para segmentos corrompidos. O cliente utiliza um
temporizador para retransmitir segmentos não confirmados.

## 2. Objetivo

Desenvolver uma aplicação cliente-servidor capaz de fornecer, na camada de
aplicação, transporte confiável de dados mesmo considerando um canal lógico com
perdas e erros simulados.

Os objetivos específicos do trabalho são:

| Objetivo | Implementação no projeto |
| --- | --- |
| Conectar cliente e servidor via socket | Cliente usa `net.Dial`; servidor usa `net.Listen` |
| Permitir conexão por `localhost` ou IP | Cliente recebe endereço no início da execução |
| Negociar parâmetros iniciais | Handshake `SYN`, `SYN+ACK`, `ACK` |
| Fragmentar mensagens | Segmentos com até 4 caracteres de carga útil |
| Controlar envio por janela | Janela definida pelo servidor, de 1 a 5 |
| Confirmar recebimento | `ACK` individual ou cumulativo |
| Detectar erro | Campo `Checksum` |
| Reagir a erro | `NAK` e retransmissão |
| Reagir a perda | Temporizador e retransmissão |
| Suportar Go-Back-N e Repetição Seletiva | Modo escolhido pelo cliente |

## 3. Requisitos da Especificação

| Item da especificação | Situação | Onde aparece |
| --- | --- | --- |
| Cliente conecta ao servidor por `localhost` ou IP | Implementado | Prompt `Server ip` no cliente |
| Comunicação por sockets | Implementado | `internal/tcp/transport.go` |
| Protocolo de aplicação descrito | Implementado | Seções 5 e 6 deste relatório |
| Tamanho máximo da mensagem definido no início | Implementado | Campo `maxChars` no handshake |
| Tamanho mínimo de 30 caracteres | Implementado | Validação em `Dial` |
| Pacotes com no máximo 4 caracteres | Implementado | Fragmentação no envio |
| Metadados dos segmentos impressos no servidor | Implementado | Logs do servidor |
| Mensagem completa exibida no servidor | Implementado | Reassemblagem ao receber `FIN` |
| Metadados das confirmações impressos no cliente | Implementado | Logs de `ACK` e `NAK` |
| Soma de verificação | Implementado | `Checksum` com CRC32 |
| Temporizador | Implementado | Timeout no cliente |
| Número de sequência | Implementado | Campo `Seq` |
| Reconhecimento positivo | Implementado | Flag `ACK` |
| Reconhecimento negativo | Implementado | Flag `NAK` |
| Janela e paralelismo | Implementado | `WindowSize` |
| Janela de 1 a 5 | Implementado | Validação no servidor |
| Janela determinada pelo servidor | Implementado | Informada no `SYN+ACK` |
| Valor inicial/padrão da janela igual a 5 | Implementado | Entrada padrão no servidor |
| Simulação de perda | Implementado | Lista de segmentos a perder |
| Simulação de falha de integridade | Implementado | Lista de segmentos a corromper |
| Erro determinístico definido no cliente | Implementado | Usuário escolhe o número do segmento |
| Envio de pacote isolado | Implementado | Mensagem de até 4 caracteres gera 1 segmento |
| Envio em lote | Implementado | Mensagem maior gera múltiplos segmentos |
| Confirmação individual | Implementado | Repetição Seletiva |
| Confirmação em grupo/janela | Implementado | Go-Back-N com ACK cumulativo |
| Código e manual de uso | Implementado | Este repositório e este README |

## 4. Arquitetura

O projeto é dividido em três partes principais:

| Diretório/arquivo | Responsabilidade |
| --- | --- |
| `cmd/server/server.go` | Inicia o servidor, configura a janela e aceita conexões |
| `cmd/client/client.go` | Inicia o cliente, coleta parâmetros e envia mensagens |
| `internal/tcp/transport.go` | Encapsula o uso de sockets TCP e serialização `gob` |
| `internal/tcp/tcp.go` | Define tipos do protocolo: estado, cabeçalho, mensagem, segmento e falhas |
| `internal/tcp/connection.go` | Implementa handshake, envio, recebimento, ACK/NAK, retransmissão e reassemblagem |
| `internal/tcp/*_test.go` | Testes automatizados para conexão, envio, recebimento e falhas |

Mesmo usando sockets TCP reais para conectar os processos, os mecanismos de
confiabilidade pedidos no trabalho são implementados no protocolo de aplicação.
Assim, perda e corrupção são simuladas acima da camada de transporte real.

## 5. Protocolo de Aplicação

### 5.1 Estados

| Estado | Significado |
| --- | --- |
| `CLOSED` | Conexão ainda não estabelecida ou encerrada |
| `LISTEN` | Servidor aguardando pedido de conexão |
| `SYN_SENT` | Cliente enviou `SYN` e aguarda resposta |
| `SYN_RECEIVED` | Servidor recebeu `SYN` e enviou `SYN+ACK` |
| `ESTABLISHED` | Cliente e servidor podem trocar dados |

### 5.2 Estrutura do Segmento

Cada pacote da camada de aplicação é representado por um `Segment`.

| Campo | Subcampo | Descrição |
| --- | --- | --- |
| `Header` | `Flags.Syn` | Solicita início de conexão |
| `Header` | `Flags.Ack` | Confirma recebimento correto |
| `Header` | `Flags.Nak` | Informa erro em segmento recebido |
| `Header` | `Flags.Fin` | Indica fim da mensagem atual |
| `Header` | `WindowSize` | Tamanho da janela definido pelo servidor |
| `Header` | `Seq` | Número de sequência do segmento |
| `Header` | `Ack` | Número de sequência confirmado |
| `Message` | `Text` | Carga útil, com no máximo 4 caracteres |
| `Message` | `Protocol` | Modo `gbn` ou `sr` |
| `Message` | `MaxChars` | Tamanho máximo da mensagem |
| `Checksum` | - | Soma de verificação do segmento |

### 5.3 Mensagens do Protocolo

| Mensagem | Enviada por | Flags | Campos relevantes | Resposta esperada |
| --- | --- | --- | --- | --- |
| Pedido de conexão | Cliente | `SYN` | `Seq`, `Protocol`, `MaxChars` | `SYN+ACK` |
| Aceite de conexão | Servidor | `SYN`, `ACK` | `Seq`, `Ack`, `WindowSize` | `ACK` final |
| Confirmação final | Cliente | `ACK` | `Seq`, `Ack` | Conexão estabelecida |
| Dados | Cliente | nenhuma flag especial | `Seq`, `Text`, `Checksum` | `ACK` ou `NAK` |
| Confirmação positiva | Servidor | `ACK` | `Ack` | Cliente avança janela |
| Confirmação negativa | Servidor | `NAK` | `Ack` | Cliente retransmite |
| Fim da mensagem | Cliente | `FIN` | - | Servidor monta e imprime mensagem |

### 5.4 Handshake Inicial

O handshake inicial segue uma estrutura inspirada no estabelecimento de conexão em
três vias:

| Etapa | Origem | Destino | Segmento | Finalidade |
| --- | --- | --- | --- | --- |
| 1 | Cliente | Servidor | `SYN` | Informa modo (`gbn`/`sr`) e `maxChars` |
| 2 | Servidor | Cliente | `SYN+ACK` | Confirma parâmetros e informa `windowSize` |
| 3 | Cliente | Servidor | `ACK` | Confirma recebimento e entra em `ESTABLISHED` |

Após essas três etapas, cliente e servidor consideram a conexão estabelecida.

## 6. Funcionamento do Envio e Recebimento

### 6.1 Fragmentação

Antes do envio, a mensagem digitada pelo usuário é dividida em segmentos de até 4
caracteres. Exemplo:

| Mensagem | Segmento | Carga útil |
| --- | ---: | --- |
| `Hello, World!` | 1 | `Hell` |
| `Hello, World!` | 2 | `o, W` |
| `Hello, World!` | 3 | `orld` |
| `Hello, World!` | 4 | `!` |

Cada segmento recebe um número de sequência próprio. O cliente envia segmentos
respeitando o limite de janela definido pelo servidor.

### 6.2 Recebimento no Servidor

Ao receber um segmento de dados, o servidor:

1. imprime os metadados do segmento;
2. verifica a soma de verificação;
3. envia `NAK` se detectar corrupção;
4. processa o segmento conforme `gbn` ou `sr`;
5. envia `ACK` quando o segmento é aceito;
6. ao receber `FIN`, reordena os fragmentos e imprime a mensagem completa.

### 6.3 Confirmações no Cliente

O cliente imprime as confirmações à medida que chegam:

| Confirmação | Significado | Ação do cliente |
| --- | --- | --- |
| `ACK n` | Segmento `n` recebido corretamente | Marca segmento como confirmado |
| `NAK n` | Segmento `n` chegou corrompido | Retransmite conforme o protocolo |
| Timeout | Nenhuma confirmação chegou a tempo | Retransmite segmentos pendentes |

## 7. Modos de Operação

### 7.1 Go-Back-N (`gbn`)

No modo Go-Back-N, o servidor aceita apenas o próximo segmento esperado.
Segmentos fora de ordem são descartados. As confirmações são cumulativas.

| Situação | Comportamento |
| --- | --- |
| Segmento esperado chega íntegro | Servidor armazena e envia `ACK` |
| Segmento fora de ordem chega | Servidor descarta e repete ACK do último correto |
| Segmento chega corrompido | Servidor descarta e envia `NAK` |
| Segmento se perde | Cliente detecta por timeout |
| Retransmissão | Cliente reenvia a partir do segmento faltante |

Exemplo com janela 5 e perda do segmento 2:

| Etapa | Evento |
| --- | --- |
| 1 | Cliente envia segmentos 1, 2, 3, 4 e 5 |
| 2 | Segmento 2 é perdido pela simulação |
| 3 | Servidor aceita 1 e descarta 3, 4 e 5 por estarem fora de ordem |
| 4 | Cliente não recebe confirmação de 2 |
| 5 | Temporizador expira |
| 6 | Cliente retransmite a partir do segmento 2 |

### 7.2 Repetição Seletiva (`sr`)

No modo Repetição Seletiva, o servidor aceita segmentos válidos mesmo que cheguem
fora de ordem. Segmentos fora de ordem ficam armazenados até que os anteriores
cheguem.

| Situação | Comportamento |
| --- | --- |
| Segmento íntegro chega | Servidor armazena e envia `ACK` individual |
| Segmento fora de ordem chega | Servidor armazena e envia `ACK` individual |
| Segmento chega corrompido | Servidor descarta e envia `NAK` |
| Segmento se perde | Cliente detecta ausência de `ACK` por timeout |
| Retransmissão | Cliente reenvia apenas segmento pendente |

Exemplo com janela 5 e corrupção do segmento 2:

| Etapa | Evento |
| --- | --- |
| 1 | Cliente envia segmentos 1, 2, 3, 4 e 5 |
| 2 | Segmento 2 é corrompido pela simulação |
| 3 | Servidor aceita 1, 3, 4 e 5 |
| 4 | Servidor envia `NAK 2` para o segmento corrompido |
| 5 | Cliente retransmite apenas o segmento 2 |
| 6 | Servidor reordena os segmentos e monta a mensagem correta |

## 8. Simulação de Erros e Perdas

A simulação é configurada no cliente antes do envio. O usuário informa os números
dos segmentos da mensagem que devem sofrer perda ou corrupção. A contagem começa
em 1, de acordo com a posição do segmento dentro da mensagem.

| Tipo de falha | Como configurar | O que acontece |
| --- | --- | --- |
| Perda | Informar segmento em `Segments to drop once` | Cliente não envia esse segmento na primeira tentativa |
| Corrupção | Informar segmento em `Segments to corrupt once` | Cliente altera a carga útil após calcular o checksum |
| Sem falha | Deixar campos vazios | Mensagem segue normalmente |

As falhas são determinísticas: se o usuário escolher o segmento 2, o segmento 2
será afetado. A falha é aplicada uma vez para permitir que a retransmissão envie o
segmento correto depois.

## 9. Manual de Utilização

### 9.1 Pré-requisitos

- Go instalado.
- Dois terminais: um para o servidor e outro para o cliente.

### 9.2 Executar o Servidor

No primeiro terminal:

```bash
go run ./cmd/server
```

O servidor solicitará:

```text
Window size (1-5 default is 5):
```

Valores aceitos:

| Entrada | Resultado |
| --- | --- |
| vazio | Usa janela 5 |
| `1` a `5` | Usa valor informado |
| maior que `5` | Ajusta para 5 |

### 9.3 Executar o Cliente

No segundo terminal:

```bash
go run ./cmd/client
```

O cliente solicitará:

| Prompt | Exemplo | Descrição |
| --- | --- | --- |
| `Server ip` | `localhost:8080` | Endereço do servidor |
| `Protocol` | `gbn` ou `sr` | Modo de operação |
| `Max chars` | `30` | Limite máximo por mensagem |
| `Segments to drop once` | `2` ou `2,4` | Segmentos que serão perdidos |
| `Segments to corrupt once` | `3` | Segmentos que serão corrompidos |

Depois do handshake, o cliente mostra o prompt:

```text
>
```

Digite uma mensagem para enviar. Digite `exit` para encerrar o cliente.

### 9.4 Exemplos de Execução

#### Exemplo 1: envio sem falhas

Servidor:

```text
Window size (1-5 default is 5): 5
```

Cliente:

```text
Server ip (default is localhost:8080):
Protocol (gbn/sr default is gbn): gbn
Max chars (min 30 default is 30): 30
Segments to drop once (comma-separated, default none):
Segments to corrupt once (comma-separated, default none):
> Hello, World!
```

Resultado esperado: o servidor imprime os segmentos recebidos e depois a mensagem
completa `Hello, World!`.

#### Exemplo 2: perda com Go-Back-N

Cliente:

```text
Protocol (gbn/sr default is gbn): gbn
Segments to drop once (comma-separated, default none): 2
Segments to corrupt once (comma-separated, default none):
> Hello, World!
```

Resultado esperado: o cliente simula perda do segmento 2, aguarda timeout e
retransmite a partir do segmento 2.

#### Exemplo 3: corrupção com Repetição Seletiva

Cliente:

```text
Protocol (gbn/sr default is gbn): sr
Segments to drop once (comma-separated, default none):
Segments to corrupt once (comma-separated, default none): 2
> Hello, World!
```

Resultado esperado: o servidor detecta checksum inválido no segmento 2, envia
`NAK 2`, e o cliente retransmite apenas o segmento 2.

## 10. Evidências de Transporte Confiável

| Característica exigida | Como verificar na execução |
| --- | --- |
| Soma de verificação | Simular corrupção e observar envio de `NAK` |
| Temporizador | Simular perda e observar timeout no cliente |
| Número de sequência | Logs mostram `SEQ` por segmento |
| ACK positivo | Cliente imprime `ACK` recebido |
| ACK negativo | Cliente imprime `NAK` recebido |
| Janela/paralelismo | Usar janela maior que 1 e observar vários segmentos enviados antes das confirmações |
| Go-Back-N | Perder segmento em `gbn` e observar retransmissão a partir dele |
| Repetição Seletiva | Corromper segmento em `sr` e observar retransmissão isolada |

## 11. Testes Automatizados

O projeto inclui testes automatizados para os principais fluxos:

| Teste | Cobertura |
| --- | --- |
| `TestDial` | Handshake do cliente |
| `TestListener` | Handshake do servidor |
| `TestSend` | Envio normal, erro de conexão, perda e corrupção |
| `TestReceive` | Recebimento normal, erro de conexão e `NAK` por corrupção |

Para executar:

```bash
go test ./...
```

Para verificar condições de corrida:

```bash
go test -race ./...
```

## 12. Conclusão

A aplicação implementa um protocolo de transporte confiável na camada de aplicação,
com conexão cliente-servidor, handshake inicial, fragmentação de mensagens,
controle por janela, confirmações positivas e negativas, soma de verificação,
temporizador e retransmissão. A entrega final permite simular perdas e falhas de
integridade de maneira determinística, mostrando o comportamento correto dos
processos nos modos Go-Back-N e Repetição Seletiva.

## Utilização de Inteligência Artificial

A inteligência artificial foi utilizada neste projeto para gerar a documentação (com exceção dessa parte que estou escrevendo) e ajuda em como o protocolo de fato funciona, qual a melhor abordagem de arquitetura, e alguns poucos bugs. Especificamente na entrega final, teve um papel fundamental na construção do código, por ser uma entrega mais díficil do ponto de vista técnico. Em geral, para 90% do projeto a IA funcionou apenas para tirar dúvidas, resolver bugs, e explicar como os algorítimos que o Kurose ensina funcionam.
