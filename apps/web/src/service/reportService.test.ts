import { describe, expect, it, vi } from 'vitest'

import { DomainError } from '../domain/error'
import type { Plot, PlotVersion } from '../domain/plot'
import type { Report } from '../domain/report'
import type { PlotGateway } from '../port/plotGateway'
import type { ReportGateway } from '../port/reportGateway'

import { ReportService } from './reportService'

const report = { id: 'report-1', plotId: 'plot-1', plotVersion: 1, plotDeleted: false, boundary: [] } as unknown as Report
const oldPoints = [{ latitude: 1, longitude: 1 }]
const newPoints = [{ latitude: 2, longitude: 2 }]
const versions = [
  { version: 2, points: newPoints },
  { version: 1, points: oldPoints },
] as unknown as PlotVersion[]

function setup(options: { report?: Report; plot?: () => Promise<Plot> } = {}) {
  const reports = {
    get: vi.fn(async () => options.report ?? report),
    list: vi.fn(async () => [report]),
    create: vi.fn(async () => report),
  }
  const plots = {
    get: vi.fn(options.plot ?? (async () => ({ id: 'plot-1', version: 2 }) as Plot)),
    versions: vi.fn(async () => versions),
  }
  const service = new ReportService(reports as unknown as ReportGateway, plots as unknown as PlotGateway)

  return { reports, plots, service }
}

describe('ReportService', () => {
  it('shows the boundary of the version the report was built for', async () => {
    const { service } = setup()

    const details = await service.details('report-1')

    expect(details.plot?.version).toBe(2)
    expect(details.points).toBe(oldPoints)
  })

  it('does not ask for the plot of a report marked as deleted', async () => {
    const { service, plots } = setup({ report: { ...report, plotDeleted: true } })

    expect(await service.details('report-1')).toEqual({
      report: { ...report, plotDeleted: true },
      plot: null,
      points: null,
      obstructions: [],
    })
    expect(plots.get).not.toHaveBeenCalled()
  })

  it('treats 410 from the plot as a deleted plot and passes other errors on', async () => {
    const gone = setup({
      plot: async () => {
        throw new DomainError({ kind: 'gone', code: 'plot_deleted', message: '' })
      },
    })
    const down = setup({
      plot: async () => {
        throw new DomainError({ kind: 'unavailable', code: 'service_unavailable', message: '' })
      },
    })

    expect((await gone.service.details('report-1')).report.plotDeleted).toBe(true)
    await expect(down.service.details('report-1')).rejects.toMatchObject({ kind: 'unavailable' })
  })

  it('lists reports with the query and creates a new one', async () => {
    const { service, reports } = setup()

    await service.list({ plotId: 'plot-1' })
    await service.create('plot-1')

    expect(reports.list).toHaveBeenCalledWith({ plotId: 'plot-1' })
    expect(reports.create).toHaveBeenCalledWith('plot-1')
  })
})
