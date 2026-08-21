import { ApplicationConfig, provideZoneChangeDetection } from '@angular/core';
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { provideRouter } from '@angular/router';
import { routes } from './app.routes';
import { errorInterceptor } from './core/interceptors/error.interceptor';

// appConfig registra os providers globais da aplicação:
// - provideRouter: habilita a navegação entre as telas (app.routes.ts).
// - provideHttpClient + withInterceptors: liga o errorInterceptor a toda
//   chamada HTTP feita pela aplicação, sem precisar registrar em cada service.
// - provideAnimationsAsync: necessário para os componentes do Angular
//   Material (spinner, snackbar, etc.) funcionarem com suas transições.
export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(withInterceptors([errorInterceptor])),
    provideAnimationsAsync(),
  ],
};
