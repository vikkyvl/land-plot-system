import type { ProblemDetails } from './problem'

export type ErrorKind =
  | 'invalid_request'
  | 'validation'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'conflict'
  | 'gone'
  | 'rate_limited'
  | 'unavailable'
  | 'network'
  | 'internal'

export interface DomainErrorOptions {
  readonly kind: ErrorKind
  readonly code: string
  readonly message: string
  readonly details?: ProblemDetails | null
  readonly requestId?: string | null
}

export class DomainError extends Error {
  readonly kind: ErrorKind
  readonly code: string
  readonly details: ProblemDetails | null
  readonly requestId: string | null

  constructor(options: DomainErrorOptions) {
    super(options.message)
    this.name = 'DomainError'
    this.kind = options.kind
    this.code = options.code
    this.details = options.details ?? null
    this.requestId = options.requestId ?? null
  }

  is(code: string): boolean {
    return this.code === code
  }
}

export function isDomainError(error: unknown): error is DomainError {
  return error instanceof DomainError
}
