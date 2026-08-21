# notas-fiscais-web

Interface Angular do Sistema de Emissão de Notas Fiscais. Consome os dois
microsserviços (`estoque-service` na porta `8081` e `faturamento-service`
na porta `8082`) diretamente, sem API Gateway.

Interface propositalmente simples — sem tema customizado, sem excesso de
componentes visuais — focada em atender exatamente os requisitos do teste.

## Como rodar

```bash
npm install
npm start
```

Abre em `http://localhost:4200`. As URLs dos microsserviços estão fixadas
em `src/environments/environment.ts` — ajuste ali se suas portas forem diferentes.

> **Pré-requisito**: os dois microsserviços (`estoque-service` e
> `faturamento-service`) precisam estar rodando antes de usar a interface
> — veja o README raiz do repositório para subir o backend.

## Telas

| Rota | Tela | Descrição |
|---|---|---|
| `/produtos` | Lista + cadastro de produtos | Formulário de cadastro embutido na própria listagem |
| `/notas` | Lista de notas fiscais | Botão "Imprimir" em cada linha, desabilitado se a nota não estiver `Aberta` |
| `/notas/novo` | Criação de nota fiscal | Lista dinâmica de itens (produto + quantidade), com produtos carregados do estoque-service |
| `/notas/:numero` | Detalhe de uma nota | Status e itens da nota |

## Decisões técnicas

### Ciclos de vida do Angular utilizados
- **`ngOnInit`**: em todos os componentes que carregam dados ao montar a tela — `produto-list` (lista de produtos), `nota-list` (lista de notas), `nota-form` (lista de produtos disponíveis para o select), `nota-detail` (busca a nota pelo parâmetro de rota).

### Uso do RxJS
- Toda chamada HTTP (`EstoqueService`, `FaturamentoService`) retorna `Observable`, tratado via `.subscribe()`.
- **`finalize()`**: usado em `produto-form`, `nota-form` e `nota-list` para desligar o indicador de carregamento/loading independentemente do resultado ser sucesso ou erro — é o que controla o spinner do botão "Imprimir", por exemplo.
- **`catchError()` + `throwError()`**: usados dentro do `error.interceptor.ts` para capturar qualquer erro HTTP da aplicação, disparar o feedback visual (snackbar) e repassar o erro adiante para o componente que fez a chamada.

### Outras bibliotecas utilizadas
- **RxJS**: nativo do Angular, usado como descrito acima.
- **Reactive Forms** (`@angular/forms`): todos os formulários (`produto-form`, `nota-form`) usam `FormBuilder`/`FormGroup`/`FormArray` reativos, com validação declarativa (`Validators.required`, `Validators.min`).

### Bibliotecas de componentes visuais
- **Angular Material**: biblioteca oficial do time Angular, usada para toolbar, tabela (`mat-table`), formulários (`mat-form-field`, `mat-select`), botões, spinner de carregamento e `MatSnackBar` para feedback de erro/sucesso. Tema `azure-blue` pronto (sem customização de cores), propositalmente neutro para não fugir do escopo de um teste técnico.

### Tratamento de erros no frontend
- `error.interceptor.ts` é um `HttpInterceptorFn` registrado globalmente em `app.config.ts` via `provideHttpClient(withInterceptors([errorInterceptor]))`. Ele intercepta toda resposta de erro HTTP vinda de qualquer um dos dois microsserviços, interpreta o campo `code` do corpo JSON (`SALDO_INSUFICIENTE`, `NOTA_JA_FECHADA`, `ESTOQUE_INDISPONIVEL`, etc. — o mesmo formato usado pelo middleware de erro do backend Go) e mostra uma mensagem amigável via `MatSnackBar`, sem que cada componente precise tratar isso individualmente.

### Idempotência no frontend
- `FaturamentoService.imprimirNota()` gera uma `Idempotency-Key` nova (`crypto.randomUUID()`) a cada chamada e a envia como header — protege contra duplo clique no botão de imprimir, complementando a idempotência já implementada no backend.

## Estrutura de pastas

```
src/app/
├── core/
│   ├── models/          # interfaces TypeScript (contrato com a API)
│   ├── services/         # EstoqueService, FaturamentoService
│   └── interceptors/      # error.interceptor.ts
├── features/
│   ├── produtos/
│   │   ├── produto-list/
│   │   └── produto-form/
│   └── notas-fiscais/
│       ├── nota-list/
│       ├── nota-form/
│       └── nota-detail/
├── app.routes.ts          # rotas com lazy loading (loadComponent)
├── app.config.ts          # providers globais (router, http+interceptor, animations)
└── app.component.ts        # shell com toolbar de navegação
```