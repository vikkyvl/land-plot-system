import type { Accuracy } from './accuracy'
import type { Coordinate } from './coordinate'
import type { Measurement } from './measurement'
import type { Climate, Terrain } from './surroundings'

export interface Report {
  readonly id: string
  readonly plotId: string
  readonly plotName: string
  readonly plotVersion: number
  readonly measurement: Measurement
  readonly createdAt: Date
  readonly plotDeleted: boolean
  readonly accuracy: Accuracy | null
  readonly terrain: Terrain | null
  readonly climate: Climate | null
  readonly boundary: readonly Coordinate[]
  readonly generatedFrom: GeneratedFrom
  readonly latest: LatestReport | null
}

export interface GeneratedFrom {
  readonly climateYears: { readonly first: number; readonly last: number } | null
  readonly terrainAt: Date | null
}

export interface LatestReport {
  readonly id: string
  readonly createdAt: Date
}

export type ReportFormat = 'pdf' | 'geojson'

export function hasNewerReport(report: Pick<Report, 'id' | 'latest'>): boolean {
  return report.latest !== null && report.latest.id !== report.id
}

export interface ReportQuery {
  readonly plotId?: string
  readonly limit?: number
}
