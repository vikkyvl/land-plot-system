import { describe, expect, it, vi } from 'vitest'

import { MapPreferencesService } from './mapPreferencesService'

describe('MapPreferencesService', () => {
  it('uses the map layer by default and remembers the choice', () => {
    const preferences = { mapLayer: vi.fn(() => null), saveMapLayer: vi.fn() }
    const service = new MapPreferencesService(preferences)

    expect(service.layer()).toBe('map')
    service.chooseLayer('satellite')
    expect(preferences.saveMapLayer).toHaveBeenCalledWith('satellite')
  })
})
