# Korp_Teste_NirvanYang

Sistema de Emissão de Notas Fiscais — teste técnico prático (Go + Angular).

## Visão geral da arquitetura

O projeto é composto por três partes, cada uma em sua própria pasta neste repositório:

```
Korp_Teste_NirvanYang/
├── docker-compose.yml         ← sobe os 2 bancos + os 2 microsserviços de uma vez
├── estoque-service/           ← microsserviço de estoque (Go)
├── faturamento-service/       ← microsserviço de faturamento (Go)
└── notas-fiscais-web/         ← frontend (Angular)
```

```
┌────────────────────┐
│  notas-fiscais-web   │  (Angular, porta 4200)
└──────────┬───────────┘
           │ HTTP
           ├─────────────────────────────┐
           ▼                             ▼
┌─────────────────────┐        ┌──────────────────────┐
│  estoque-service      │◄───────┤  faturamento-service   │
│  (porta 8081)          │  HTTP  │  (porta 8082)           │
└──────────┬─────────────┘        └───────────┬─────────────┘
           ▼                                   ▼
     estoque_db (Postgres)             faturamento_db (Postgres)
```

Cada microsserviço tem seu próprio banco de dados ("database per service") — nenhum dos dois acessa o banco do outro diretamente, toda comunicação entre eles acontece via HTTP.

## Como rodar o projeto completo

### 1. Backend (os dois microsserviços + bancos)
```bash
docker compose up --build
```
Isso sobe o `estoque-db`, `faturamento-db`, `estoque-service` (porta `8081`) e `faturamento-service` (porta `8082`), aplicando as migrations automaticamente.

### 2. Frontend
```bash
cd notas-fiscais-web
npm install
npm start
```
Abre em `http://localhost:4200`.

Detalhes de configuração, endpoints e testes específicos de cada parte estão nos READMEs de cada pasta:
- [`estoque-service/README.md`](./estoque-service/README.md)
- [`faturamento-service/README.md`](./faturamento-service/README.md)
- [`notas-fiscais-web/README.md`](./notas-fiscais-web/README.md)

## Fluxo de teste completo (via API)

```bash
# 1. Cadastrar um produto no estoque
curl -X POST http://localhost:8081/api/v1/produtos \
  -H "Content-Type: application/json" \
  -d '{"codigo":"PROD-001","descricao":"Caneta azul","saldo":10}'

# 2. Criar uma nota fiscal usando esse produto
curl -X POST http://localhost:8082/api/v1/notas \
  -H "Content-Type: application/json" \
  -d '{"itens":[{"produtoCodigo":"PROD-001","quantidade":2}]}'
# resposta: {"numero":1,"status":"Aberta","itens":[...]}

# 3. Imprimir a nota
curl -X POST http://localhost:8082/api/v1/notas/1/imprimir \
  -H "Idempotency-Key: impressao-nota-1"
# resposta esperada: status "Fechada"

# 4. Conferir que o saldo do produto baixou de 10 para 8
curl http://localhost:8081/api/v1/produtos/PROD-001

# 5. Tentar imprimir a mesma nota de novo → 409 NOTA_JA_FECHADA
curl -X POST http://localhost:8082/api/v1/notas/1/imprimir
```
O mesmo fluxo também pode ser feito pela interface Angular em `http://localhost:4200`.

## Cenário de falha e recuperação (circuit breaker)

Este é o cenário que atende ao requisito obrigatório de tratamento de falhas: um microsserviço fica indisponível, o sistema detecta isso, dá feedback ao usuário, e se recupera sozinho quando o serviço volta.

```bash
# 1. Com tudo rodando, criar mais uma nota
curl -X POST http://localhost:8082/api/v1/notas \
  -H "Content-Type: application/json" \
  -d '{"itens":[{"produtoCodigo":"PROD-001","quantidade":1}]}'
# supondo que retornou numero: 2

# 2. Derrubar o estoque-service
docker compose stop estoque-service

# 3. Tentar imprimir — as primeiras chamadas demoram até uns 3s (timeout do
#    http.Client) e falham; depois de 5 falhas consecutivas o circuit
#    breaker abre, e as chamadas seguintes falham IMEDIATAMENTE
for i in 1 2 3 4 5 6; do
  echo "tentativa $i:"
  time curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8082/api/v1/notas/2/imprimir
done

# 4. Confirmar que a nota permanece Aberta (não foi fechada pela metade)
curl http://localhost:8082/api/v1/notas/2

# 5. Subir o estoque-service de novo
docker compose start estoque-service

# 6. Esperar o timeout do breaker (10s) e tentar de novo — deve funcionar
sleep 12
curl -X POST http://localhost:8082/api/v1/notas/2/imprimir
```
O mesmo cenário também é visível pela interface: com o `estoque-service` parado, clicar em "Imprimir" no Angular mostra a mensagem de erro (via `MatSnackBar`); ao subir o serviço de novo, o próximo clique funciona normalmente.

## Tratamento de concorrência (diferencial opcional)

```bash
curl -X POST http://localhost:8081/api/v1/produtos \
  -H "Content-Type: application/json" \
  -d '{"codigo":"PROD-002","descricao":"Produto com saldo baixo","saldo":1}'

# duas baixas de 1 unidade, disparadas ao mesmo tempo
curl -X PATCH http://localhost:8081/api/v1/produtos/PROD-002/saldo \
  -H "Content-Type: application/json" -d '{"quantidade":1,"operacao":"baixa"}' &
curl -X PATCH http://localhost:8081/api/v1/produtos/PROD-002/saldo \
  -H "Content-Type: application/json" -d '{"quantidade":1,"operacao":"baixa"}' &
wait
```
Uma das duas deve retornar sucesso (saldo final `0`) e a outra `409 SALDO_INSUFICIENTE` — nunca as duas com sucesso.

---

## Detalhamento técnico

Respostas diretas a cada item pedido no documento de especificação do teste.

### Quais ciclos de vida do Angular foram utilizados?

**`ngOnInit`**, em todos os componentes que carregam dados ao montar a tela:
- `ProdutoListComponent`: carrega a lista de produtos do estoque.
- `NotaListComponent`: carrega a lista de notas fiscais.
- `NotaFormComponent`: carrega a lista de produtos disponíveis para popular o `<mat-select>` de cada item da nota.
- `NotaDetailComponent`: lê o parâmetro `numero` da rota e busca a nota correspondente.

### Foi feito uso da biblioteca RxJS? Em caso afirmativo, como?

Sim. RxJS é usado em três pontos principais:
- **Chamadas HTTP**: todo método dos services (`EstoqueService`, `FaturamentoService`) retorna `Observable`, consumido via `.subscribe()` nos componentes — é o próprio `HttpClient` do Angular que já trabalha nativamente com RxJS.
- **`finalize()`**: usado em `ProdutoFormComponent`, `NotaFormComponent` e `NotaListComponent` para desligar o indicador de carregamento (spinner) independentemente do resultado ser sucesso ou erro — é o que controla, por exemplo, o spinner do botão "Imprimir" enquanto a chamada ao backend está em andamento.
- **`catchError()` + `throwError()`**: usados dentro do `error.interceptor.ts` (`core/interceptors/error.interceptor.ts`) para capturar qualquer erro HTTP vindo de qualquer um dos dois microsserviços, disparar o feedback visual (`MatSnackBar`) de forma centralizada, e repassar o erro adiante para o componente que fez a chamada original.

### Quais outras bibliotecas foram utilizadas e para qual finalidade?

**Frontend (Angular):**
- **`@angular/forms` (Reactive Forms)**: todos os formulários (`produto-form`, `nota-form`) usam `FormBuilder`/`FormGroup`/`FormArray` reativos, com validação declarativa (`Validators.required`, `Validators.min`). O `nota-form` usa especificamente um `FormArray` para permitir múltiplos itens (produto + quantidade) por nota, com linhas adicionadas/removidas dinamicamente.
- **`@angular/router`**: navegação entre as 4 telas, com lazy loading (`loadComponent`) — cada tela só é baixada quando o usuário navega até ela.

**Backend (Go), em ambos os serviços:**
- **`gin-gonic/gin`**: framework HTTP e roteamento (ver pergunta específica de frameworks abaixo).
- **`jmoiron/sqlx`**: acesso a banco com SQL explícito (sem ORM), com mapeamento de linhas para structs via tags `db:"..."`.
- **`lib/pq`**: driver Postgres usado por baixo do `sqlx`.
- **`golang-migrate/migrate`**: aplica as migrations SQL (pasta `migrations/`) automaticamente na subida de cada serviço.
- **`sony/gobreaker`** (só no `faturamento-service`): implementa o circuit breaker usado na comunicação com o `estoque-service` — ver detalhes na resposta sobre tratamento de falhas.

### Para componentes visuais, quais bibliotecas foram utilizadas?

**Angular Material** (`@angular/material`), tema `azure-blue` padrão, sem customização de cor — decisão consciente para manter a interface simples, já que o foco do teste é a lógica de negócio e a arquitetura, não o design visual. Componentes usados: `mat-toolbar` (navegação), `mat-table` (listagens de produtos e notas), `mat-form-field` / `mat-input` / `mat-select` (formulários), `mat-button` / `mat-icon-button` / `mat-stroked-button` (ações), `mat-progress-spinner` (indicadores de carregamento) e `MatSnackBar` (feedback de sucesso/erro).

### Como foi realizado o gerenciamento de dependências no Golang?

**Go Modules**, o gerenciador de dependências nativo do Go desde a versão 1.11 (`go.mod`/`go.sum`). Cada microsserviço é um **módulo Go independente**, com seu próprio `go.mod` — não há dependência de código compartilhada entre `estoque-service` e `faturamento-service`; a única forma de comunicação entre eles é HTTP, reforçando a separação real de microsserviços (cada um pode, inclusive, ser versionado e deployado de forma totalmente independente).

### Quais frameworks foram utilizados no Golang?

**Gin** (`github.com/gin-gonic/gin`), em ambos os serviços — usado para roteamento HTTP, agrupamento de rotas (`router.Group`), binding e validação de JSON de entrada via tags (`binding:"required"`, `binding:"gt=0"` etc.), e middlewares (usado aqui para o tratamento centralizado de erros e para CORS).

*(Este projeto não usa C#, portanto os itens sobre C#/LINQ do documento de especificação não se aplicam.)*

### Como foram tratados os erros e exceções no backend?

Go não possui exceções — erros são valores de retorno explícitos (`error`). A estratégia adotada em ambos os serviços:
1. **Erros customizados** (`internal/apierror/errors.go` em cada serviço): um tipo `APIError` com um `Code` estável (ex: `SALDO_INSUFICIENTE`, `NOTA_JA_FECHADA`, `PRODUTO_NAO_ENCONTRADO`) e uma mensagem amigável, implementando `Error()` e `Unwrap()` para se integrar ao mecanismo padrão de erros do Go (`errors.Is`/`errors.As`).
2. **Propagação por wrapping**: a camada de `repository` traduz erros de baixo nível (ex: violação de `UNIQUE` do Postgres) em `APIError`s de negócio; a camada de `service` aplica as regras (ex: nota já fechada); os `handler`s apenas chamam `c.Error(err)` e retornam, sem decidir status HTTP.
3. **Middleware central de erro** (`internal/middleware/error_middleware.go`): único lugar que converte cada `APIError` em uma resposta HTTP padronizada `{ "error": "mensagem", "code": "CODIGO" }`, com o status code correspondente (400/404/409/503/500). Erros internos genéricos são logados no servidor mas expostos ao cliente com mensagem neutra, por segurança.
4. No `faturamento-service`, o client HTTP do estoque (`internal/client/estoque_client.go`) também distingue **erros de negócio** (ex: saldo insuficiente — o serviço respondeu normalmente) de **erros de infraestrutura** (timeout, conexão recusada, 5xx) — só os segundos contam como falha para o circuit breaker, evitando que ele abra por causa de uma regra de negócio legítima.

### Caso a implementação utilize C#, indicar se foi utilizado LINQ e de que forma.

Não aplicável — a implementação foi feita em **Go**, não em C#.

---

## Requisitos obrigatórios — onde cada um foi atendido

| Requisito | Onde |
|---|---|
| Arquitetura de microsserviços (mín. 2) | `estoque-service` + `faturamento-service`, bancos separados, comunicação via HTTP |
| Tratamento de falhas com recuperação | Circuit breaker (`sony/gobreaker`) em `faturamento-service/internal/client/estoque_client.go` |
| Conexão real com banco de dados | Postgres real em cada serviço (`estoque_db`, `faturamento_db`), via `docker-compose.yml` |

## Diferenciais opcionais implementados

| Diferencial | Onde |
|---|---|
| Tratamento de concorrência | `SELECT ... FOR UPDATE` em `estoque-service/internal/repository/produto_repository.go`, método `AtualizarSaldo` |
| Idempotência | Header `Idempotency-Key`, suportado no endpoint de baixa de saldo (`estoque-service`) e no fluxo de impressão de nota (`faturamento-service`, repassado por item); gerado automaticamente pelo frontend a cada clique em "Imprimir" (`crypto.randomUUID()`) |
| Uso de Inteligência Artificial | Não implementado nesta entrega |