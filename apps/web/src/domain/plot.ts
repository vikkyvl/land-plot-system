import type { Accuracy } from './accuracy'
import type { Coordinate } from './coordinate'
import type { Measurement } from './measurement'
import type { Problem } from './problem'
import type { Capture, Obstruction } from './survey'

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
  readonly capture: Capture | null
  readonly obstructions: readonly Obstruction[]
  readonly accuracy: Accuracy | null
  readonly outline: readonly OutlinePoint[]
  readonly version: number
  readonly status: PlotStatus
  readonly createdAt: Date
  readonly updatedAt: Date
}

export interface OutlinePoint {
  readonly x: number
  readonly y: number
}

export interface PlotVersion {
  readonly version: number
  readonly name: string
  readonly points: readonly Coordinate[]
  readonly measurement: Measurement
  readonly obstructions: readonly Obstruction[]
  readonly accuracy: Accuracy | null
  readonly createdAt: Date
}

export interface SurveyFacts {
  readonly capture: Capture | null
  readonly obstructions: readonly Obstruction[]
}

export interface NewPlot extends SurveyFacts {
  readonly name: string
  readonly points: readonly Coordinate[]
}

export interface PlotChanges extends Partial<SurveyFacts> {
  readonly name?: string
  readonly points?: readonly Coordinate[]
}

export interface PlotSummary {
  readonly count: number
  readonly totalHectares: number
}

export const emptyPlotSummary: PlotSummary = { count: 0, totalHectares: 0 }

export interface PlotPreview {
  readonly valid: boolean
  readonly measurement: Measurement | null
  readonly accuracy: Accuracy | null
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
