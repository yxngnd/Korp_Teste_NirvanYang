import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIconModule } from '@angular/material/icon';
import { MatInputModule } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatSnackBar } from '@angular/material/snack-bar';
import { finalize } from 'rxjs';
import { EstoqueService } from '../../../core/services/estoque.service';
import { FaturamentoService } from '../../../core/services/faturamento.service';
import { Produto } from '../../../core/models/produto.model';

// Formulário de criação de nota fiscal, com uma lista dinâmica de itens
// (FormArray) — cada item escolhe um produto já cadastrado e uma quantidade.
@Component({
  selector: 'app-nota-form',
  standalone: true,
  imports: [
    CommonModule,
    ReactiveFormsModule,
    RouterLink,
    MatButtonModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSelectModule,
  ],
  templateUrl: './nota-form.component.html',
  styleUrl: './nota-form.component.css',
})
export class NotaFormComponent implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly estoqueService = inject(EstoqueService);
  private readonly faturamentoService = inject(FaturamentoService);
  private readonly snackBar = inject(MatSnackBar);
  private readonly router = inject(Router);

  produtosDisponiveis: Produto[] = [];
  salvando = false;

  form = this.fb.group({
    itens: this.fb.array([this.criarItemForm()]),
  });

  get itens() {
    return this.form.controls.itens;
  }

  // ngOnInit carrega a lista de produtos disponíveis para popular o
  // <mat-select> de cada linha de item.
  ngOnInit(): void {
    this.estoqueService.listarProdutos().subscribe({
      next: (produtos) => (this.produtosDisponiveis = produtos),
      error: () => {},
    });
  }

  private criarItemForm() {
    return this.fb.nonNullable.group({
      produtoCodigo: ['', Validators.required],
      quantidade: [1, [Validators.required, Validators.min(1)]],
    });
  }

  adicionarItem(): void {
    this.itens.push(this.criarItemForm());
  }

  removerItem(index: number): void {
    if (this.itens.length > 1) {
      this.itens.removeAt(index);
    }
  }

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.salvando = true;
    const request = { itens: this.itens.getRawValue() };

    this.faturamentoService
      .criarNota(request)
      .pipe(finalize(() => (this.salvando = false)))
      .subscribe({
        next: (nota) => {
          this.snackBar.open(`Nota fiscal ${nota.numero} criada com sucesso.`, 'Fechar', { duration: 3000 });
          this.router.navigate(['/notas']);
        },
        error: () => {},
      });
  }
}
