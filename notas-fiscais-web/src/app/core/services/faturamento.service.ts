import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CriarNotaRequest, NotaFiscal } from '../models/nota-fiscal.model';

// FaturamentoService encapsula toda a comunicação HTTP com o faturamento-service.
@Injectable({ providedIn: 'root' })
export class FaturamentoService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiFaturamentoUrl}/notas`;

  listarNotas(): Observable<NotaFiscal[]> {
    return this.http.get<NotaFiscal[]>(this.baseUrl);
  }

  criarNota(request: CriarNotaRequest): Observable<NotaFiscal> {
    return this.http.post<NotaFiscal>(this.baseUrl, request);
  }

  buscarNota(numero: number): Observable<NotaFiscal> {
    return this.http.get<NotaFiscal>(`${this.baseUrl}/${numero}`);
  }

  // imprimirNota gera uma chave de idempotência nova a cada chamada
  // (crypto.randomUUID) e a envia no header Idempotency-Key. Isso protege
  // contra duplo clique no botão: se o usuário clicar duas vezes rápido
  // antes do loading desabilitar o botão, cada clique gera uma chave
  // diferente por interação do usuário — mas dentro de uma mesma chamada,
  // se o navegador reenviar a requisição (retry automático), a chave
  // permanece igual e o backend não reprocessa a baixa duas vezes.
  imprimirNota(numero: number): Observable<NotaFiscal> {
    const idempotencyKey = crypto.randomUUID();
    const headers = new HttpHeaders({ 'Idempotency-Key': idempotencyKey });
    return this.http.post<NotaFiscal>(`${this.baseUrl}/${numero}/imprimir`, {}, { headers });
  }
}
