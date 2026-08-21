import { Routes } from '@angular/router';

// Rotas com lazy loading (loadComponent): cada tela só é baixada quando o
// usuário navega até ela, mantendo o bundle inicial pequeno.
export const routes: Routes = [
  { path: '', redirectTo: 'produtos', pathMatch: 'full' },
  {
    path: 'produtos',
    loadComponent: () =>
      import('./features/produtos/produto-list/produto-list.component').then((m) => m.ProdutoListComponent),
  },
  {
    path: 'notas',
    loadComponent: () =>
      import('./features/notas-fiscais/nota-list/nota-list.component').then((m) => m.NotaListComponent),
  },
  {
    path: 'notas/novo',
    loadComponent: () =>
      import('./features/notas-fiscais/nota-form/nota-form.component').then((m) => m.NotaFormComponent),
  },
  {
    path: 'notas/:numero',
    loadComponent: () =>
      import('./features/notas-fiscais/nota-detail/nota-detail.component').then((m) => m.NotaDetailComponent),
  },
];
