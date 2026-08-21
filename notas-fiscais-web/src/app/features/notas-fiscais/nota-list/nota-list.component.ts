import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatIconModule } from '@angular/material/icon';
import { MatProgressSpinnerModule } from '@angular/material/progress-spinner';
import { MatTableModule } from '@angular/material/table';
import { finalize } from 'rxjs';
import { FaturamentoService } from '../../../core/services/faturamento.service';
import { NotaFiscal } from '../../../core/models/nota-fiscal.model';

// Tela de listagem de notas fiscais, com o botão de impressão por linha —
// é aqui que o requisito "botão de impressão visível e intuitivo" é atendido.
@Component({
  selector: 'app-nota-list',
  standalone: true,
  imports: [
    CommonModule,
    RouterLink,
    MatButtonModule,
    MatChipsModule,
    MatIconModule,
    MatProgressSpinnerModule,
    MatTableModule,
  ],
  templateUrl: './nota-list.component.html',
  styleUrl: './nota-list.component.css',
})
export class NotaListComponent implements OnInit {
  private readonly faturamentoService = inject(FaturamentoService);

  notas: NotaFiscal[] = [];
  carregando = false;
  colunas = ['numero', 'status', 'acoes'];

  // Controla o estado de "imprimindo" por nota individualmente (não um
  // único booleano global), assim só o botão da nota clicada mostra o
  // spinner, sem travar a tela inteira.
  imprimindo = new Set<number>();

  ngOnInit(): void {
    this.carregarNotas();
  }

  carregarNotas(): void {
    this.carregando = true;
    this.faturamentoService.listarNotas().subscribe({
      next: (notas) => {
        this.notas = notas;
        this.carregando = false;
      },
      error: () => {
        this.carregando = false;
      },
    });
  }

  imprimir(nota: NotaFiscal): void {
    this.imprimindo.add(nota.numero);
    this.faturamentoService
      .imprimirNota(nota.numero)
      .pipe(finalize(() => this.imprimindo.delete(nota.numero)))
      .subscribe({
        // Atualiza o status da nota diretamente na lista já carregada, sem
        // precisar refazer a chamada de listagem inteira.
        next: (notaAtualizada) => {
          nota.status = notaAtualizada.status;
        },
        // erro (nota já fechada, saldo insuficiente, estoque indisponível)
        // já é exibido pelo interceptor global.
        error: () => {},
      });
  }
}
