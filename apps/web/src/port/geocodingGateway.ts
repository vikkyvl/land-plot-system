import type { Place } from '../domain/place'

export interface GeocodingGateway {
  search(query: string, signal?: AbortSignal): Promise<Place[]>
}
