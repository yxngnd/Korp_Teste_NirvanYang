// Package service concentra as regras de negócio. Não conhece HTTP nem SQL
// diretamente — depende apenas da interface repository.ProdutoRepository.
package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/yxngnd/estoque-service/internal/apierror"
	"github.com/yxngnd/estoque-service/internal/dto"
	"github.com/yxngnd/estoque-service/internal/model"
	"github.com/yxngnd/estoque-service/internal/repository"
)

// ProdutoService define as operações de negócio do domínio de produtos.
type ProdutoService interface {
	CriarProduto(ctx context.Context, req dto.CreateProdutoRequest) (*dto.ProdutoResponse, error)
	ListarProdutos(ctx context.Context) ([]dto.ProdutoResponse, error)
	BuscarProduto(ctx context.Context, codigo string) (*dto.ProdutoResponse, error)
	AtualizarSaldo(ctx context.Context, codigo string, req dto.AtualizarSaldoRequest, idempotencyKey string) (*dto.ProdutoResponse, error)
}

type produtoService struct {
	repo repository.ProdutoRepository
}

// NewProdutoService constrói o service a partir de um repository já pronto.
func NewProdutoService(repo repository.ProdutoRepository) ProdutoService {
	return &produtoService{repo: repo}
}

// CriarProduto valida e persiste um novo produto.
// Entrada: DTO de criação (já validado estruturalmente pelo binding do Gin).
// Saída: DTO de resposta, ou erro de validação/duplicidade.
func (s *produtoService) CriarProduto(ctx context.Context, req dto.CreateProdutoRequest) (*dto.ProdutoResponse, error) {
	codigo := strings.TrimSpace(req.Codigo)
	descricao := strings.TrimSpace(req.Descricao)

	if codigo == "" || descricao == "" {
		return nil, apierror.New(apierror.CodeValidacao, "código e descrição são obrigatórios")
	}

	produto := &model.Produto{
		Codigo:    codigo,
		Descricao: descricao,
		Saldo:     req.Saldo,
	}

	if err := s.repo.Create(ctx, produto); err != nil {
		return nil, err
	}

	response := dto.ToProdutoResponse(produto)
	return &response, nil
}

// ListarProdutos devolve todos os produtos cadastrados.
func (s *produtoService) ListarProdutos(ctx context.Context) ([]dto.ProdutoResponse, error) {
	produtos, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	return dto.ToProdutoResponseList(produtos), nil
}

// BuscarProduto devolve um produto específico pelo código.
func (s *produtoService) BuscarProduto(ctx context.Context, codigo string) (*dto.ProdutoResponse, error) {
	produto, err := s.repo.FindByCodigo(ctx, codigo)
	if err != nil {
		return nil, err
	}
	response := dto.ToProdutoResponse(produto)
	return &response, nil
}

// AtualizarSaldo aplica uma baixa ou estorno de saldo, respeitando idempotência
// quando uma chave é informada.
//
// Entrada: código do produto, request com quantidade/operação, e a chave de
// idempotência (pode vir vazia, caso em que o comportamento de idempotência
// simplesmente não se aplica).
// Saída: DTO com o saldo já atualizado, ou erro de negócio.
func (s *produtoService) AtualizarSaldo(ctx context.Context, codigo string, req dto.AtualizarSaldoRequest, idempotencyKey string) (*dto.ProdutoResponse, error) {
	// Se a chave já foi processada antes, devolve a resposta salva sem
	// tocar no saldo de novo. Isso protege contra retries do circuit
	// breaker do faturamento-service e contra duplo clique no frontend.
	if idempotencyKey != "" {
		respostaSalva, existe, err := s.repo.BuscarIdempotencyKey(ctx, idempotencyKey)
		if err != nil {
			return nil, apierror.Wrap(apierror.CodeInterno, "erro ao verificar idempotência", err)
		}
		if existe {
			var response dto.ProdutoResponse
			if err := json.Unmarshal([]byte(respostaSalva), &response); err != nil {
				return nil, apierror.Wrap(apierror.CodeInterno, "erro ao decodificar resposta idempotente", err)
			}
			return &response, nil
		}
	}

	delta := req.Quantidade
	if req.Operacao == dto.OperacaoBaixa {
		delta = -delta
	}

	produto, err := s.repo.AtualizarSaldo(ctx, codigo, delta)
	if err != nil {
		return nil, err
	}

	response := dto.ToProdutoResponse(produto)

	if idempotencyKey != "" {
		respostaJSON, err := json.Marshal(response)
		if err == nil {
			// Falha ao salvar a chave não deve derrubar a operação principal
			// (o saldo já foi corretamente atualizado); é apenas uma
			// degradação da proteção de idempotência em retries futuros.
			_ = s.repo.SalvarIdempotencyKey(ctx, idempotencyKey, string(respostaJSON))
		}
	}

	return &response, nil
}
