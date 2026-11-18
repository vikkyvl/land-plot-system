import { describe, expect, it } from 'vitest'

import { ErrorCode } from '../domain/errorCode'

import { kindOfStatus, parseRetryAfter, toDomainError, toProblemDetails } from './errors'

describe('error map', () => {
  it.each([
    [400, 'invalid_request'],
    [401, 'unauthorized'],
    [403, 'forbidden'],
    [404, 'not_found'],
    [409, 'conflict'],
    [410, 'gone'],
    [422, 'validation'],
    [429, 'rate_limited'],
    [502, 'unavailable'],
    [503, 'unavailable'],
    [500, 'internal'],
    [418, 'internal'],
  ])('maps HTTP %i to %s', (status, kind) => {
    expect(kindOfStatus(status)).toBe(kind)
  })

  it.each([
    [422, ErrorCode.TooManyPoints, 'validation'],
    [409, ErrorCode.PlotNotConfirmed, 'conflict'],
    [410, ErrorCode.PlotDeleted, 'gone'],
    [429, ErrorCode.RateLimited, 'rate_limited'],
    [400, ErrorCode.InvalidQuery, 'invalid_request'],
    [403, ErrorCode.InsufficientScope, 'forbidden'],
  ])('keeps the server code for %i %s', (status, code, kind) => {
    const error = toDomainError(status, { code, message: 'server text', request_id: 'req-1' })

    expect(error).toMatchObject({ kind, code, message: 'server text', requestId: 'req-1', details: null })
  })

  it('falls back to a code by status when the body is not an error body', () => {
    expect(toDomainError(503, null)).toMatchObject({
      kind: 'unavailable',
      code: ErrorCode.ServiceUnavailable,
      message: 'HTTP 503',
      requestId: null,
      retryAfterSeconds: null,
    })
  })

  it('reads self-intersection details from a 422 response', () => {
    const body = {
      code: ErrorCode.SelfIntersection,
      message: 'polygon sides intersect',
      details: { segments: [[1, 2], [3, 0]], swap: [2, 3] },
    }

    expect(toDomainError(422, body).details).toEqual({
      kind: 'self_intersection',
      segments: [[1, 2], [3, 0]],
      swap: [2, 3],
    })
  })

  it('reads duplicate point and invalid coordinate details', () => {
    expect(toProblemDetails(ErrorCode.DuplicatePoint, { indices: [0, 2] })).toEqual({
      kind: 'duplicate_point',
      indices: [0, 2],
    })
    expect(toProblemDetails(ErrorCode.InvalidCoordinate, { index: 1 })).toEqual({
      kind: 'invalid_coordinate',
      index: 1,
    })
  })

  it.each([
    [ErrorCode.SelfIntersection, { segments: [[1, 2]], swap: [2, 3] }],
    [ErrorCode.SelfIntersection, { segments: [[1, 2], [3, 0]], swap: [2] }],
    [ErrorCode.DuplicatePoint, { indices: [0, -1] }],
    [ErrorCode.InvalidCoordinate, { index: 1.5 }],
    [ErrorCode.ZeroArea, { index: 1 }],
    [ErrorCode.SelfIntersection, null],
  ])('ignores malformed or unexpected details for %s', (code, details) => {
    expect(toProblemDetails(code, details)).toBeNull()
  })

  it('takes the retry delay of a rate-limited request from Retry-After', () => {
    const error = toDomainError(429, { code: ErrorCode.RateLimited }, { requestId: 'req-5', retryAfter: '3' })

    expect(error).toMatchObject({ kind: 'rate_limited', retryAfterSeconds: 3, requestId: 'req-5' })
  })

  it.each([
    ['7', 7],
    [' 0 ', 0],
    ['Wed, 18 Nov 2026 10:00:05 GMT', 5],
    ['Wed, 18 Nov 2026 09:59:00 GMT', 0],
    ['soon', null],
    ['', null],
    [null, null],
  ])('reads Retry-After %j as %j seconds', (value, seconds) => {
    expect(parseRetryAfter(value, new Date('2026-11-18T10:00:00Z'))).toBe(seconds)
  })
})
