// Interfaces do domínio de notas fiscais, espelhando o contrato JSON
// exposto pelo faturamento-service (internal/dto/nota_dto.go).

export type StatusNota = 'Aberta' | 'Fechada';

export interface NotaItem {
  produtoCodigo: string;
  quantidade: number;
}

export interface NotaFiscal {
  numero: number;
  status: StatusNota;
  itens: NotaItem[];
}

export interface CriarNotaRequest {
  itens: NotaItem[];
}
