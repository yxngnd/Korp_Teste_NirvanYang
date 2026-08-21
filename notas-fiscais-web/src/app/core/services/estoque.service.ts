import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { CreateProdutoRequest, Produto } from '../models/produto.model';

// EstoqueService encapsula toda a comunicação HTTP com o estoque-service.
// Os componentes nunca chamam HttpClient diretamente — sempre por aqui,
// o que facilita trocar a URL base ou mockar em testes.
@Injectable({ providedIn: 'root' })
export class EstoqueService {
  private readonly http = inject(HttpClient);
  private readonly baseUrl = `${environment.apiEstoqueUrl}/produtos`;

  listarProdutos(): Observable<Produto[]> {
    return this.http.get<Produto[]>(this.baseUrl);
  }

  criarProduto(request: CreateProdutoRequest): Observable<Produto> {
    return this.http.post<Produto>(this.baseUrl, request);
  }

  buscarProduto(codigo: string): Observable<Produto> {
    return this.http.get<Produto>(`${this.baseUrl}/${codigo}`);
  }
}
