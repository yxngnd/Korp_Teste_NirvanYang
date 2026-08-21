// Package client contém os clients HTTP usados para comunicação entre
// microsserviços. Hoje só existe o client do estoque-service, mas o pacote
// já fica isolado para caso surjam outras integrações no futuro.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/candidato/faturamento-service/internal/apierror"
	"github.com/sony/gobreaker"
)

// operacaoRequest espelha o dto.AtualizarSaldoRequest do estoque-service.
// Não importamos o pacote do outro serviço propositalmente — cada
// microsserviço é um módulo Go independente, o contrato entre eles é o
// JSON trafegado via HTTP, não um tipo Go compartilhado.
type operacaoRequest struct {
	Quantidade int    `json:"quantidade"`
	Operacao   string `json:"operacao"`
}

// errorResponse espelha o formato de erro padronizado devolvido pelo
// middleware de erro do estoque-service: {"error": "...", "code": "..."}.
type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// EstoqueClient encapsula toda a comunicação com o estoque-service,
// protegida por um circuit breaker.
type EstoqueClient struct {
	baseURL    string
	httpClient *http.Client
	breaker    *gobreaker.CircuitBreaker
}

// NewEstoqueClient constrói o client já configurado com timeout e circuit breaker.
//
// Configuração do breaker:
//   - Name: identifica o breaker nos logs/callbacks.
//   - MaxRequests: quantas chamadas de teste são permitidas no estado
//     half-open antes de decidir se fecha o circuito de novo.
//   - Interval: janela de tempo em que as contagens de falha são resetadas
//     enquanto o circuito está fechado (0 = nunca reseta sozinho, só conta
//     desde a última abertura).
//   - Timeout: quanto tempo o circuito fica aberto antes de tentar
//     half-open de novo.
//   - ReadyToTrip: função que decide quando abrir o circuito — aqui, a
//     partir de 5 requisições consecutivas com falha.
func NewEstoqueClient(baseURL string) *EstoqueClient {
	settings := gobreaker.Settings{
		Name:        "estoque-service",
		MaxRequests: 1,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}

	return &EstoqueClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			// Timeout curto: se o estoque-service não responder rápido,
			// falha rápido também — é isso que alimenta o contador de
			// falhas consecutivas do breaker.
			Timeout: 3 * time.Second,
		},
		breaker: gobreaker.NewCircuitBreaker(settings),
	}
}

// BaixarEstoque decrementa o saldo de um produto no estoque-service.
// Chamado pelo service ao imprimir uma nota fiscal, uma vez por item.
//
// Entrada: contexto, código do produto, quantidade a decrementar, e uma
// chave de idempotência opcional (repassada como header Idempotency-Key).
// Saída: erro — apierror.ErrEstoqueIndisponivel se o breaker estiver aberto
// ou a chamada falhar por timeout/rede; apierror.ErrSaldoInsuficiente ou
// apierror.ErrProdutoNaoEncontrado se o estoque-service responder com esse
// erro de negócio; nil em caso de sucesso.
func (c *EstoqueClient) BaixarEstoque(ctx context.Context, codigoProduto string, quantidade int, idempotencyKey string) error {
	return c.atualizarSaldo(ctx, codigoProduto, quantidade, "baixa", idempotencyKey)
}

// EstornarEstoque devolve o saldo de um produto no estoque-service.
// Usado como compensação quando a impressão de uma nota falha no meio do
// processo (alguns itens já tiveram baixa, outros não) — ver
// service.NotaService.ImprimirNota para o fluxo completo.
//
// Entrada: mesmos parâmetros de BaixarEstoque.
// Saída: erro nas mesmas condições, exceto ErrSaldoInsuficiente (estorno
// sempre aumenta o saldo, nunca é bloqueado por saldo).
func (c *EstoqueClient) EstornarEstoque(ctx context.Context, codigoProduto string, quantidade int, idempotencyKey string) error {
	return c.atualizarSaldo(ctx, codigoProduto, quantidade, "estorno", idempotencyKey)
}

func (c *EstoqueClient) atualizarSaldo(ctx context.Context, codigoProduto string, quantidade int, operacao string, idempotencyKey string) error {
	// businessErr guarda um erro de negócio (ex: saldo insuficiente) vindo
	// de uma resposta HTTP que o estoque-service respondeu normalmente.
	// Importante: nesse caso a função interna retorna (nil, nil) para o
	// breaker.Execute, porque do ponto de vista de infraestrutura a chamada
	// TEVE SUCESSO — o serviço respondeu, só que com uma regra de negócio
	// que bloqueou a operação. Se contássemos isso como falha, o circuito
	// abriria por causa de saldo insuficiente legítimo, não por o estoque
	// estar fora do ar — e é exatamente esse tipo de falso positivo que o
	// circuit breaker precisa evitar.
	var businessErr *apierror.APIError

	// A chamada real acontece dentro de breaker.Execute. Se o circuito
	// estiver aberto, Execute nem chega a rodar a função — retorna
	// gobreaker.ErrOpenState imediatamente, sem tentar rede.
	_, err := c.breaker.Execute(func() (interface{}, error) {
		callErr := c.doAtualizarSaldo(ctx, codigoProduto, quantidade, operacao, idempotencyKey)

		var apiErr *apierror.APIError
		if errors.As(callErr, &apiErr) {
			businessErr = apiErr
			return nil, nil // sucesso de infraestrutura, não conta como falha do breaker
		}

		return nil, callErr // erro de rede/timeout/5xx — este sim conta como falha
	})

	if businessErr != nil {
		return businessErr
	}
	if err == nil {
		return nil
	}

	if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
		return apierror.ErrEstoqueIndisponivel
	}

	// Qualquer outro erro (timeout, conexão recusada, 5xx do estoque, etc.)
	// é tratado como indisponibilidade do estoque.
	return apierror.ErrEstoqueIndisponivel
}

func (c *EstoqueClient) doAtualizarSaldo(ctx context.Context, codigoProduto string, quantidade int, operacao string, idempotencyKey string) error {
	body, err := json.Marshal(operacaoRequest{Quantidade: quantidade, Operacao: operacao})
	if err != nil {
		return fmt.Errorf("erro ao serializar request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/produtos/%s/saldo", c.baseURL, codigoProduto)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao montar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Erro de rede (timeout, conexão recusada) — é exatamente o que
		// deve contar como falha para o circuit breaker.
		return fmt.Errorf("erro ao chamar estoque-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	var errResp errorResponse
	_ = json.Unmarshal(respBody, &errResp)

	switch {
	case resp.StatusCode == http.StatusConflict && errResp.Code == "SALDO_INSUFICIENTE":
		return apierror.ErrSaldoInsuficiente(codigoProduto)
	case resp.StatusCode == http.StatusNotFound:
		return apierror.ErrProdutoNaoEncontrado(codigoProduto)
	default:
		// Qualquer outro status (erro interno do estoque-service, por
		// exemplo) é tratado como indisponibilidade — não é um erro de
		// negócio conhecido, então também alimenta o contador do breaker.
		return fmt.Errorf("estoque-service respondeu status %d: %s", resp.StatusCode, string(respBody))
	}
}
