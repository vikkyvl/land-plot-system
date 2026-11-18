import { describe, expect, it } from 'vitest'

import { DomainError, isDomainError } from './error'
import { ErrorCode } from './errorCode'

describe('DomainError', () => {
  it('keeps kind, code, details and request id', () => {
    const error = new DomainError({
      kind: 'validation',
      code: ErrorCode.DuplicatePoint,
      message: 'duplicate point',
      details: { kind: 'duplicate_point', indices: [0, 2] },
      requestId: 'req-1',
    })

    expect(isDomainError(error)).toBe(true)
    expect(error.is(ErrorCode.DuplicatePoint)).toBe(true)
    expect(error.is(ErrorCode.SelfIntersection)).toBe(false)
    expect(error).toMatchObject({ message: 'duplicate point', details: { indices: [0, 2] }, requestId: 'req-1' })
  })

  it('defaults details and request id to null', () => {
    const error = new DomainError({ kind: 'internal', code: ErrorCode.InternalError, message: 'internal' })

    expect(error.details).toBeNull()
    expect(error.requestId).toBeNull()
    expect(isDomainError(new Error('plain'))).toBe(false)
  })
})
