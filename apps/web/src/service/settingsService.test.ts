import { describe, expect, it, vi } from 'vitest'

import type { Locale } from '../domain/locale'

import { SettingsService } from './settingsService'

function store(locale: Locale | null = null) {
  return { mapLayer: vi.fn(() => null), saveMapLayer: vi.fn(), locale: vi.fn(() => locale), saveLocale: vi.fn() }
}

describe('SettingsService', () => {
  it('uses the map layer by default and remembers the choice', () => {
    const settings = store()
    const service = new SettingsService(settings, 'uk-UA')

    expect(service.layer()).toBe('map')
    service.chooseLayer('satellite')
    expect(settings.saveMapLayer).toHaveBeenCalledWith('satellite')
  })

  it('takes the language from the browser until the user chooses one', () => {
    expect(new SettingsService(store(), 'uk-UA').locale()).toBe('uk')
    expect(new SettingsService(store(), 'de-DE').locale()).toBe('en')
    expect(new SettingsService(store('uk'), 'en-US').locale()).toBe('uk')
  })

  it('remembers the chosen language', () => {
    const settings = store()

    new SettingsService(settings, 'uk').chooseLocale('en')

    expect(settings.saveLocale).toHaveBeenCalledWith('en')
  })
})
