// Package dto contém os contratos de entrada e saída da API HTTP.
// Ficam separados do model para não expor a estrutura interna do banco
// e para permitir validação declarativa via tags do go-playground/validator
// (já embutido no Gin através de ShouldBindJSON).
package dto

import "github.com/yxngnd/estoque-service/internal/model"

// CreateProdutoRequest é o corpo esperado em POST /produtos.
type CreateProdutoRequest struct {
	Codigo    string `json:"codigo" binding:"required"`
	Descricao string `json:"descricao" binding:"required"`
	Saldo     int    `json:"saldo" binding:"gte=0"`
}

// Operacao identifica o tipo de ajuste de saldo solicitado.
type Operacao string

const (
	OperacaoBaixa   Operacao = "baixa"
	OperacaoEstorno Operacao = "estorno"
)

// AtualizarSaldoRequest é o corpo esperado em PATCH /produtos/{codigo}/saldo.
// Endpoint consumido pelo faturamento-service ao imprimir (baixa) ou
// compensar (estorno) uma nota fiscal.
type AtualizarSaldoRequest struct {
	Quantidade int      `json:"quantidade" binding:"required,gt=0"`
	Operacao   Operacao `json:"operacao" binding:"required,oneof=baixa estorno"`
}

// ProdutoResponse é o formato devolvido pela API para o cliente (Angular ou faturamento-service).
type ProdutoResponse struct {
	Codigo    string `json:"codigo"`
	Descricao string `json:"descricao"`
	Saldo     int    `json:"saldo"`
}

// ToProdutoResponse mapeia o model interno para o DTO de resposta.
// Entrada: produto vindo do banco.
// Saída: DTO pronto para ser serializado em JSON.
func ToProdutoResponse(p *model.Produto) ProdutoResponse {
	return ProdutoResponse{
		Codigo:    p.Codigo,
		Descricao: p.Descricao,
		Saldo:     p.Saldo,
	}
}

// ToProdutoResponseList aplica ToProdutoResponse a uma lista inteira.
func ToProdutoResponseList(produtos []model.Produto) []ProdutoResponse {
	response := make([]ProdutoResponse, 0, len(produtos))
	for i := range produtos {
		response = append(response, ToProdutoResponse(&produtos[i]))
	}
	return response
}
