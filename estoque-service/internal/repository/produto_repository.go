// Package repository é a única camada que executa SQL. Service nunca fala
// diretamente com o banco — sempre através das interfaces definidas aqui.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yxngnd/estoque-service/internal/apierror"
	"github.com/yxngnd/estoque-service/internal/model"
)

// pqUniqueViolation é o código de erro do Postgres para violação de UNIQUE constraint.
// Referência: https://www.postgresql.org/docs/current/errcodes-appendix.html
const pqUniqueViolation = "23505"

// ProdutoRepository define o contrato de persistência do domínio de produtos.
// A interface existe para permitir mockar o repository nos testes do service,
// sem precisar de um banco real.
type ProdutoRepository interface {
	Create(ctx context.Context, p *model.Produto) error
	FindByCodigo(ctx context.Context, codigo string) (*model.Produto, error)
	FindAll(ctx context.Context) ([]model.Produto, error)
	AtualizarSaldo(ctx context.Context, codigo string, delta int) (*model.Produto, error)
	BuscarIdempotencyKey(ctx context.Context, chave string) (string, bool, error)
	SalvarIdempotencyKey(ctx context.Context, chave string, respostaJSON string) error
}

type postgresProdutoRepository struct {
	db *sqlx.DB
}

// NewProdutoRepository constrói a implementação Postgres do repository.
func NewProdutoRepository(db *sqlx.DB) ProdutoRepository {
	return &postgresProdutoRepository{db: db}
}

// Create insere um novo produto.
// Entrada: contexto e produto preenchido (sem ID).
// Saída: erro apierror.ErrCodigoDuplicado se o código já existir, ou erro genérico de banco.
func (r *postgresProdutoRepository) Create(ctx context.Context, p *model.Produto) error {
	const query = `
		INSERT INTO produtos (codigo, descricao, saldo)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query, p.Codigo, p.Descricao, p.Saldo).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqUniqueViolation {
			return apierror.ErrCodigoDuplicado
		}
		return apierror.Wrap(apierror.CodeInterno, "erro ao criar produto", err)
	}
	return nil
}

// FindByCodigo busca um produto pelo código de negócio.
// Entrada: código do produto.
// Saída: produto encontrado, ou apierror.ErrProdutoNaoEncontrado.
func (r *postgresProdutoRepository) FindByCodigo(ctx context.Context, codigo string) (*model.Produto, error) {
	const query = `SELECT id, codigo, descricao, saldo, created_at, updated_at FROM produtos WHERE codigo = $1`

	var produto model.Produto
	err := r.db.GetContext(ctx, &produto, query, codigo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierror.ErrProdutoNaoEncontrado
		}
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao buscar produto", err)
	}
	return &produto, nil
}

// FindAll lista todos os produtos cadastrados, ordenados por código.
func (r *postgresProdutoRepository) FindAll(ctx context.Context) ([]model.Produto, error) {
	const query = `SELECT id, codigo, descricao, saldo, created_at, updated_at FROM produtos ORDER BY codigo`

	var produtos []model.Produto
	if err := r.db.SelectContext(ctx, &produtos, query); err != nil {
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao listar produtos", err)
	}
	return produtos, nil
}

// AtualizarSaldo aplica um delta (positivo ou negativo) ao saldo do produto,
// dentro de uma transação com lock pessimista (SELECT ... FOR UPDATE).
//
// O lock é o que impede a condição de corrida descrita no requisito opcional
// de concorrência: se duas notas tentarem dar baixa no mesmo produto ao
// mesmo tempo, a segunda transação fica bloqueada esperando a primeira
// commitar, e só então lê o saldo já atualizado — evitando que ambas leiam
// o mesmo saldo "1" e as duas decrementem com sucesso.
//
// Entrada: código do produto e delta (negativo para baixa, positivo para estorno).
// Saída: produto com saldo atualizado, ou apierror.ErrSaldoInsuficiente /
// apierror.ErrProdutoNaoEncontrado.
func (r *postgresProdutoRepository) AtualizarSaldo(ctx context.Context, codigo string, delta int) (*model.Produto, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao iniciar transação", err)
	}
	// Se algo dentro da função retornar antes do Commit, o Rollback garante
	// que a transação não fique pendurada. Chamar Rollback após um Commit
	// bem-sucedido é seguro (retorna sql.ErrTxDone, que ignoramos aqui).
	defer func() { _ = tx.Rollback() }()

	var produto model.Produto
	const selectForUpdate = `
		SELECT id, codigo, descricao, saldo, created_at, updated_at
		FROM produtos
		WHERE codigo = $1
		FOR UPDATE
	`
	if err := tx.GetContext(ctx, &produto, selectForUpdate, codigo); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apierror.ErrProdutoNaoEncontrado
		}
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao buscar produto para atualização", err)
	}

	novoSaldo := produto.Saldo + delta
	if novoSaldo < 0 {
		return nil, apierror.ErrSaldoInsuficiente(codigo)
	}

	const update = `UPDATE produtos SET saldo = $1, updated_at = now() WHERE codigo = $2 RETURNING updated_at`
	if err := tx.QueryRowContext(ctx, update, novoSaldo, codigo).Scan(&produto.UpdatedAt); err != nil {
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao atualizar saldo", err)
	}
	produto.Saldo = novoSaldo

	if err := tx.Commit(); err != nil {
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao commitar transação", err)
	}

	return &produto, nil
}

// BuscarIdempotencyKey verifica se uma chave de idempotência já foi processada.
// Entrada: chave (normalmente vinda do header Idempotency-Key).
// Saída: (resposta salva em JSON, true) se a chave já existe; ("", false, nil) se não existe.
func (r *postgresProdutoRepository) BuscarIdempotencyKey(ctx context.Context, chave string) (string, bool, error) {
	if chave == "" {
		return "", false, nil
	}

	const query = `SELECT resposta FROM idempotency_keys WHERE chave = $1`
	var resposta string
	err := r.db.GetContext(ctx, &resposta, query, chave)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("erro ao consultar idempotency key: %w", err)
	}
	return resposta, true, nil
}

// SalvarIdempotencyKey grava o resultado de uma operação associada a uma chave,
// para que retentativas com a mesma chave devolvam a resposta já processada
// em vez de reprocessar a baixa de estoque.
func (r *postgresProdutoRepository) SalvarIdempotencyKey(ctx context.Context, chave string, respostaJSON string) error {
	if chave == "" {
		return nil
	}

	const query = `
		INSERT INTO idempotency_keys (chave, resposta)
		VALUES ($1, $2)
		ON CONFLICT (chave) DO NOTHING
	`
	if _, err := r.db.ExecContext(ctx, query, chave, respostaJSON); err != nil {
		return fmt.Errorf("erro ao salvar idempotency key: %w", err)
	}
	return nil
}
