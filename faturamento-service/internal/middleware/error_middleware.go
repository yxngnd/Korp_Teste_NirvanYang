package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/candidato/faturamento-service/internal/apierror"
	"github.com/gin-gonic/gin"
)

type errorResponse struct {
	Error string        `json:"error"`
	Code  apierror.Code `json:"code"`
}

// ErrorHandlerMiddleware roda após o handler e traduz qualquer erro
// registrado em c.Errors para uma resposta HTTP consistente.
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) {
			if apiErr.HTTPStatus() == http.StatusInternalServerError {
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

		log.Printf("erro não mapeado: %v", err)
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "erro interno do servidor",
			Code:  apierror.CodeInterno,
		})
	}
}
