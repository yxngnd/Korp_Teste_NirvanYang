// Package apierror define os erros de negócio da aplicação, desacoplados
// de qualquer detalhe HTTP. O middleware de erro (internal/middleware) é o
// único lugar que sabe converter esses erros em status code + JSON.
package apierror

import "net/http"

// Code identifica o tipo de erro de negócio de forma estável, para o
// frontend poder tratar cada caso sem depender de parsing de mensagem.
type Code string

const (
	CodeValidacao            Code = "VALIDATION_ERROR"
	CodeProdutoNaoEncontrado Code = "PRODUTO_NAO_ENCONTRADO"
	CodeCodigoDuplicado      Code = "CODIGO_DUPLICADO"
	CodeSaldoInsuficiente    Code = "SALDO_INSUFICIENTE"
	CodeInterno              Code = "INTERNAL_ERROR"
)

// httpStatusByCode mapeia cada código de negócio para o status HTTP correspondente.
// Fica centralizado aqui para não espalhar "if status == ..." pelos handlers.
var httpStatusByCode = map[Code]int{
	CodeValidacao:            http.StatusBadRequest,
	CodeProdutoNaoEncontrado: http.StatusNotFound,
	CodeCodigoDuplicado:      http.StatusConflict,
	CodeSaldoInsuficiente:    http.StatusConflict,
	CodeInterno:              http.StatusInternalServerError,
}

// APIError é o tipo de erro que atravessa repository -> service -> handler.
// Implementa a interface error padrão do Go (método Error()).
type APIError struct {
	Code    Code   `json:"code"`
	Message string `json:"error"`
	// Err guarda o erro original (ex: erro do driver do banco), para logging
	// interno. Nunca é exposto na resposta HTTP.
	Err error `json:"-"`
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap permite usar errors.Is / errors.As sobre o erro original,
// preservando a causa raiz mesmo depois do wrapping.
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

// New cria um novo APIError sem erro original (ex: erro de validação
// detectado na própria aplicação, não vindo de outra camada).
func New(code Code, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

// Wrap cria um novo APIError preservando o erro original como causa raiz.
// Usado quando a camada de repository recebe um erro do driver do banco
// e precisa traduzi-lo para um erro de negócio conhecido.
func Wrap(code Code, message string, err error) *APIError {
	return &APIError{Code: code, Message: message, Err: err}
}

// Erros sentinela reutilizáveis para os casos mais comuns do domínio de produtos.
var (
	ErrProdutoNaoEncontrado = New(CodeProdutoNaoEncontrado, "produto não encontrado")
	ErrCodigoDuplicado      = New(CodeCodigoDuplicado, "já existe um produto com este código")
)

// ErrSaldoInsuficiente monta a mensagem já com o código do produto,
// para dar contexto direto ao usuário final.
func ErrSaldoInsuficiente(codigo string) *APIError {
	return New(CodeSaldoInsuficiente, "saldo insuficiente para o produto "+codigo)
}
