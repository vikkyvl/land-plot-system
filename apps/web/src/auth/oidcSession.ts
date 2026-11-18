import { UserManager, WebStorageStateStore, type User } from 'oidc-client-ts'

import type { SignInMethod } from '../domain/session'
import type { AuthSession } from '../port/authSession'

export const callbackPath = '/auth/callback'
export const oidcScope = 'openid profile email plots:read plots:write reports:read reports:write'

export type OidcUserManager = Pick<
  UserManager,
  'getUser' | 'signinSilent' | 'signinRedirect' | 'signinRedirectCallback' | 'signoutRedirect' | 'removeUser'
>

export interface OidcSettings {
  readonly authority: string
  readonly clientId: string
  readonly origin: string
  readonly storage: Storage
}

interface SignInState {
  readonly returnTo: string
}

const googleIdentityProvider = 'google'

export function createUserManager(settings: OidcSettings): UserManager {
  return new UserManager({
    authority: settings.authority,
    client_id: settings.clientId,
    redirect_uri: settings.origin + callbackPath,
    post_logout_redirect_uri: `${settings.origin}/`,
    response_type: 'code',
    scope: oidcScope,
    userStore: new WebStorageStateStore({ store: settings.storage }),
    stateStore: new WebStorageStateStore({ store: settings.storage }),
    automaticSilentRenew: true,
    monitorSession: false,
    loadUserInfo: false,
  })
}

export class OidcSession implements AuthSession {
  private readonly manager: OidcUserManager

  constructor(manager: OidcUserManager) {
    this.manager = manager
  }

  async accessToken(): Promise<string | null> {
    const user = await this.manager.getUser()
    if (user === null) {
      return null
    }

    return user.expired ? this.renewAccessToken() : user.access_token
  }

  async renewAccessToken(): Promise<string | null> {
    const user = await this.manager.getUser()
    if (!user?.refresh_token) {
      return null
    }

    try {
      return (await this.manager.signinSilent())?.access_token ?? null
    } catch {
      await this.manager.removeUser()

      return null
    }
  }

  async isSignedIn(): Promise<boolean> {
    const user = await this.manager.getUser()

    return user !== null && (user.expired !== true || Boolean(user.refresh_token))
  }

  signIn(method: SignInMethod, returnTo: string): Promise<void> {
    const state: SignInState = { returnTo }

    return this.manager.signinRedirect({
      state,
      prompt: method === 'registration' ? 'create' : undefined,
      extraQueryParams: method === 'google' ? { kc_idp_hint: googleIdentityProvider } : undefined,
    })
  }

  async completeSignIn(): Promise<string> {
    const user: User = await this.manager.signinRedirectCallback()

    return returnToOf(user.state)
  }

  signOut(): Promise<void> {
    return this.manager.signoutRedirect()
  }
}

function returnToOf(state: unknown): string {
  if (typeof state === 'object' && state !== null && 'returnTo' in state && typeof state.returnTo === 'string') {
    return state.returnTo
  }

  return '/'
}
