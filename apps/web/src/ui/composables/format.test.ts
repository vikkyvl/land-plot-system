import { describe, expect, it } from 'vitest'

import {
  formatAres,
  formatCoordinate,
  formatDate,
  formatHectares,
  formatMeters,
  formatPlotCount,
  formatPointCount,
  formatSquareMeters,
  formatVersionCount,
  initials,
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

  it('formats dates as dd.mm.yyyy and names plots and versions', () => {
    expect(formatDate(new Date(2026, 9, 7))).toBe('07.10.2026')
    expect(formatPlotCount(3)).toBe('3 ділянки')
    expect(formatPlotCount(5)).toBe('5 ділянок')
    expect(formatVersionCount(2)).toBe('2 версії')
    expect(formatVersionCount(1)).toBe('1 версія')
  })

  it('takes initials from a name or an email', () => {
    expect(initials('Вікторія Лещенко')).toBe('ВЛ')
    expect(initials('taras.melnyk@example.com')).toBe('TM')
  })
})
