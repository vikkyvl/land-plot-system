import { describe, expect, it } from 'vitest'

import { safeReturnPath } from './returnPath'

describe('safeReturnPath', () => {
  it.each([
    ['/plots/1', '/plots/1'],
    ['/reports?plot_id=1', '/reports?plot_id=1'],
    [null, '/'],
    ['', '/'],
    ['plots', '/'],
    ['//evil.example/path', '/'],
    ['/\\evil.example', '/'],
    ['https://evil.example/', '/'],
    ['/login?redirect=/plots', '/'],
    ['/auth/callback?code=1', '/'],
  ])('turns %j into %j', (path, expected) => {
    expect(safeReturnPath(path)).toBe(expected)
  })
})
