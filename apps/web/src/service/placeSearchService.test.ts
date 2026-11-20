import { describe, expect, it, vi } from 'vitest'

import { PlaceSearchService } from './placeSearchService'

describe('PlaceSearchService', () => {
  it('searches from three characters and trims the query', async () => {
    const geocoding = { search: vi.fn(async () => []) }
    const service = new PlaceSearchService(geocoding)

    expect(await service.search(' Бі ')).toEqual([])
    expect(geocoding.search).not.toHaveBeenCalled()

    await service.search('  Біла ')
    expect(geocoding.search).toHaveBeenCalledWith('Біла', undefined)
  })
})
