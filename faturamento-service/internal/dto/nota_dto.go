// Package dto contém os contratos de entrada e saída da API HTTP do faturamento-service.
package dto

import "github.com/candidato/faturamento-service/internal/model"

// ItemRequest representa um item dentro do corpo de criação de nota.
type ItemRequest struct {
	ProdutoCodigo string `json:"produtoCodigo" binding:"required"`
	Quantidade    int    `json:"quantidade" binding:"required,gt=0"`
}

// CriarNotaRequest é o corpo esperado em POST /notas.
// A validação "min=1" garante que a nota não seja criada sem nenhum item,
// já que o requisito descreve "inclusão de múltiplos produtos" como parte
// essencial do cadastro de nota fiscal.
type CriarNotaRequest struct {
	Itens []ItemRequest `json:"itens" binding:"required,min=1,dive"`
}

// ItemResponse é o formato de item devolvido pela API.
type ItemResponse struct {
	ProdutoCodigo string `json:"produtoCodigo"`
	Quantidade    int    `json:"quantidade"`
}

// NotaResponse é o formato de nota fiscal devolvido pela API.
type NotaResponse struct {
	Numero int            `json:"numero"`
	Status string         `json:"status"`
	Itens  []ItemResponse `json:"itens"`
}

// ToNotaResponse mapeia o model interno (nota + itens) para o DTO de resposta.
// Entrada: nota e sua lista de itens, vindos do banco.
// Saída: DTO pronto para ser serializado em JSON.
func ToNotaResponse(nota *model.NotaFiscal, itens []model.NotaItem) NotaResponse {
	itensResponse := make([]ItemResponse, 0, len(itens))
	for _, item := range itens {
		itensResponse = append(itensResponse, ItemResponse{
			ProdutoCodigo: item.ProdutoCodigo,
			Quantidade:    item.Quantidade,
		})
	}

	return NotaResponse{
		Numero: nota.Numero,
		Status: string(nota.Status),
		Itens:  itensResponse,
	}
}
