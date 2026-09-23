# Rate Limiter

Projeto desenvolvido como desafio técnico do módulo Desafios Técnicos do curso Go Expert da Full Cycle.

O sistema funciona como um middleware HTTP para controlar o fluxo de requisições de um serviço web. A limitação pode ser feita pelo endereço IP do cliente ou pelo token enviado no header `API_KEY`. Os contadores e bloqueios são armazenados no Redis.

## API

`GET /`

Quando a requisição é permitida, a aplicação retorna:

```text
ok
```

Exemplo sem token:

```bash
curl -i http://localhost:8080/
```

Exemplo com token:

```bash
curl -i -H 'API_KEY: meu-token' http://localhost:8080/
```

Quando o limite é excedido, a aplicação retorna `429 Too Many Requests` com exatamente este corpo, sem newline:

```text
you have reached the maximum number of requests or actions allowed within a certain time frame
```

## Regras de limitação

- Sem `API_KEY`, a requisição é limitada pela chave `ip:<endereço>` usando `RATE_LIMIT_IP`.
- Com `API_KEY` não vazio, a requisição é limitada pela chave `token:<token>` usando `RATE_LIMIT_TOKEN`.
- O limite do token tem precedência sobre o limite do IP.
- Uma requisição com token não incrementa o contador do IP.
- A janela de contagem é de um segundo.
- A primeira requisição que excede o limite cria o bloqueio da chave infratora.
- Durante o bloqueio, novas requisições da mesma chave retornam `429`.

## Configuração

Copie o arquivo de exemplo:

```bash
cp .env.example .env
```

Variáveis disponíveis:

| Variável | Descrição |
| --- | --- |
| `PORT` | Porta HTTP da aplicação. |
| `RATE_LIMIT_IP` | Máximo de requisições por segundo para cada IP. |
| `RATE_LIMIT_TOKEN` | Máximo de requisições por segundo para cada token não vazio. |
| `BLOCK_TIME` | Tempo de bloqueio após exceder o limite, por exemplo `5m`. |
| `REDIS_ADDR` | Endereço do Redis, por exemplo `redis:6379`. |
| `REDIS_PASSWORD` | Senha do Redis, se houver. |
| `REDIS_DB` | Número do database Redis. |

Variáveis de ambiente têm precedência sobre os valores do arquivo `.env`.

## Execução

Suba a aplicação e o Redis:

```bash
docker compose up --build
```

A aplicação ficará disponível em `http://localhost:8080`.

## Testes

Execute os testes pelo Docker Compose:

```bash
docker compose run --rm app go test ./...
```

Execute também o detector de condições de concorrência:

```bash
docker compose run --rm app go test -race ./...
```

Os testes de integração usam o Redis iniciado pelo Compose.

## Estratégia de persistência

A persistência é definida pela interface:

```go
type RateLimiterStorage interface {
    Allow(ctx context.Context, key string, limit int, blockTime time.Duration) (bool, error)
}
```

`internal/redis.Storage` é a implementação usada pela aplicação. Para utilizar outra estratégia, implemente essa interface e injete a nova implementação no construtor do rate limiter.

## Estrutura

- `cmd/server`: inicialização do servidor HTTP e conexão com o Redis;
- `internal/config`: leitura das variáveis de ambiente e do arquivo `.env`;
- `internal/ratelimiter`: regra de limitação e interface de persistência;
- `internal/redis`: estratégia de persistência no Redis;
- `internal/web`: middleware HTTP e seleção da chave por IP ou token.
