import { createI18n } from 'vue-i18n'
import { en as quotaPkgEn, zh as quotaPkgZh } from '@quotapanel/quota/src/locales'
import { messages } from '../locales'

export const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  fallbackLocale: 'en',
  messages,
})

i18n.global.mergeLocaleMessage('zh', quotaPkgZh)
i18n.global.mergeLocaleMessage('en', quotaPkgEn)
