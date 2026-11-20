import { describe, expect, it } from 'vitest'

import type { Plot } from './plot'
import type { Report } from './report'
import { hasNewerReport } from './report'
import { isOutdated, isPlotDeleted, reportNumber, type ReportDetails } from './reportDetails'

const report: Report = {
  id: '0014abcd-0000-4000-8000-000000000001',
  plotId: 'plot-1',
  plotName: 'Поле біля ставка',
  plotVersion: 1,
  measurement: { area: { squareMeters: 24100, ares: 241, hectares: 2.41 }, perimeterMeters: 598 },
  createdAt: new Date('2026-10-26T10:00:00Z'),
  plotDeleted: false,
  accuracy: null,
  terrain: null,
  climate: null,
  boundary: [],
  generatedFrom: { climateYears: null, terrainAt: null },
  latest: null,
}

const points = [
  { latitude: 50.45021, longitude: 30.52011 },
  { latitude: 50.45048, longitude: 30.52342 },
  { latitude: 50.44962, longitude: 30.5245 },
]

function details(plotVersion: number | null, deleted = false): ReportDetails {
  return {
    report: { ...report, plotDeleted: deleted },
    plot: plotVersion === null ? null : ({ version: plotVersion } as Plot),
    points: plotVersion === null ? null : points,
    obstructions: [],
  }
}

describe('report details', () => {
  it('numbers a report by the start of its id', () => {
    expect(reportNumber(report)).toBe('0014')
  })

  it('is outdated only when the plot has a newer version', () => {
    expect(isOutdated(details(2))).toBe(true)
    expect(isOutdated(details(1))).toBe(false)
    expect(isOutdated(details(null, true))).toBe(false)
  })

  it('knows that the plot is gone', () => {
    expect(isPlotDeleted(details(null, true))).toBe(true)
    expect(isPlotDeleted(details(2))).toBe(false)
  })

  it('knows when a newer report of the plot exists', () => {
    expect(hasNewerReport(report)).toBe(false)
    expect(hasNewerReport({ ...report, latest: { id: report.id, createdAt: report.createdAt } })).toBe(false)
    expect(hasNewerReport({ ...report, latest: { id: 'newer', createdAt: new Date() } })).toBe(true)
  })
})
