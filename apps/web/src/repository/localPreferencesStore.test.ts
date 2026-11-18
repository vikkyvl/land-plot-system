import { afterEach, describe, expect, it } from 'vitest'

import { LocalPreferencesStore } from './localPreferencesStore'

afterEach(() => localStorage.clear())

describe('LocalPreferencesStore', () => {
  it('remembers the map layer', () => {
    new LocalPreferencesStore(localStorage).saveMapLayer('satellite')

    expect(new LocalPreferencesStore(localStorage).mapLayer()).toBe('satellite')
  })

  it('ignores unknown values and broken storage', () => {
    localStorage.setItem('arqetra.mapLayer', 'terrain')
    const broken = {
      getItem: () => {
        throw new Error('denied')
      },
      setItem: () => {
        throw new Error('denied')
      },
    } as unknown as Storage

    expect(new LocalPreferencesStore(localStorage).mapLayer()).toBeNull()
    expect(new LocalPreferencesStore(broken).mapLayer()).toBeNull()
    expect(() => new LocalPreferencesStore(broken).saveMapLayer('map')).not.toThrow()
  })
})
