import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { FaturamentoService } from '../../../core/services/faturamento.service';
import { NotaFiscal } from '../../../core/models/nota-fiscal.model';

// Tela de detalhe de uma nota fiscal específica, acessada via /notas/:numero.
@Component({
  selector: 'app-nota-detail',
  standalone: true,
  imports: [CommonModule, RouterLink, MatButtonModule, MatProgressSpinnerModule, MatTableModule],
  templateUrl: './nota-detail.component.html',
  styleUrl: './nota-detail.component.css',
})
export class NotaDetailComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly faturamentoService = inject(FaturamentoService);

  nota: NotaFiscal | null = null;
  carregando = false;
  colunas = ['produtoCodigo', 'quantidade'];

  // ngOnInit lê o parâmetro "numero" da rota atual e busca a nota
  // correspondente. Como a tela é sempre montada de novo ao navegar para
  // um número diferente (não há reuso do mesmo componente entre notas),
  // não é necessário assinar mudanças de paramMap — snapshot é suficiente.
  ngOnInit(): void {
    const numero = Number(this.route.snapshot.paramMap.get('numero'));
    this.carregando = true;
    this.faturamentoService.buscarNota(numero).subscribe({
      next: (nota) => {
        this.nota = nota;
        this.carregando = false;
      },
      error: () => {
        this.carregando = false;
      },
    });
  }
}
