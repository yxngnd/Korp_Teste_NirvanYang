// Package repository é a única camada que executa SQL do faturamento-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/candidato/faturamento-service/internal/apierror"
	"github.com/candidato/faturamento-service/internal/model"
	"github.com/jmoiron/sqlx"
)

// NotaRepository define o contrato de persistência do domínio de notas fiscais.
type NotaRepository interface {
	CriarNota(ctx context.Context, itens []model.NotaItem) (*model.NotaFiscal, []model.NotaItem, error)
	FindByNumero(ctx context.Context, numero int) (*model.NotaFiscal, []model.NotaItem, error)
	FindAll(ctx context.Context) ([]model.NotaFiscal, error)
	AtualizarStatus(ctx context.Context, numero int, status model.StatusNota) error
	BuscarIdempotencyKey(ctx context.Context, chave string) (string, bool, error)
	SalvarIdempotencyKey(ctx context.Context, chave string, respostaJSON string) error
}

type postgresNotaRepository struct {
	db *sqlx.DB
}

// NewNotaRepository constrói a implementação Postgres do repository.
func NewNotaRepository(db *sqlx.DB) NotaRepository {
	return &postgresNotaRepository{db: db}
}

// CriarNota insere a nota (status Aberta, número sequencial automático via
// DEFAULT nextval(...) definido na migration) e todos os seus itens, dentro
// de uma única transação: ou tudo é gravado, ou nada é.
//
// Entrada: contexto e a lista de itens (sem NotaID preenchido ainda).
// Saída: a nota criada (já com número e ID) e os itens (já com NotaID e ID),
// ou erro.
func (r *postgresNotaRepository) CriarNota(ctx context.Context, itens []model.NotaItem) (*model.NotaFiscal, []model.NotaItem, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao iniciar transação", err)
	}
	defer func() { _ = tx.Rollback() }()

	var nota model.NotaFiscal
	const insertNota = `
		INSERT INTO notas_fiscais DEFAULT VALUES
		RETURNING id, numero, status, created_at, fechada_em
	`
	if err := tx.QueryRowxContext(ctx, insertNota).StructScan(&nota); err != nil {
		return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao criar nota fiscal", err)
	}

	const insertItem = `
		INSERT INTO nota_itens (nota_id, produto_codigo, quantidade)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	itensCriados := make([]model.NotaItem, 0, len(itens))
	for _, item := range itens {
		item.NotaID = nota.ID
		if err := tx.QueryRowContext(ctx, insertItem, item.NotaID, item.ProdutoCodigo, item.Quantidade).Scan(&item.ID); err != nil {
			return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao criar item da nota fiscal", err)
		}
		itensCriados = append(itensCriados, item)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao commitar criação da nota", err)
	}

	return &nota, itensCriados, nil
}

// FindByNumero busca uma nota fiscal e seus itens pelo número sequencial.
func (r *postgresNotaRepository) FindByNumero(ctx context.Context, numero int) (*model.NotaFiscal, []model.NotaItem, error) {
	var nota model.NotaFiscal
	const queryNota = `SELECT id, numero, status, created_at, fechada_em FROM notas_fiscais WHERE numero = $1`
	if err := r.db.GetContext(ctx, &nota, queryNota, numero); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, apierror.ErrNotaNaoEncontrada
		}
		return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao buscar nota fiscal", err)
	}

	var itens []model.NotaItem
	const queryItens = `SELECT id, nota_id, produto_codigo, quantidade FROM nota_itens WHERE nota_id = $1 ORDER BY id`
	if err := r.db.SelectContext(ctx, &itens, queryItens, nota.ID); err != nil {
		return nil, nil, apierror.Wrap(apierror.CodeInterno, "erro ao buscar itens da nota fiscal", err)
	}

	return &nota, itens, nil
}

// FindAll lista todas as notas fiscais (sem os itens — a listagem é usada
// na tela de lista, o detalhe completo fica em FindByNumero).
func (r *postgresNotaRepository) FindAll(ctx context.Context) ([]model.NotaFiscal, error) {
	const query = `SELECT id, numero, status, created_at, fechada_em FROM notas_fiscais ORDER BY numero DESC`

	var notas []model.NotaFiscal
	if err := r.db.SelectContext(ctx, &notas, query); err != nil {
		return nil, apierror.Wrap(apierror.CodeInterno, "erro ao listar notas fiscais", err)
	}
	return notas, nil
}

// AtualizarStatus muda o status da nota e, se for para Fechada, registra o
// timestamp de fechamento.
func (r *postgresNotaRepository) AtualizarStatus(ctx context.Context, numero int, status model.StatusNota) error {
	const query = `
		UPDATE notas_fiscais
		SET status = $1, fechada_em = CASE WHEN $2 = 'Fechada' THEN now() ELSE fechada_em END
		WHERE numero = $3
	`
	result, err := r.db.ExecContext(ctx, query, status, status, numero)
	if err != nil {
		return apierror.Wrap(apierror.CodeInterno, "erro ao atualizar status da nota fiscal", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return apierror.Wrap(apierror.CodeInterno, "erro ao verificar atualização de status", err)
	}
	if rows == 0 {
		return apierror.ErrNotaNaoEncontrada
	}
	return nil
}

// BuscarIdempotencyKey verifica se uma chave de idempotência já foi processada.
func (r *postgresNotaRepository) BuscarIdempotencyKey(ctx context.Context, chave string) (string, bool, error) {
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

// SalvarIdempotencyKey grava o resultado de uma operação associada a uma chave.
func (r *postgresNotaRepository) SalvarIdempotencyKey(ctx context.Context, chave string, respostaJSON string) error {
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
