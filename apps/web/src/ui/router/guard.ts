import type { RouteLocationNormalized, RouteLocationRaw } from 'vue-router'

import { homePath, safeReturnPath } from '../../service/returnPath'

export const loginRouteName = 'login'
export const homeRouteName = 'home'

export interface SignedInCheck {
  isSignedIn(): Promise<boolean>
}

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guestOnly?: boolean
    ownMobileHeader?: boolean
    bottomNav?: boolean
  }
}

export async function guardRoute(
  to: RouteLocationNormalized,
  session: SignedInCheck,
): Promise<true | RouteLocationRaw> {
  if (!to.meta.requiresAuth && !to.meta.guestOnly) {
    return true
  }

  const signedIn = await session.isSignedIn()
  if (to.meta.requiresAuth && !signedIn) {
    return { name: loginRouteName, query: { redirect: to.fullPath } }
  }

  if (to.meta.guestOnly && signedIn) {
    const redirect = to.query['redirect']

    return safeReturnPath(typeof redirect === 'string' ? redirect : homePath)
  }

  return true
}
