import { HttpErrorResponse, HttpEvent, HttpHandlerFn, HttpRequest } from '@angular/common/http';
import { inject } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Observable, catchError, throwError } from 'rxjs';

// Formato de erro padronizado devolvido pelo middleware de erro dos dois
// microsserviços Go: { "error": "mensagem", "code": "ALGUM_CODIGO" }.
interface ApiErrorBody {
  error?: string;
  code?: string;
}

// errorInterceptor centraliza o tratamento de erro HTTP para toda a
// aplicação: qualquer chamada feita via HttpClient (EstoqueService ou
// FaturamentoService) passa por aqui antes de chegar ao .subscribe() do
// componente. Assim, o feedback ao usuário (snackbar) é escrito uma única
// vez, em vez de repetido em cada service.
export function errorInterceptor(
  req: HttpRequest<unknown>,
  next: HttpHandlerFn,
): Observable<HttpEvent<unknown>> {
  const snackBar = inject(MatSnackBar);

  return next(req).pipe(
    catchError((err: HttpErrorResponse) => {
      snackBar.open(mensagemAmigavel(err), 'Fechar', {
        duration: 5000,
        horizontalPosition: 'center',
        verticalPosition: 'bottom',
      });
      // Re-lança o erro para que o componente que fez a chamada ainda
      // possa reagir localmente se precisar (ex: manter um botão
      // desabilitado) — o interceptor cuida só do feedback genérico.
      return throwError(() => err);
    }),
  );
}

function mensagemAmigavel(err: HttpErrorResponse): string {
  // status 0 normalmente significa que a requisição nem chegou a um
  // servidor (serviço fora do ar, CORS bloqueado, sem rede).
  if (err.status === 0) {
    return 'Não foi possível conectar ao servidor. Verifique se os serviços estão no ar.';
  }

  const body = err.error as ApiErrorBody | undefined;

  switch (body?.code) {
    case 'SALDO_INSUFICIENTE':
      return body.error ?? 'Saldo insuficiente para concluir a operação.';
    case 'NOTA_JA_FECHADA':
      return 'Esta nota fiscal já foi impressa anteriormente.';
    case 'ESTOQUE_INDISPONIVEL':
      return 'Serviço de estoque indisponível no momento. Tente novamente em instantes.';
    case 'PRODUTO_NAO_ENCONTRADO':
    case 'NOTA_NAO_ENCONTRADA':
      return body.error ?? 'Registro não encontrado.';
    case 'CODIGO_DUPLICADO':
      return 'Já existe um produto cadastrado com este código.';
    case 'VALIDATION_ERROR':
      return body.error ?? 'Dados inválidos. Confira os campos preenchidos.';
    default:
      return body?.error ?? 'Ocorreu um erro inesperado. Tente novamente.';
  }
}
