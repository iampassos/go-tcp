# Infraestrutura de Comunicação (GRUPO 1)

Projeto da disciplina para simular, na camada de aplicação, um transporte confiável cliente-servidor sobre sockets. As funcionalidades presentes no momento são:

- estabelecer conexão entre cliente e servidor via socket;
- realizar o handshake inicial;
- trocar pelo menos o modo de operação e o tamanho máximo da mensagem.

## Arquitetura

O projeto está dividido em três partes:

- `cmd/server`: ponto de entrada do servidor, responsável por escutar na porta `8080` e aceitar conexões.
- `cmd/client`: ponto de entrada do cliente, responsável por iniciar a conexão com o servidor.
- `internal/tcp`: implementação da camada de aplicação, incluindo estados da conexão, segmentos, transporte utilizando a biblitoteca padrão do Go e o fluxo de handshake.

### Handshake atual

O protocolo implementado segue um fluxo simples em três etapas:

1. O cliente envia `SYN` com `protocol=gbn` e `maxChars=30`.
2. O servidor responde com `SYN+ACK`, confirmando os parâmetros iniciais.
3. O cliente envia `ACK` final e a conexão passa para `ESTABLISHED`.

As mensagens são serializadas com `gob` e transportadas sobre TCP usando a biblioteca padrão do Go.

## Como executar

Pré-requisito: Go instalado `https://go.dev/`.

Em um terminal, inicie o servidor:

```bash
go run ./cmd/server
```

Em outro terminal, execute o cliente:

```bash
go run ./cmd/client
```

## Utilização de Inteligência Artificial

A inteligência artificial foi utilizada neste projeto apenas para gerar a documentação (com exceção dessa parte que estou escrevendo) e ajuda conceitual de como o protocolo de fato funciona e qual a melhor abordagem de arquitetura.
