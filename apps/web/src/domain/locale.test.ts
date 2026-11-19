import { describe, expect, it } from 'vitest'

import { isLocale, localeFromLanguage } from './locale'

describe('locale', () => {
  it.each([
    ['uk', 'uk'],
    ['uk-UA', 'uk'],
    ['UK', 'uk'],
    ['en-US', 'en'],
    ['de-DE', 'en'],
    ['ru-RU', 'en'],
    [undefined, 'en'],
  ])('picks %s → %s', (language, locale) => {
    expect(localeFromLanguage(language)).toBe(locale)
  })

  it('knows the two supported languages', () => {
    expect(isLocale('uk')).toBe(true)
    expect(isLocale('en')).toBe(true)
    expect(isLocale('de')).toBe(false)
  })
})
