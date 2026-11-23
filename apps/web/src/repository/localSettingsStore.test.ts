import { afterEach, describe, expect, it } from 'vitest'

import { LocalSettingsStore } from './localSettingsStore'

afterEach(() => localStorage.clear())

describe('LocalSettingsStore', () => {
  it('remembers the map layer and the language', () => {
    new LocalSettingsStore(localStorage).saveMapLayer('satellite')
    new LocalSettingsStore(localStorage).saveLocale('en')

    const settings = new LocalSettingsStore(localStorage)
    expect(settings.mapLayer()).toBe('satellite')
    expect(settings.locale()).toBe('en')
  })

  it('ignores unknown values and broken storage', () => {
    localStorage.setItem('arqetra.mapLayer', 'terrain')
    localStorage.setItem('arqetra.locale', 'de')
    const broken = {
      getItem: () => {
        throw new Error('denied')
      },
      setItem: () => {
        throw new Error('denied')
      },
    } as unknown as Storage

    expect(new LocalSettingsStore(localStorage).mapLayer()).toBeNull()
    expect(new LocalSettingsStore(localStorage).locale()).toBeNull()
    expect(new LocalSettingsStore(broken).locale()).toBeNull()
    expect(() => new LocalSettingsStore(broken).saveLocale('uk')).not.toThrow()
  })
})
