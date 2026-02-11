# Lab Rate Limiter

Rate limiter em Go que limita requisições por segundo com base em endereço IP ou token de acesso (`API_KEY`).

## Como funciona

O rate limiter atua como middleware HTTP. Para cada requisição:

1. Verifica se o IP/token está bloqueado
2. Incrementa o contador dentro de uma janela de 1 segundo
3. Se o contador exceder o limite, bloqueia o IP/token pelo tempo configurado
4. Requisições com token (`API_KEY` no header) usam o limite do token, que se sobrepõe ao limite por IP

Quando o limite é excedido, retorna:
- **HTTP 429** com a mensagem: `you have reached the maximum number of requests or actions allowed within a certain time frame`

## Arquitetura

```
cmd/server/          → Entrypoint da aplicação
internal/
  config/            → Carregamento de configuração via env
  dto/               → Objetos de resposta HTTP
  limiter/           → Lógica do rate limiting (separada do middleware)
  middleware/        → Middleware HTTP que injeta o rate limiter
  storage/           → Interface Storage + implementações (Redis, Memory)
```

A interface `Storage` permite trocar o Redis por outro mecanismo de persistência sem alterar a lógica do limiter.

## Configuração

Variáveis de ambiente (ou arquivo `.env` na raiz):

| Variável | Descrição | Default |
|---|---|---|
| `IP_LIMIT_RPS` | Requisições por segundo por IP | `10` |
| `IP_BLOCK_DURATION` | Tempo de bloqueio do IP | `300s` |
| `TOKEN_LIMIT_RPS` | Requisições por segundo por token | `100` |
| `TOKEN_BLOCK_DURATION` | Tempo de bloqueio do token | `300s` |
| `REDIS_ADDR` | Endereço do Redis | `localhost:6379` |
| `REDIS_PASSWORD` | Senha do Redis | (vazio) |
| `REDIS_DB` | Database do Redis | `0` |
| `SERVER_PORT` | Porta do servidor HTTP | `8080` |

## Executando

### Com Docker Compose

```bash
# Subir os containers (build + background)
docker-compose up --build -d

# Ver logs
docker-compose logs -f

# Parar os containers
docker-compose stop

# Parar e remover containers, rede e volumes
docker-compose down -v
```

O servidor sobe na porta `8080` com Redis como dependência.

### Local (requer Redis rodando)

```bash
go run cmd/server/main.go
```

## Uso

```bash
# Requisição por IP
curl http://localhost:8080/

# Requisição com token
curl -H "API_KEY: meu-token" http://localhost:8080/

# Health check
curl http://localhost:8080/health
```

## Testes

```bash
# Testes unitários (usa storage em memória)
go test -v -short ./internal/... -count=1

# Testes de integração (usa Redis via testcontainers)
go test -v ./test/integration/... -count=1
```

### Testes de carga

Com a aplicação rodando (`docker-compose up --build -d`), use uma das ferramentas abaixo.

#### Com hey

```bash
# Instalar
go install github.com/rakyll/hey@latest

# 200 requisições, 10 concorrentes - por IP
hey -n 200 -c 10 http://localhost:8080/

# 500 requisições, 20 concorrentes - por token
hey -n 500 -c 20 -H "API_KEY: meu-token" http://localhost:8080/
```

#### Com k6

```bash
# Instalar: https://grafana.com/docs/k6/latest/set-up/install-k6/

# Criar script de teste (k6-test.js)
cat <<'EOF' > k6-test.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    ip_limit: {
      executor: 'constant-arrival-rate',
      rate: 20,
      timeUnit: '1s',
      duration: '10s',
      preAllocatedVUs: 20,
    },
  },
};

export default function () {
  const res = http.get('http://localhost:8080/');
  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
EOF

# Executar
k6 run k6-test.js
```

Nos resultados, espera-se que as primeiras requisições retornem `200` e as seguintes `429` após atingir o limite configurado.

#### Exemplo de resultado (hey)

```text
hey -n 500 -c 20 -H "API_KEY: meu-token" http://localhost:8080/

Summary:
  Total:        0.0349 secs
  Slowest:      0.0142 secs
  Fastest:      0.0002 secs
  Average:      0.0013 secs
  Requests/sec: 14316.3378

  Total data:   45300 bytes
  Size/request: 90 bytes

Response time histogram:
  0.000 [1]     |
  0.002 [402]   |■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.003 [59]    |■■■■■■
  0.004 [21]    |■■
  0.006 [4]     |
  0.007 [12]    |■
  0.009 [0]     |
  0.010 [0]     |
  0.011 [0]     |
  0.013 [0]     |
  0.014 [1]     |

Latency distribution:
  10% in 0.0004 secs
  25% in 0.0006 secs
  50% in 0.0009 secs
  75% in 0.0014 secs
  90% in 0.0025 secs
  95% in 0.0037 secs
  99% in 0.0068 secs

Details (average, fastest, slowest):
  DNS+dialup:   0.0000 secs, 0.0000 secs, 0.0008 secs
  DNS-lookup:   0.0000 secs, 0.0000 secs, 0.0003 secs
  req write:    0.0000 secs, 0.0000 secs, 0.0019 secs
  resp wait:    0.0010 secs, 0.0001 secs, 0.0136 secs
  resp read:    0.0002 secs, 0.0000 secs, 0.0043 secs

Status code distribution:
  [200] 100 responses
  [429] 400 responses
```

Das 500 requisições com token (limite configurado em 100 req/s), 100 retornaram `200` e 400 foram bloqueadas com `429`.