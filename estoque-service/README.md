# estoque-service

Microsserviço responsável pelo cadastro de produtos e controle de saldo em estoque.

## Como rodar

### Opção 1 — Docker Compose (recomendado)
```bash
docker compose up --build
```
Isso sobe o Postgres na porta `5433` (host) e o serviço na porta `8081`. As
migrations rodam automaticamente ao iniciar (veja `internal/db/migrate.go`).

### Opção 2 — Localmente
```bash
cp .env.example .env
# suba um Postgres local ou aponte para um existente, ajustando o .env
go mod tidy
go run ./cmd/api
```

## Endpoints

| Método | Rota                              | Descrição                                   |
|--------|------------------------------------|----------------------------------------------|
| GET    | `/health`                          | Health check                                  |
| POST   | `/api/v1/produtos`                 | Cria um produto                               |
| GET    | `/api/v1/produtos`                 | Lista todos os produtos                       |
| GET    | `/api/v1/produtos/:codigo`         | Busca um produto pelo código                  |
| PATCH  | `/api/v1/produtos/:codigo/saldo`   | Dá baixa ou estorna saldo (usado pelo faturamento-service) |

## Testes manuais via curl

```bash
# Criar produto
curl -X POST http://localhost:8081/api/v1/produtos \
  -H "Content-Type: application/json" \
  -d '{"codigo":"PROD-001","descricao":"Caneta azul","saldo":10}'

# Listar produtos
curl http://localhost:8081/api/v1/produtos

# Buscar um produto
curl http://localhost:8081/api/v1/produtos/PROD-001

# Dar baixa de 2 unidades (simula o que o faturamento-service faz ao imprimir uma nota)
curl -X PATCH http://localhost:8081/api/v1/produtos/PROD-001/saldo \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: teste-123" \
  -d '{"quantidade":2,"operacao":"baixa"}'

# Repetir a mesma chamada com a MESMA Idempotency-Key: o saldo não deve
# diminuir de novo — a resposta salva é devolvida sem reprocessar.
curl -X PATCH http://localhost:8081/api/v1/produtos/PROD-001/saldo \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: teste-123" \
  -d '{"quantidade":2,"operacao":"baixa"}'

# Tentar dar baixa maior que o saldo disponível → 409 SALDO_INSUFICIENTE
curl -X PATCH http://localhost:8081/api/v1/produtos/PROD-001/saldo \
  -H "Content-Type: application/json" \
  -d '{"quantidade":999,"operacao":"baixa"}'
```