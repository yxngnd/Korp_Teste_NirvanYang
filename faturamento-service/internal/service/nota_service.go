// Package service concentra as regras de negócio do faturamento-service.
package service

import (
	"context"
	"log"

	"github.com/candidato/faturamento-service/internal/apierror"
	"github.com/candidato/faturamento-service/internal/client"
	"github.com/candidato/faturamento-service/internal/dto"
	"github.com/candidato/faturamento-service/internal/model"
	"github.com/candidato/faturamento-service/internal/repository"
)

// NotaService define as operações de negócio do domínio de notas fiscais.
type NotaService interface {
	CriarNota(ctx context.Context, req dto.CriarNotaRequest) (*dto.NotaResponse, error)
	ListarNotas(ctx context.Context) ([]dto.NotaResponse, error)
	BuscarNota(ctx context.Context, numero int) (*dto.NotaResponse, error)
	ImprimirNota(ctx context.Context, numero int, idempotencyKey string) (*dto.NotaResponse, error)
}

type notaService struct {
	repo          repository.NotaRepository
	estoqueClient *client.EstoqueClient
}

// NewNotaService constrói o service a partir de um repository e do client do estoque.
func NewNotaService(repo repository.NotaRepository, estoqueClient *client.EstoqueClient) NotaService {
	return &notaService{repo: repo, estoqueClient: estoqueClient}
}

// CriarNota valida os itens e persiste uma nova nota fiscal com status Aberta.
// Entrada: DTO com a lista de itens (produto + quantidade).
// Saída: DTO de resposta, ou erro de validação.
func (s *notaService) CriarNota(ctx context.Context, req dto.CriarNotaRequest) (*dto.NotaResponse, error) {
	if len(req.Itens) == 0 {
		return nil, apierror.New(apierror.CodeValidacao, "a nota fiscal precisa ter ao menos um item")
	}

	itens := make([]model.NotaItem, 0, len(req.Itens))
	for _, item := range req.Itens {
		if item.Quantidade <= 0 {
			return nil, apierror.New(apierror.CodeValidacao, "quantidade do item deve ser maior que zero")
		}
		itens = append(itens, model.NotaItem{
			ProdutoCodigo: item.ProdutoCodigo,
			Quantidade:    item.Quantidade,
		})
	}

	nota, itensCriados, err := s.repo.CriarNota(ctx, itens)
	if err != nil {
		return nil, err
	}

	response := dto.ToNotaResponse(nota, itensCriados)
	return &response, nil
}

// ListarNotas devolve todas as notas fiscais cadastradas.
func (s *notaService) ListarNotas(ctx context.Context) ([]dto.NotaResponse, error) {
	notas, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	// A listagem não carrega os itens de cada nota (evitaria N+1 queries);
	// a tela de detalhe busca os itens individualmente via BuscarNota.
	response := make([]dto.NotaResponse, 0, len(notas))
	for i := range notas {
		response = append(response, dto.ToNotaResponse(&notas[i], nil))
	}
	return response, nil
}

// BuscarNota devolve uma nota fiscal específica com seus itens.
func (s *notaService) BuscarNota(ctx context.Context, numero int) (*dto.NotaResponse, error) {
	nota, itens, err := s.repo.FindByNumero(ctx, numero)
	if err != nil {
		return nil, err
	}
	response := dto.ToNotaResponse(nota, itens)
	return &response, nil
}

// ImprimirNota é o fluxo central do sistema: valida o status da nota, dá
// baixa no estoque de cada item, e só então fecha a nota.
//
// Entrada: número da nota e uma chave de idempotência (gerada pelo frontend
// a cada clique no botão de imprimir, para proteger contra duplo clique).
// Saída: nota com status Fechada, ou erro (nota já fechada, saldo
// insuficiente em algum item, ou estoque indisponível).
//
// Estratégia de compensação: se o item N falhar depois que os itens
// 1..N-1 já tiveram baixa confirmada no estoque, os itens já processados
// são estornados antes de retornar o erro — a nota permanece Aberta e o
// estoque volta ao estado anterior à tentativa. Isso evita o cenário de
// "nota fechada pela metade" ou "estoque debitado sem a nota realmente
// ter sido impressa".
func (s *notaService) ImprimirNota(ctx context.Context, numero int, idempotencyKey string) (*dto.NotaResponse, error) {
	nota, itens, err := s.repo.FindByNumero(ctx, numero)
	if err != nil {
		return nil, err
	}

	if nota.Status != model.StatusAberta {
		return nil, apierror.ErrNotaJaFechada
	}

	itensProcessados := make([]model.NotaItem, 0, len(itens))
	for _, item := range itens {
		// A idempotency key é derivada por item (não só pelo número da
		// nota), para que cada chamada individual ao estoque-service seja
		// idempotente de forma independente.
		itemKey := ""
		if idempotencyKey != "" {
			itemKey = idempotencyKey + "-" + item.ProdutoCodigo
		}

		if baixaErr := s.estoqueClient.BaixarEstoque(ctx, item.ProdutoCodigo, item.Quantidade, itemKey); baixaErr != nil {
			s.compensar(ctx, itensProcessados)
			return nil, baixaErr
		}
		itensProcessados = append(itensProcessados, item)
	}

	if err := s.repo.AtualizarStatus(ctx, numero, model.StatusFechada); err != nil {
		// Neste ponto o estoque já foi debitado com sucesso para todos os
		// itens; se o UPDATE de status falhar (ex: banco caiu bem nesse
		// instante), compensamos o estoque também, para não deixar uma
		// baixa "órfã" sem a nota correspondente ter fechado.
		s.compensar(ctx, itensProcessados)
		return nil, err
	}

	nota.Status = model.StatusFechada
	response := dto.ToNotaResponse(nota, itens)
	return &response, nil
}

// compensar estorna, no estoque-service, todos os itens que já tiveram
// baixa confirmada antes de um erro interromper a impressão da nota.
func (s *notaService) compensar(ctx context.Context, itensProcessados []model.NotaItem) {
	for _, item := range itensProcessados {
		if err := s.estoqueClient.EstornarEstoque(ctx, item.ProdutoCodigo, item.Quantidade, ""); err != nil {
			// Se o próprio estorno falhar (ex: estoque também indisponível
			// nesse momento), não há mais o que fazer automaticamente —
			// loga com detalhe para permitir uma correção manual, já que
			// isso representa uma inconsistência real entre os serviços.
			log.Printf("ATENÇÃO: falha ao compensar estoque do produto %s (quantidade %d): %v",
				item.ProdutoCodigo, item.Quantidade, err)
		}
	}
}
