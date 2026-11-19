import { describe, expect, it } from 'vitest'

import { isPending, PlotStatus, reflectsChanges, type Plot } from './plot'

const plot: Plot = {
  id: 'plot-1',
  ownerId: 'owner-1',
  name: 'Поле',
  points: [],
  measurement: { area: { squareMeters: 1, ares: 0.01, hectares: 0.0001 }, perimeterMeters: 4 },
  capture: null,
  outline: [],
  obstructions: [],
  accuracy: null,
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

describe('reflectsChanges', () => {
  const confirmed: Plot = {
    ...plot,
    status: PlotStatus.Confirmed,
    name: 'Поле',
    points: [
      { latitude: 1, longitude: 2 },
      { latitude: 3, longitude: 4 },
    ],
  }

  it('matches the submitted name and points', () => {
    expect(reflectsChanges(confirmed, { name: ' Поле ' })).toBe(true)
    expect(reflectsChanges(confirmed, { points: confirmed.points })).toBe(true)
    expect(reflectsChanges(confirmed, {})).toBe(true)
  })

  it('notices a restored plot', () => {
    expect(reflectsChanges(confirmed, { name: 'FAIL нова назва' })).toBe(false)
    expect(reflectsChanges(confirmed, { points: [{ latitude: 1, longitude: 2 }] })).toBe(false)
    expect(reflectsChanges(confirmed, { points: [{ latitude: 1, longitude: 2 }, { latitude: 3, longitude: 5 }] })).toBe(
      false,
    )
  })
})
