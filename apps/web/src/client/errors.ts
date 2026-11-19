import type { Coordinate } from '../domain/coordinate'
import { DomainError, type ErrorKind } from '../domain/error'
import { ErrorCode } from '../domain/errorCode'
import type { Problem, ProblemDetails } from '../domain/problem'

export interface ProblemDto {
  readonly code: string
  readonly message: string
  readonly details?: unknown
}

const kindByStatus: Readonly<Record<number, ErrorKind>> = {
  400: 'invalid_request',
  401: 'unauthorized',
  403: 'forbidden',
  404: 'not_found',
  409: 'conflict',
  410: 'gone',
  422: 'validation',
  429: 'rate_limited',
  502: 'unavailable',
  503: 'unavailable',
  504: 'unavailable',
}

const codeByKind: Readonly<Record<ErrorKind, string>> = {
  invalid_request: ErrorCode.InvalidRequest,
  validation: ErrorCode.InvalidRequest,
  unauthorized: ErrorCode.Unauthorized,
  forbidden: ErrorCode.Forbidden,
  not_found: ErrorCode.RouteNotFound,
  conflict: ErrorCode.VersionConflict,
  gone: ErrorCode.PlotDeleted,
  rate_limited: ErrorCode.RateLimited,
  unavailable: ErrorCode.ServiceUnavailable,
  network: ErrorCode.NetworkError,
  internal: ErrorCode.InternalError,
}

const networkErrorMessage = 'network request failed'

export function kindOfStatus(status: number): ErrorKind {
  return kindByStatus[status] ?? 'internal'
}

export interface ErrorHeaders {
  readonly requestId: string | null
  readonly retryAfter: string | null
}

const noHeaders: ErrorHeaders = { requestId: null, retryAfter: null }
const millisecondsPerSecond = 1000

export function toDomainError(
  status: number,
  body: unknown,
  headers: ErrorHeaders = noHeaders,
  now: Date = new Date(),
): DomainError {
  const kind = kindOfStatus(status)
  const fields = isRecord(body) ? body : {}
  const code = stringOr(fields['code'], codeByKind[kind])

  return new DomainError({
    kind,
    code,
    message: stringOr(fields['message'], `HTTP ${status}`),
    details: toProblemDetails(code, fields['details']),
    requestId: stringOr(fields['request_id'], headers.requestId ?? '') || null,
    retryAfterSeconds: parseRetryAfter(headers.retryAfter, now),
  })
}

export function parseRetryAfter(value: string | null, now: Date): number | null {
  if (value === null || value.trim() === '') {
    return null
  }

  if (/^\d+$/.test(value.trim())) {
    return Number(value.trim())
  }

  const date = Date.parse(value)
  if (Number.isNaN(date)) {
    return null
  }

  return Math.max(0, Math.ceil((date - now.getTime()) / millisecondsPerSecond))
}

export function networkError(): DomainError {
  return new DomainError({ kind: 'network', code: ErrorCode.NetworkError, message: networkErrorMessage })
}

export function toProblem(dto: ProblemDto): Problem {
  return { code: dto.code, message: dto.message, details: toProblemDetails(dto.code, dto.details) }
}

export function toProblemDetails(code: string, details: unknown): ProblemDetails | null {
  if (!isRecord(details)) {
    return null
  }

  switch (code) {
    case ErrorCode.SelfIntersection: {
      const { segments, swap, point } = details
      if (isPairOf(segments, isIndexPair) && isIndexPair(swap)) {
        return { kind: ErrorCode.SelfIntersection, segments, swap, point: toPoint(point) }
      }

      return null
    }
    case ErrorCode.DuplicatePoint:
      return isIndexPair(details['indices'])
        ? { kind: ErrorCode.DuplicatePoint, indices: details['indices'] }
        : null
    case ErrorCode.InvalidCoordinate:
      return isIndex(details['index']) ? { kind: ErrorCode.InvalidCoordinate, index: details['index'] } : null
    default:
      return null
  }
}

function toPoint(value: unknown): Coordinate | null {
  if (!isRecord(value) || !Number.isFinite(value['lat']) || !Number.isFinite(value['lon'])) {
    return null
  }

  return { latitude: value['lat'] as number, longitude: value['lon'] as number }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function stringOr(value: unknown, fallback: string): string {
  return typeof value === 'string' && value !== '' ? value : fallback
}

function isIndex(value: unknown): value is number {
  return Number.isInteger(value) && (value as number) >= 0
}

function isPairOf<T>(value: unknown, isItem: (item: unknown) => item is T): value is [T, T] {
  return Array.isArray(value) && value.length === 2 && value.every(isItem)
}

function isIndexPair(value: unknown): value is [number, number] {
  return isPairOf(value, isIndex)
}
