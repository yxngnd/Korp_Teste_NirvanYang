package model

// NotaItem representa a linha da tabela `nota_itens`.
//
// ProdutoCodigo não é uma foreign key de verdade: o produto vive no banco
// do estoque-service, outro serviço, outro banco de dados. A consistência
// entre os dois é garantida pela chamada HTTP feita no momento da impressão
// (client/estoque_client.go), não por uma constraint de banco — essa é a
// natureza de uma arquitetura "database per service".
type NotaItem struct {
	ID            int64  `db:"id"`
	NotaID        int64  `db:"nota_id"`
	ProdutoCodigo string `db:"produto_codigo"`
	Quantidade    int    `db:"quantidade"`
}
