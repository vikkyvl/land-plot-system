import { describe, expect, it } from 'vitest'

import { DomainError } from '../domain/error'
import type { Plot } from '../domain/plot'
import type { PlotGateway } from '../port/plotGateway'

import { ConfirmationService } from './confirmationService'

const points = [
  { latitude: 1, longitude: 2 },
  { latitude: 3, longitude: 4 },
  { latitude: 5, longitude: 1 },
]

function service(get: () => Promise<Plot>): ConfirmationService {
  return new ConfirmationService({ get } as unknown as PlotGateway)
}

function plot(status: Plot['status'], name = 'Поле'): Plot {
  return { id: 'plot-1', name, points, status, version: 2 } as unknown as Plot
}

describe('ConfirmationService', () => {
  it('reports a plot that is still pending', async () => {
    expect(await service(async () => plot('PENDING')).check('plot-1')).toMatchObject({ state: 'pending' })
  })

  it('confirms a plot with the submitted changes', async () => {
    const check = await service(async () => plot('CONFIRMED')).check('plot-1', { name: 'Поле', points })

    expect(check).toMatchObject({ state: 'confirmed', plot: { name: 'Поле' } })
  })

  it('recognises a change rolled back by the saga', async () => {
    const check = await service(async () => plot('CONFIRMED', 'Старе поле')).check('plot-1', { name: 'FAIL поле' })

    expect(check).toMatchObject({ state: 'reverted', plot: { name: 'Старе поле' } })
  })

  it('treats 404 as a cancelled save and 410 as a deleted plot', async () => {
    const notFound = service(async () => {
      throw new DomainError({ kind: 'not_found', code: 'plot_not_found', message: '' })
    })
    const gone = service(async () => {
      throw new DomainError({ kind: 'gone', code: 'plot_deleted', message: '' })
    })

    expect(await notFound.check('plot-1')).toEqual({ state: 'cancelled' })
    expect(await gone.check('plot-1')).toEqual({ state: 'gone' })
  })

  it('passes other errors on', async () => {
    const down = service(async () => {
      throw new DomainError({ kind: 'network', code: 'network_error', message: '' })
    })

    await expect(down.check('plot-1')).rejects.toMatchObject({ kind: 'network' })
  })
})
