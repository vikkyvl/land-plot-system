import { describe, expect, it, vi } from 'vitest'

import type { Plot, PlotVersion } from '../domain/plot'
import type { Report } from '../domain/report'
import type { PlotGateway } from '../port/plotGateway'
import type { ReportGateway } from '../port/reportGateway'

import { PlotService } from './plotService'

function setup() {
  const plot = { id: 'plot-1', name: 'Поле' } as Plot
  const plots = {
    list: vi.fn(async () => [plot]),
    get: vi.fn(async () => plot),
    versions: vi.fn(async () => [] as PlotVersion[]),
    update: vi.fn(async () => plot),
    remove: vi.fn(async () => undefined),
  } as unknown as PlotGateway & Record<'list' | 'get' | 'versions' | 'update' | 'remove', ReturnType<typeof vi.fn>>
  const reports = { create: vi.fn(async () => ({ id: 'report-1' }) as Report) } as unknown as ReportGateway & {
    create: ReturnType<typeof vi.fn>
  }

  return { plots, reports, service: new PlotService(plots, reports) }
}

describe('PlotService', () => {
  it('renames with a trimmed name and changes nothing else', async () => {
    const { plots, service } = setup()

    await service.rename('plot-1', '  Нова назва ')

    expect(plots.update).toHaveBeenCalledWith('plot-1', { name: 'Нова назва' })
  })

  it('passes list, details, history, deletion and report creation to the gateways', async () => {
    const { plots, reports, service } = setup()

    await service.list()
    await service.get('plot-1')
    await service.versions('plot-1')
    await service.remove('plot-1')
    await service.createReport('plot-1')

    expect(plots.list).toHaveBeenCalled()
    expect(plots.get).toHaveBeenCalledWith('plot-1')
    expect(plots.versions).toHaveBeenCalledWith('plot-1')
    expect(plots.remove).toHaveBeenCalledWith('plot-1')
    expect(reports.create).toHaveBeenCalledWith('plot-1')
  })
})
