// Package middleware contém os middlewares HTTP compartilhados por todos os handlers.
package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yxngnd/estoque-service/internal/apierror"
)

// errorResponse é o formato JSON padronizado devolvido para qualquer erro da API.
// O campo Code permite que o frontend trate cada caso sem parsear a mensagem.
type errorResponse struct {
	Error string        `json:"error"`
	Code  apierror.Code `json:"code"`
}

// ErrorHandlerMiddleware roda após o handler (via c.Next()) e traduz
// qualquer erro registrado em c.Errors para uma resposta HTTP consistente.
//
// Os handlers, ao invés de escrever a resposta de erro diretamente, apenas
// chamam c.Error(err) e retornam — esse middleware é o único responsável
// por decidir o status code e o corpo da resposta.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// Usa o último erro registrado, que normalmente é o mais específico.
		err := c.Errors.Last().Err

		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) {
			if apiErr.HTTPStatus() == http.StatusInternalServerError {
				// Erros internos são logados com detalhe no servidor, mas a
				// mensagem exposta ao cliente permanece genérica por segurança.
				log.Printf("erro interno: %v", apiErr)
				c.JSON(http.StatusInternalServerError, errorResponse{
					Error: "erro interno do servidor",
					Code:  apierror.CodeInterno,
				})
				return
			}
			c.JSON(apiErr.HTTPStatus(), errorResponse{
				Error: apiErr.Message,
				Code:  apiErr.Code,
			})
			return
		}

		// Erro não mapeado (ex: bug inesperado) — trata como erro interno
		// e loga a stack para investigação.
		log.Printf("erro não mapeado: %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "erro interno do servidor",
			Code:  apierror.CodeInterno,
		})
	}
}
