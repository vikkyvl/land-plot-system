import { describe, expect, it } from 'vitest'

import type { Coordinate } from './coordinate'
import { addPoint, insertPointAfter, movePoint, removeLastPoint, swapPoints } from './draft'

const a: Coordinate = { latitude: 1, longitude: 1 }
const b: Coordinate = { latitude: 2, longitude: 2 }
const c: Coordinate = { latitude: 3, longitude: 3 }
const d: Coordinate = { latitude: 4, longitude: 4 }

describe('draft points', () => {
  it('adds, moves, inserts and removes points without changing the input', () => {
    const points = [a, b]

    expect(addPoint(points, c)).toEqual([a, b, c])
    expect(movePoint(points, 1, d)).toEqual([a, d])
    expect(insertPointAfter([a, b, c], 0, d)).toEqual([a, d, b, c])
    expect(insertPointAfter([a, b, c], 2, d)).toEqual([a, b, c, d])
    expect(removeLastPoint(points)).toEqual([a])
    expect(points).toEqual([a, b])
  })

  it('swaps two points by their indices', () => {
    expect(swapPoints([a, b, c, d], 2, 3)).toEqual([a, b, d, c])
    expect(swapPoints([a, b], 0, 5)).toEqual([a, b])
  })
})
