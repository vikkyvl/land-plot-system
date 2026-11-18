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

export function reflectsChanges(plot: Plot, changes: PlotChanges): boolean {
  const nameMatches = changes.name === undefined || plot.name === changes.name.trim()
  const pointsMatch =
    changes.points === undefined ||
    (plot.points.length === changes.points.length &&
      plot.points.every(
        (point, index) =>
          point.latitude === changes.points?.[index]?.latitude && point.longitude === changes.points[index]?.longitude,
      ))

  return nameMatches && pointsMatch
}
