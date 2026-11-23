import type { Place } from '../domain/place'
import type { GeocodingGateway } from '../port/geocodingGateway'

export const minimumQueryLength = 3

export class PlaceSearchService {
  private readonly geocoding: GeocodingGateway

  constructor(geocoding: GeocodingGateway) {
    this.geocoding = geocoding
  }

  search(query: string, signal?: AbortSignal): Promise<Place[]> {
    const trimmed = query.trim()
    if (trimmed.length < minimumQueryLength) {
      return Promise.resolve([])
    }

    return this.geocoding.search(trimmed, signal)
  }
}
