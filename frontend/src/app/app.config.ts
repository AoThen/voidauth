import { type ApplicationConfig, provideZoneChangeDetection, importProvidersFrom } from '@angular/core'
import { provideRouter } from '@angular/router'
import { routes } from './app.routes'
import { provideHttpClient, withInterceptors, type HttpInterceptorFn } from '@angular/common/http'
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async'
import { HttpClient, provideHttpClient } from '@angular/common/http'
import { TranslateModule, TranslateLoader } from '@ngx-translate/core'
import { TranslateHttpLoader } from '@ngx-translate/http-loader'
import { getBaseHrefPath } from './services/config.service'

export function HttpLoaderFactory(http: HttpClient) {
  return new TranslateHttpLoader(http, `${getBaseHrefPath()}assets/i18n/`, '.json')
}

const baseHrefInterceptor: HttpInterceptorFn = (req, next) => {
  // Skip external URLs
  if (req.url.startsWith('http://') || req.url.startsWith('https://')) {
    return next(req)
  }

  // Clone and modify request
  const modifiedReq = req.clone({
    url: `${getBaseHrefPath()}${req.url}`,
  })

  // Proceed with modified request
  return next(modifiedReq)
}

export const appConfig: ApplicationConfig = {
  providers: [
    provideZoneChangeDetection({ eventCoalescing: true }),
    provideRouter(routes),
    provideHttpClient(
      withInterceptors([baseHrefInterceptor]),
    ),
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    provideAnimationsAsync(),
    importProvidersFrom(
      TranslateModule.forRoot({
        loader: {
          provide: TranslateLoader,
          useFactory: HttpLoaderFactory,
          deps: [HttpClient],
        },
      }),
    ),
  ],
}
