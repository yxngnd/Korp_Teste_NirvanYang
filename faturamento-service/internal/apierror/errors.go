// Package apierror define os erros de negócio do faturamento-service,
// desacoplados de detalhe HTTP. O middleware de erro é o único lugar que
// sabe converter esses erros em status code + JSON.
package apierror

import "net/http"

// Code identifica o tipo de erro de negócio de forma estável, para o
// frontend tratar cada caso sem depender de parsing de mensagem.
type Code string

const (
	CodeValidacao            Code = "VALIDATION_ERROR"
	CodeNotaNaoEncontrada    Code = "NOTA_NAO_ENCONTRADA"
	CodeNotaJaFechada        Code = "NOTA_JA_FECHADA"
	CodeSaldoInsuficiente    Code = "SALDO_INSUFICIENTE"
	CodeProdutoNaoEncontrado Code = "PRODUTO_NAO_ENCONTRADO"
	CodeEstoqueIndisponivel  Code = "ESTOQUE_INDISPONIVEL"
	CodeInterno              Code = "INTERNAL_ERROR"
)

var httpStatusByCode = map[Code]int{
	CodeValidacao:            http.StatusBadRequest,
	CodeNotaNaoEncontrada:    http.StatusNotFound,
	CodeNotaJaFechada:        http.StatusConflict,
	CodeSaldoInsuficiente:    http.StatusConflict,
	CodeProdutoNaoEncontrado: http.StatusNotFound,
	CodeEstoqueIndisponivel:  http.StatusServiceUnavailable,
	CodeInterno:              http.StatusInternalServerError,
}

// APIError é o tipo de erro que atravessa repository/client -> service -> handler.
type APIError struct {
	Code    Code   `json:"code"`
	Message string `json:"error"`
	Err     error  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// HTTPStatus devolve o status code HTTP correspondente ao código de negócio.
func (e *APIError) HTTPStatus() int {
	if status, ok := httpStatusByCode[e.Code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

func New(code Code, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

func Wrap(code Code, message string, err error) *APIError {
	return &APIError{Code: code, Message: message, Err: err}
}

// Erros sentinela reutilizáveis para os casos mais comuns do domínio de notas fiscais.
var (
	ErrNotaNaoEncontrada = New(CodeNotaNaoEncontrada, "nota fiscal não encontrada")
	ErrNotaJaFechada     = New(CodeNotaJaFechada, "nota fiscal não pode ser impressa pois não está com status Aberta")
	// ErrEstoqueIndisponivel é usado quando o circuit breaker está aberto ou
	// a chamada ao estoque-service falha por timeout/conexão recusada —
	// o usuário recebe feedback claro em vez de a requisição ficar travada.
	ErrEstoqueIndisponivel = New(CodeEstoqueIndisponivel, "serviço de estoque indisponível no momento, tente novamente em instantes")
)

// ErrSaldoInsuficiente monta a mensagem já com o código do produto.
func ErrSaldoInsuficiente(codigoProduto string) *APIError {
	return New(CodeSaldoInsuficiente, "saldo insuficiente para o produto "+codigoProduto)
}

// ErrProdutoNaoEncontrado monta a mensagem já com o código do produto.
func ErrProdutoNaoEncontrado(codigoProduto string) *APIError {
	return New(CodeProdutoNaoEncontrado, "produto "+codigoProduto+" não encontrado no estoque")
}
