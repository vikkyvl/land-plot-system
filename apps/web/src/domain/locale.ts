export const locales = ['uk', 'en'] as const

export type Locale = (typeof locales)[number]

export const fallbackLocale: Locale = 'en'

export function isLocale(value: unknown): value is Locale {
  return locales.some((locale) => locale === value)
}

export function localeFromLanguage(language: string | undefined): Locale {
  return language?.toLowerCase().startsWith('uk') ? 'uk' : fallbackLocale
}
