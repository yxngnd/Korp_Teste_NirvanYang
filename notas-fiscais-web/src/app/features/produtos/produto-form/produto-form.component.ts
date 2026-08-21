import { CommonModule } from '@angular/common';
import { Component, EventEmitter, OnInit, Output, inject } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { MatButtonModule } from '@angular/material/button';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatInputModule } from '@angular/material/input';
import { MatSnackBar } from '@angular/material/snack-bar';
import { finalize } from 'rxjs';
import { EstoqueService } from '../../../core/services/estoque.service';

// Formulário de cadastro de produto. Usa Reactive Forms (FormBuilder) em
// vez de Template-driven, o padrão recomendado para validação mais robusta.
@Component({
  selector: 'app-produto-form',
  standalone: true,
  imports: [CommonModule, ReactiveFormsModule, MatFormFieldModule, MatInputModule, MatButtonModule],
  templateUrl: './produto-form.component.html',
  styleUrl: './produto-form.component.css',
})
export class ProdutoFormComponent implements OnInit {
  private readonly fb = inject(FormBuilder);
  private readonly estoqueService = inject(EstoqueService);
  private readonly snackBar = inject(MatSnackBar);

  // Emite um evento para o componente pai (produto-list) recarregar a
  // lista após a criação — evita que este componente precise conhecer
  // como a lista é exibida.
  @Output() produtoCriado = new EventEmitter<void>();

  salvando = false;

  form = this.fb.nonNullable.group({
    codigo: ['', Validators.required],
    descricao: ['', Validators.required],
    saldo: [0, [Validators.required, Validators.min(0)]],
  });

  // ngOnInit existe aqui só por consistência de ciclo de vida com os
  // demais componentes — o form já é montado na declaração do campo acima,
  // não há nada assíncrono a carregar neste formulário em particular.
  ngOnInit(): void {}

  onSubmit(): void {
    if (this.form.invalid) {
      this.form.markAllAsTouched();
      return;
    }

    this.salvando = true;
    this.estoqueService
      .criarProduto(this.form.getRawValue())
      .pipe(finalize(() => (this.salvando = false)))
      .subscribe({
        next: () => {
          this.snackBar.open('Produto cadastrado com sucesso.', 'Fechar', { duration: 3000 });
          this.form.reset({ codigo: '', descricao: '', saldo: 0 });
          this.produtoCriado.emit();
        },
        // erro já é exibido pelo interceptor global; nada a fazer aqui.
        error: () => {},
      });
  }
}
