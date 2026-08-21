# faturamento-service

Microsserviço responsável pelo cadastro e impressão de notas fiscais.
Depende do `estoque-service` para dar baixa no saldo dos produtos ao imprimir.

> Este serviço não sobe sozinho de forma útil, pois depende do
> `estoque-service` rodando para completar o fluxo de impressão. Use o
> `docker-compose.yml` da pasta raiz do repositório, que sobe os dois
> serviços e seus bancos juntos — veja o README raiz para o fluxo de
> testes completo, incluindo o cenário de falha e recuperação.

## Como rodar localmente (sem Docker)

```bash
cp .env.example .env
# ajuste ESTOQUE_SERVICE_URL no .env se o estoque-service não estiver em localhost:8081
go mod tidy
go run ./cmd/api
```

## Endpoints

| Método | Rota                              | Descrição                                          |
|--------|-------------------------------------|-----------------------------------------------------|
| GET    | `/health`                           | Health check                                         |
| POST   | `/api/v1/notas`                     | Cria nota fiscal (status inicial `Aberta`)           |
| GET    | `/api/v1/notas`                     | Lista todas as notas                                 |
| GET    | `/api/v1/notas/:numero`             | Busca uma nota específica com seus itens             |
| POST   | `/api/v1/notas/:numero/imprimir`    | Imprime a nota: baixa estoque de cada item e fecha   |

Veja o README raiz do projeto para os comandos `curl` completos, incluindo o
fluxo de teste do cenário de falha do `estoque-service` e recuperação via
circuit breaker.