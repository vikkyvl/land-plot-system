import { describe, expect, it } from 'vitest'

import {
  formatAres,
  formatCoordinate,
  formatHectares,
  formatMeters,
  formatPointCount,
  formatSquareMeters,
} from './format'

const space = ' '

describe('format', () => {
  it('formats the measurement as agreed: ha 2 digits, ares 1, m² and metres whole', () => {
    expect(formatHectares(2.48124)).toBe('2,48 га')
    expect(formatAres(248.124)).toBe('248,1')
    expect(formatSquareMeters(24812.4)).toBe(`24${space}812`)
    expect(formatMeters(611.6)).toBe('612 м')
  })

  it('shows coordinates with five digits', () => {
    expect(formatCoordinate({ latitude: 50.4502, longitude: 30.52011 })).toBe('50.45020, 30.52011')
  })

  it.each([
    [1, '1 точка'],
    [3, '3 точки'],
    [5, '5 точок'],
    [11, '11 точок'],
    [12, '12 точок'],
    [22, '22 точки'],
    [21, '21 точка'],
  ])('names %i points', (count, text) => {
    expect(formatPointCount(count)).toBe(text)
  })
})
