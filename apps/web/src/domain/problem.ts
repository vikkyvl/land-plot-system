import type { Coordinate } from './coordinate'

export type SegmentIndices = readonly [number, number]

export interface SelfIntersectionDetails {
  readonly kind: 'self_intersection'
  readonly segments: readonly [SegmentIndices, SegmentIndices]
  readonly swap: readonly [number, number]
  readonly point: Coordinate | null
}

export interface DuplicatePointDetails {
  readonly kind: 'duplicate_point'
  readonly indices: readonly [number, number]
}

export interface InvalidCoordinateDetails {
  readonly kind: 'invalid_coordinate'
  readonly index: number
}

export type ProblemDetails = SelfIntersectionDetails | DuplicatePointDetails | InvalidCoordinateDetails

export interface Problem {
  readonly code: string
  readonly message: string
  readonly details: ProblemDetails | null
}
