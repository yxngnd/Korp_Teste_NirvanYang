// Package handler contém os controllers HTTP do faturamento-service.
package handler

import (
	"net/http"
	"strconv"

	"github.com/candidato/faturamento-service/internal/dto"
	"github.com/candidato/faturamento-service/internal/service"
	"github.com/gin-gonic/gin"
)

// NotaHandler agrupa os endpoints HTTP do domínio de notas fiscais.
type NotaHandler struct {
	service service.NotaService
}

// NewNotaHandler constrói o handler a partir de um service já pronto.
func NewNotaHandler(service service.NotaService) *NotaHandler {
	return &NotaHandler{service: service}
}

// CriarNota trata POST /api/v1/notas.
// Entrada: corpo JSON no formato dto.CriarNotaRequest (lista de itens).
// Saída HTTP: 201 com a nota criada (status Aberta); 400 se inválida.
func (h *NotaHandler) CriarNota(c *gin.Context) {
	var req dto.CriarNotaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido: " + err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	nota, err := h.service.CriarNota(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, nota)
}

// ListarNotas trata GET /api/v1/notas.
// Saída HTTP: 200 com a lista de notas (sem os itens de cada uma).
func (h *NotaHandler) ListarNotas(c *gin.Context) {
	notas, err := h.service.ListarNotas(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, notas)
}

// BuscarNota trata GET /api/v1/notas/:numero.
// Entrada: path param "numero" (inteiro).
// Saída HTTP: 200 com a nota e seus itens; 400 se o número não for válido; 404 se não existir.
func (h *NotaHandler) BuscarNota(c *gin.Context) {
	numero, err := parseNumero(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "número da nota inválido", "code": "VALIDATION_ERROR"})
		return
	}

	nota, err := h.service.BuscarNota(c.Request.Context(), numero)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, nota)
}

// ImprimirNota trata POST /api/v1/notas/:numero/imprimir.
// Este é o endpoint acionado pelo botão "Imprimir" do Angular.
//
// Entrada: path param "numero", header opcional "Idempotency-Key".
// Saída HTTP: 200 com a nota já Fechada; 409 se a nota já estava fechada ou
// se algum item teve saldo insuficiente; 503 se o estoque-service estiver
// indisponível (circuit breaker aberto ou timeout).
func (h *NotaHandler) ImprimirNota(c *gin.Context) {
	numero, err := parseNumero(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "número da nota inválido", "code": "VALIDATION_ERROR"})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	nota, err := h.service.ImprimirNota(c.Request.Context(), numero, idempotencyKey)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, nota)
}

func parseNumero(c *gin.Context) (int, error) {
	return strconv.Atoi(c.Param("numero"))
}
