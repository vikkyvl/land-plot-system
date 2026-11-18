import type { Coordinate } from './coordinate'
import type { Measurement } from './measurement'
import type { Problem } from './problem'

export const PlotStatus = {
  Pending: 'PENDING',
  Confirmed: 'CONFIRMED',
  Cancelled: 'CANCELLED',
} as const

export type PlotStatus = (typeof PlotStatus)[keyof typeof PlotStatus]

export interface Plot {
  readonly id: string
  readonly ownerId: string
  readonly name: string
  readonly points: readonly Coordinate[]
  readonly measurement: Measurement
  readonly version: number
  readonly status: PlotStatus
  readonly createdAt: Date
  readonly updatedAt: Date
}

export interface PlotVersion {
  readonly version: number
  readonly name: string
  readonly points: readonly Coordinate[]
  readonly measurement: Measurement
  readonly createdAt: Date
}

export interface NewPlot {
  readonly name: string
  readonly points: readonly Coordinate[]
}

export interface PlotChanges {
  readonly name?: string
  readonly points?: readonly Coordinate[]
}

export interface PlotPreview {
  readonly valid: boolean
  readonly measurement: Measurement | null
  readonly problems: readonly Problem[]
}

export function isPending(plot: Plot): boolean {
  return plot.status === PlotStatus.Pending
}
