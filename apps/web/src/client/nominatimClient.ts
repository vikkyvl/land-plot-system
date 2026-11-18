import { DomainError } from '../domain/error'
import { ErrorCode } from '../domain/errorCode'
import type { Place, PlaceBounds } from '../domain/place'
import type { GeocodingGateway } from '../port/geocodingGateway'

import { networkError } from './errors'

interface NominatimPlace {
  readonly name?: string
  readonly display_name: string
  readonly lat: string
  readonly lon: string
  readonly boundingbox?: readonly [string, string, string, string]
}

export interface NominatimClientOptions {
  readonly baseUrl: string
  readonly fetch?: typeof fetch
  readonly now?: () => number
  readonly wait?: (milliseconds: number) => Promise<void>
}

export const minimumRequestInterval = 1000
const resultLimit = '5'
const language = 'uk'
const countries = 'ua'
const searchPath = '/search'
const descriptionSeparator = ', '

export class NominatimClient implements GeocodingGateway {
  private readonly baseUrl: string
  private readonly fetch: typeof fetch
  private readonly now: () => number
  private readonly wait: (milliseconds: number) => Promise<void>
  private nextAllowedAt = 0

  constructor(options: NominatimClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '')
    this.fetch = options.fetch ?? globalThis.fetch.bind(globalThis)
    this.now = options.now ?? Date.now
    this.wait = options.wait ?? ((milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)))
  }

  async search(query: string, signal?: AbortSignal): Promise<Place[]> {
    await this.takeTurn()
    signal?.throwIfAborted()

    const url = new URL(this.baseUrl + searchPath)
    url.search = new URLSearchParams({
      q: query,
      format: 'jsonv2',
      limit: resultLimit,
      'accept-language': language,
      countrycodes: countries,
    }).toString()

    let response: Response
    try {
      response = await this.fetch(url.toString(), { headers: { Accept: 'application/json' }, signal })
    } catch (error) {
      if (signal?.aborted) {
        throw error
      }

      throw networkError()
    }

    if (!response.ok) {
      throw new DomainError({
        kind: 'unavailable',
        code: ErrorCode.ServiceUnavailable,
        message: `geocoding returned HTTP ${response.status}`,
      })
    }

    return ((await response.json()) as NominatimPlace[]).map(toPlace)
  }

  private async takeTurn(): Promise<void> {
    const now = this.now()
    const startAt = Math.max(now, this.nextAllowedAt)
    this.nextAllowedAt = startAt + minimumRequestInterval
    if (startAt > now) {
      await this.wait(startAt - now)
    }
  }
}

function toPlace(place: NominatimPlace): Place {
  const parts = place.display_name.split(descriptionSeparator)
  const name = place.name || parts[0] || place.display_name

  return {
    name,
    description: parts.filter((part) => part !== name).join(descriptionSeparator),
    coordinate: { latitude: Number(place.lat), longitude: Number(place.lon) },
    bounds: toBounds(place.boundingbox),
  }
}

function toBounds(box: NominatimPlace['boundingbox']): PlaceBounds | null {
  if (box === undefined) {
    return null
  }

  const [south, north, west, east] = box.map(Number) as [number, number, number, number]

  return [south, north, west, east].every(Number.isFinite) ? { south, west, north, east } : null
}
