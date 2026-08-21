// Interfaces do domínio de produtos, espelhando o contrato JSON exposto
// pelo estoque-service (internal/dto/produto_dto.go).

export interface Produto {
  codigo: string;
  descricao: string;
  saldo: number;
}

export interface CreateProdutoRequest {
  codigo: string;
  descricao: string;
  saldo: number;
}
