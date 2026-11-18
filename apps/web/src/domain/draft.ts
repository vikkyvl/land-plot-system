import type { Coordinate } from './coordinate'

export interface PlotDraft {
  readonly name: string
  readonly points: readonly Coordinate[]
  readonly saveAfterSignIn: boolean
}

export const emptyDraft: PlotDraft = { name: '', points: [], saveAfterSignIn: false }

export function addPoint(points: readonly Coordinate[], point: Coordinate): Coordinate[] {
  return [...points, point]
}

export function movePoint(points: readonly Coordinate[], index: number, point: Coordinate): Coordinate[] {
  return points.map((current, position) => (position === index ? point : current))
}

export function insertPointAfter(points: readonly Coordinate[], index: number, point: Coordinate): Coordinate[] {
  return [...points.slice(0, index + 1), point, ...points.slice(index + 1)]
}

export function removeLastPoint(points: readonly Coordinate[]): Coordinate[] {
  return points.slice(0, -1)
}

export function swapPoints(points: readonly Coordinate[], first: number, second: number): Coordinate[] {
  const swapped = [...points]
  const firstPoint = points[first]
  const secondPoint = points[second]
  if (firstPoint === undefined || secondPoint === undefined) {
    return swapped
  }

  swapped[first] = secondPoint
  swapped[second] = firstPoint

  return swapped
}
