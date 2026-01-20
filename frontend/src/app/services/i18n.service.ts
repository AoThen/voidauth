import { Injectable } from '@angular/core'
import { TranslateService } from '@ngx-translate/core'

@Injectable({
  providedIn: 'root',
})
export class I18nService {
  private translationsLoaded = new Set<string>()

  constructor(private translate: TranslateService) {
    this.translate.addLangs(['en', 'zh'])
    const savedLang = localStorage.getItem('lang')
    const browserLang = this.translate.getBrowserLang() ?? 'en'
    const defaultLang = savedLang || (['en', 'zh'].includes(browserLang) ? browserLang : 'en')
    this.translate.use(defaultLang)
    this.preloadTranslations(defaultLang)
  }

  private preloadTranslations(lang: string) {
    if (!this.translationsLoaded.has(lang)) {
      this.translate.getTranslation(lang).subscribe()
      this.translationsLoaded.add(lang)
    }
  }

  setLang(lang: 'en' | 'zh') {
    this.translate.use(lang)
    localStorage.setItem('lang', lang)
    this.preloadTranslations(lang)
  }

  getLang(): 'en' | 'zh' {
    return (localStorage.getItem('lang') as 'en' | 'zh') || 'en'
  }

  instant(key: string, params?: Record<string, unknown>) {
    return this.translate.instant(key, params)
  }
}
