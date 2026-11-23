import type { Coordinate } from './coordinate'

export interface PlaceBounds {
  readonly south: number
  readonly west: number
  readonly north: number
  readonly east: number
}

export interface Place {
  readonly name: string
  readonly description: string
  readonly coordinate: Coordinate
  readonly bounds: PlaceBounds | null
}
