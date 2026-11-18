import type { Measurement } from './measurement'

export interface Report {
  readonly id: string
  readonly plotId: string
  readonly plotName: string
  readonly plotVersion: number
  readonly measurement: Measurement
  readonly createdAt: Date
  readonly plotDeleted: boolean
}

export interface ReportQuery {
  readonly plotId?: string
  readonly limit?: number
}
