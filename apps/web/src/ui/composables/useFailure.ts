import { DomainError, isDomainError } from '../../domain/error'
import { useToastStore } from '../stores/toast'

import { errorText } from './errorText'
import { useSignInRedirect } from './useSignInRedirect'

export interface Failure {
  readonly text: string
  readonly requestId: string | null
  readonly error: DomainError
}

export function toFailure(error: unknown): Failure {
  const domainError = isDomainError(error)
    ? error
    : new DomainError({ kind: 'internal', code: 'internal_error', message: String(error) })

  return { text: errorText(domainError), requestId: domainError.requestId, error: domainError }
}

export function useFailure(): (error: unknown, options?: { toast?: boolean }) => Promise<Failure | null> {
  const signIn = useSignInRedirect()
  const toasts = useToastStore()

  return async (error, options = {}) => {
    const failure = toFailure(error)
    if (failure.error.kind === 'unauthorized') {
      await signIn()

      return null
    }
    if (options.toast) {
      toasts.show(failure.text, 'error')
    }

    return failure
  }
}
