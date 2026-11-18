import { inject, type InjectionKey } from 'vue'

import type { SessionService } from '../../service/sessionService'

export const sessionServiceKey: InjectionKey<SessionService> = Symbol('SessionService')

export function useSessionService(): SessionService {
  const service = inject(sessionServiceKey)
  if (service === undefined) {
    throw new Error('SessionService is not provided')
  }

  return service
}
