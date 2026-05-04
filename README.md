# Infraestrutura de Comunicação (GRUPO 1)

Projeto da disciplina para simular, na camada de aplicação, um transporte confiável cliente-servidor sobre sockets. As funcionalidades presentes no momento são:

- Estabelecer conexão entre cliente e servidor via socket (localhost ou IP);
- Realizar handshake inicial de três vias (SYN → SYN+ACK → ACK);
- Negociar modo de operação (Go-Back-N ou Repetição Seletiva), tamanho máximo da mensagem e tamanho da janela;
- Enviar mensagens de texto fragmentadas em segmentos de no máximo 4 caracteres de carga útil;
- Controle de fluxo por janela deslizante (tamanho de 1 a 5, definido pelo servidor);
- Número de sequência e reconhecimento positivo (ACK) por segmento;
- Reassemblagem ordenada da mensagem completa no servidor.

## Arquitetura

O projeto está dividido em três partes:

- `cmd/server`: ponto de entrada do servidor, responsável por escutar na porta `8080` e aceitar conexões.
- `cmd/client`: ponto de entrada do cliente, responsável por iniciar a conexão com o servidor.
- `internal/tcp`: implementação da camada de aplicação, incluindo estados da conexão, segmentos, transporte utilizando a biblioteca padrão do Go e os fluxos de handshake, envio e recebimento.

## Protocolo de Aplicação

### Handshake (3 vias)

1. O cliente envia `SYN` com `protocol` (gbn ou sr) e `maxChars` (mínimo 30).
2. O servidor responde com `SYN+ACK`, confirmando os parâmetros e informando o `windowSize`.
3. O cliente envia `ACK` final e a conexão passa para `ESTABLISHED`.

### Estrutura do Segmento

Cada segmento contém:

- **Header**: flags (`SYN`, `ACK`, `FIN`), número de sequência (`Seq`), número de reconhecimento (`Ack`), tamanho da janela (`WindowSize`).
- **Message**: texto (máximo 4 caracteres por segmento), protocolo e tamanho máximo de caracteres.

### Envio de Mensagens (Cliente)

1. O texto é fragmentado em segmentos de até 4 caracteres, cada um com seu número de sequência.
2. Os segmentos são enviados dentro da janela deslizante (paralelismo controlado pelo `windowSize`).
3. O cliente aguarda ACKs do servidor e avança a janela conforme o protocolo escolhido.
4. Ao final da mensagem, o cliente envia um segmento com flag `FIN`.

### Recebimento de Mensagens (Servidor)

1. O servidor recebe cada segmento, imprime seus metadados (texto, SEQ, endereço do remetente) e envia um ACK.
2. Ao receber o `FIN`, o servidor reassembla e exibe a mensagem completa.

### Modos de Operação

- **Go-Back-N (gbn)**: o receptor aceita apenas segmentos em ordem. Segmentos fora de ordem são descartados e o ACK do último segmento em ordem é reenviado (ACK cumulativo). O emissor avança a janela com base no ACK cumulativo.
- **Repetição Seletiva (sr)**: o receptor aceita e armazena segmentos fora de ordem, enviando ACK individual para cada um. O emissor marca ACKs individuais e avança a janela apenas quando o segmento de menor sequência não confirmado recebe seu ACK.

## Como Executar

Pré-requisito: Go instalado (https://go.dev/).

Em um terminal, inicie o servidor:

```bash
go run ./cmd/server
```

O servidor solicitará o tamanho da janela (1-5, padrão 5).

Em outro terminal, execute o cliente:

```bash
go run ./cmd/client
```

O cliente solicitará:

- **IP do servidor**: endereço do servidor (padrão `localhost:8080`).
- **Protocolo**: `gbn` (Go-Back-N) ou `sr` (Repetição Seletiva), padrão `gbn`.
- **Máximo de caracteres**: limite máximo de caracteres por mensagem (mínimo 30, padrão 30).

Após a conexão, digite mensagens no prompt `>` para enviar ao servidor. Digite `exit` para encerrar.

## Utilização de Inteligência Artificial

A inteligência artificial foi utilizada neste projeto para gerar a documentação (com exceção dessa parte que estou escrevendo) e ajuda em como o protocolo de fato funciona, qual a melhor abordagem de arquitetura, e alguns poucos bugs.
