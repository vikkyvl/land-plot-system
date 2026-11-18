import { describe, expect, it } from 'vitest'

import { isPending, PlotStatus, type Plot } from './plot'

const plot: Plot = {
  id: 'plot-1',
  ownerId: 'owner-1',
  name: 'Поле',
  points: [],
  measurement: { area: { squareMeters: 1, ares: 0.01, hectares: 0.0001 }, perimeterMeters: 4 },
  version: 1,
  status: PlotStatus.Pending,
  createdAt: new Date(0),
  updatedAt: new Date(0),
}

describe('plot', () => {
  it('is pending only while the saga runs', () => {
    expect(isPending(plot)).toBe(true)
    expect(isPending({ ...plot, status: PlotStatus.Confirmed })).toBe(false)
  })
})
