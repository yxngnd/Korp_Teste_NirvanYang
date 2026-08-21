// Package handler contém os controllers HTTP: fazem o parsing da requisição,
// chamam o service, e devolvem a resposta. Nenhuma regra de negócio mora aqui.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yxngnd/estoque-service/internal/dto"
	"github.com/yxngnd/estoque-service/internal/service"
)

// ProdutoHandler agrupa os endpoints HTTP do domínio de produtos.
type ProdutoHandler struct {
	service service.ProdutoService
}

// NewProdutoHandler constrói o handler a partir de um service já pronto.
func NewProdutoHandler(service service.ProdutoService) *ProdutoHandler {
	return &ProdutoHandler{service: service}
}

// CriarProduto trata POST /api/v1/produtos.
// Entrada: corpo JSON no formato dto.CreateProdutoRequest.
// Saída HTTP: 201 com o produto criado; 400 se o JSON for inválido/incompleto;
// 409 se o código já existir (erro tratado pelo middleware central).
func (h *ProdutoHandler) CriarProduto(c *gin.Context) {
	var req dto.CreateProdutoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido: " + err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	produto, err := h.service.CriarProduto(c.Request.Context(), req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, produto)
}

// ListarProdutos trata GET /api/v1/produtos.
// Saída HTTP: 200 com a lista completa de produtos.
func (h *ProdutoHandler) ListarProdutos(c *gin.Context) {
	produtos, err := h.service.ListarProdutos(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, produtos)
}

// BuscarProduto trata GET /api/v1/produtos/:codigo.
// Entrada: path param "codigo".
// Saída HTTP: 200 com o produto, ou 404 se não existir.
func (h *ProdutoHandler) BuscarProduto(c *gin.Context) {
	codigo := c.Param("codigo")

	produto, err := h.service.BuscarProduto(c.Request.Context(), codigo)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, produto)
}

// AtualizarSaldo trata PATCH /api/v1/produtos/:codigo/saldo.
// Este é o endpoint consumido pelo faturamento-service para dar baixa
// (ou estornar) o saldo de um produto ao imprimir/compensar uma nota fiscal.
//
// Entrada: path param "codigo", corpo JSON dto.AtualizarSaldoRequest,
// e opcionalmente o header "Idempotency-Key".
// Saída HTTP: 200 com o produto atualizado; 404 se não encontrado;
// 409 se o saldo for insuficiente para uma baixa.
func (h *ProdutoHandler) AtualizarSaldo(c *gin.Context) {
	codigo := c.Param("codigo")

	var req dto.AtualizarSaldoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "corpo da requisição inválido: " + err.Error(), "code": "VALIDATION_ERROR"})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	produto, err := h.service.AtualizarSaldo(c.Request.Context(), codigo, req, idempotencyKey)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, produto)
}
