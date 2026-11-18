import type { Accuracy } from './accuracy'
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
}

export interface ReportQuery {
  readonly plotId?: string
  readonly limit?: number
}
