import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { EstoqueService } from '../../../core/services/estoque.service';
import { Produto } from '../../../core/models/produto.model';
import { ProdutoFormComponent } from '../produto-form/produto-form.component';

// Tela de listagem de produtos. Traz também o formulário de criação embutido
// (sem rota própria), já que o cadastro de produto é simples o suficiente
// para não precisar de uma tela separada.
@Component({
  selector: 'app-produto-list',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MatButtonModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTableModule,
    ProdutoFormComponent,
  ],
  templateUrl: './produto-list.component.html',
  styleUrl: './produto-list.component.css',
})
export class ProdutoListComponent implements OnInit {
  private readonly estoqueService = inject(EstoqueService);

  produtos: Produto[] = [];
  carregando = false;
  colunas = ['codigo', 'descricao', 'saldo'];

  // ngOnInit carrega a lista de produtos assim que a tela é montada.
  ngOnInit(): void {
    this.carregarProdutos();
  }

  carregarProdutos(): void {
    this.carregando = true;
    this.estoqueService.listarProdutos().subscribe({
      next: (produtos) => {
        this.produtos = produtos;
        this.carregando = false;
      },
      error: () => {
        // O interceptor global já mostra o snackbar com a mensagem de erro;
        // aqui só precisamos garantir que o spinner de carregamento pare.
        this.carregando = false;
      },
    });
  }

  // Chamado pelo produto-form quando um produto é criado com sucesso,
  // para recarregar a lista sem precisar dar F5 na página.
  onProdutoCriado(): void {
    this.carregarProdutos();
  }
}
