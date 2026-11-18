import { describe, expect, it } from 'vitest'

import { ErrorCode } from '../../domain/errorCode'

import { errorText, problemText } from './errorText'

describe('errorText', () => {
  it('has a Ukrainian text for every backend code', () => {
    for (const code of Object.values(ErrorCode).filter((value) => value !== ErrorCode.InternalError)) {
      expect(errorText({ code }), code).not.toBe('Щось пішло не так. Спробуйте ще раз.')
    }
    expect(errorText({ code: ErrorCode.InternalError })).toBe('Щось пішло не так. Спробуйте ще раз.')
  })

  it('falls back for an unknown code', () => {
    expect(errorText({ code: 'brand_new' })).toBe('Щось пішло не так. Спробуйте ще раз.')
  })

  it('describes the self-intersection with one-based sides as in the design', () => {
    const text = problemText({
      code: ErrorCode.SelfIntersection,
      message: 'polygon sides intersect',
      details: { kind: 'self_intersection', segments: [[1, 2], [3, 0]], swap: [2, 3] },
    })

    expect(text.title).toBe('Сторони контуру перетинаються')
    expect(text.text).toContain('Сторони 2–3 і 4–1 перетинаються.')
  })

  it('names the duplicate points and the bad coordinate', () => {
    expect(
      problemText({ code: ErrorCode.DuplicatePoint, message: '', details: { kind: 'duplicate_point', indices: [0, 2] } })
        .text,
    ).toContain('Точки 1 і 3 збігаються')
    expect(
      problemText({ code: ErrorCode.InvalidCoordinate, message: '', details: { kind: 'invalid_coordinate', index: 1 } })
        .text,
    ).toContain('Точка 2')
    expect(problemText({ code: ErrorCode.TooManyPoints, message: '', details: null }).title).toBe('Забагато точок')
  })
})
