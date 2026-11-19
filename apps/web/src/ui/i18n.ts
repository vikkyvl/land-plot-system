import { createI18n } from 'vue-i18n'

import type { Locale } from '../domain/locale'

import en from './locales/en.json'
import uk from './locales/uk.json'

const intlLocales: Readonly<Record<Locale, string>> = { uk: 'uk-UA', en: 'en-GB' }

export function ukrainianPlural(choice: number, choicesLength: number): number {
  const lastTwo = Math.abs(choice) % 100
  const last = lastTwo % 10
  if (last === 1 && lastTwo !== 11) {
    return 0
  }
  if (last >= 2 && last <= 4 && (lastTwo < 12 || lastTwo > 14)) {
    return Math.min(1, choicesLength - 1)
  }

  return Math.min(2, choicesLength - 1)
}

export const i18n = createI18n({
  legacy: false,
  locale: 'uk' satisfies Locale,
  fallbackLocale: 'en' satisfies Locale,
  messages: { uk, en },
  pluralRules: { uk: ukrainianPlural },
})

export const t = i18n.global.t

export function currentLocale(): Locale {
  return i18n.global.locale.value as Locale
}

export function intlLocale(): string {
  return intlLocales[currentLocale()]
}

export function applyLocale(locale: Locale): void {
  i18n.global.locale.value = locale
  if (typeof document !== 'undefined') {
    document.documentElement.lang = locale
  }
}
