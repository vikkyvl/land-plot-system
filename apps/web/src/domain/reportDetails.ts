import type { Coordinate } from './coordinate'
import type { Plot } from './plot'
import type { Report } from './report'
import type { Obstruction } from './survey'

export interface ReportDetails {
  readonly report: Report
  readonly plot: Plot | null
  readonly points: readonly Coordinate[] | null
  readonly obstructions: readonly Obstruction[]
}

const reportNumberLength = 4

export function reportNumber(report: Pick<Report, 'id'>): string {
  return report.id.replace(/-/g, '').slice(0, reportNumberLength).toUpperCase()
}

export function isOutdated(details: ReportDetails): boolean {
  return details.plot !== null && details.report.plotVersion < details.plot.version
}

export function isPlotDeleted(details: ReportDetails): boolean {
  return details.report.plotDeleted || details.plot === null
}
