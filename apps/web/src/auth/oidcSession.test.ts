import { User } from 'oidc-client-ts'
import { afterEach, describe, expect, it, vi, type Mock } from 'vitest'

import { callbackPath, createUserManager, OidcSession, oidcScope, type OidcUserManager } from './oidcSession'

const authority = 'http://keycloak.test/realms/land-plot'
const clientId = 'land-plot-web'
const origin = 'http://web.test'

function token(label: string): string {
  return ['fake', label, 'token'].join('-')
}

function user(options: { expired?: boolean; refresh?: boolean; state?: unknown } = {}): User {
  const now = Math.floor(Date.now() / 1000)

  return new User({
    access_token: token('access'),
    refresh_token: options.refresh === false ? undefined : token('refresh'),
    token_type: 'Bearer',
    scope: oidcScope,
    profile: { sub: 'user-1', iss: authority, aud: clientId, exp: now + 300, iat: now },
    expires_at: options.expired ? now - 10 : now + 300,
    userState: options.state,
  })
}

type FakeManager = { [Method in keyof OidcUserManager]: Mock<OidcUserManager[Method]> }

function fakeManager(current: User | null): FakeManager {
  return {
    getUser: vi.fn(async () => current),
    signinSilent: vi.fn(async () => {
      const renewed = user()
      Object.assign(renewed, { access_token: token('renewed') })

      return renewed
    }),
    signinRedirect: vi.fn(async () => undefined),
    signinRedirectCallback: vi.fn(async () => current ?? user()),
    signoutRedirect: vi.fn(async () => undefined),
    removeUser: vi.fn(async () => undefined),
  }
}

afterEach(() => {
  sessionStorage.clear()
  localStorage.clear()
})

describe('OidcSession', () => {
  it('has no token and is signed out without a user', async () => {
    const session = new OidcSession(fakeManager(null))

    expect(await session.accessToken()).toBeNull()
    expect(await session.isSignedIn()).toBe(false)
  })

  it('returns the access token of a valid user', async () => {
    const session = new OidcSession(fakeManager(user()))

    expect(await session.accessToken()).toBe(token('access'))
    expect(await session.isSignedIn()).toBe(true)
  })

  it('renews an expired token with the refresh token', async () => {
    const manager = fakeManager(user({ expired: true }))
    const session = new OidcSession(manager)

    expect(await session.isSignedIn()).toBe(true)
    expect(await session.accessToken()).toBe(token('renewed'))
    expect(manager.signinSilent).toHaveBeenCalledTimes(1)
  })

  it('does not try to renew without a refresh token', async () => {
    const manager = fakeManager(user({ expired: true, refresh: false }))
    const session = new OidcSession(manager)

    expect(await session.renewAccessToken()).toBeNull()
    expect(await session.isSignedIn()).toBe(false)
    expect(manager.signinSilent).not.toHaveBeenCalled()
  })

  it('forgets the user when the silent renewal fails', async () => {
    const manager = fakeManager(user())
    manager.signinSilent.mockRejectedValueOnce(new Error('invalid_grant'))
    const session = new OidcSession(manager)

    expect(await session.renewAccessToken()).toBeNull()
    expect(manager.removeUser).toHaveBeenCalledTimes(1)
  })

  it.each([
    ['password', { prompt: undefined, extraQueryParams: { ui_locales: 'en' } }],
    ['registration', { prompt: 'create', extraQueryParams: { ui_locales: 'en' } }],
    ['google', { prompt: undefined, extraQueryParams: { ui_locales: 'en', kc_idp_hint: 'google' } }],
  ] as const)('redirects to Keycloak for %s sign-in in the chosen language', async (method, expected) => {
    const manager = fakeManager(null)

    await new OidcSession(manager).signIn(method, '/plots/1', 'en')

    expect(manager.signinRedirect).toHaveBeenCalledWith({ state: { returnTo: '/plots/1' }, ...expected })
  })

  it('completes the sign-in and returns the saved path', async () => {
    const session = new OidcSession(fakeManager(user({ state: { returnTo: '/plots/1' } })))

    expect(await session.completeSignIn()).toBe('/plots/1')
  })

  it('returns home when the callback has no saved path', async () => {
    const session = new OidcSession(fakeManager(user({ state: 'unexpected' })))

    expect(await session.completeSignIn()).toBe('/')
  })

  it('signs out through Keycloak', async () => {
    const manager = fakeManager(user())

    await new OidcSession(manager).signOut()

    expect(manager.signoutRedirect).toHaveBeenCalledTimes(1)
  })
})

describe('createUserManager', () => {
  it('uses the code flow with PKCE, refresh-token renewal and the agreed scopes', () => {
    const { settings } = createUserManager({ authority, clientId, origin, storage: sessionStorage })

    expect(settings).toMatchObject({
      authority,
      client_id: clientId,
      redirect_uri: origin + callbackPath,
      post_logout_redirect_uri: `${origin}/`,
      response_type: 'code',
      scope: 'openid profile email plots:read plots:write reports:read reports:write',
      automaticSilentRenew: true,
      monitorSession: false,
      disablePKCE: false,
    })
  })

  it('keeps the user in sessionStorage and never in localStorage', async () => {
    const manager = createUserManager({ authority, clientId, origin, storage: sessionStorage })

    await manager.storeUser(user())

    expect(sessionStorage.getItem(`oidc.user:${authority}:${clientId}`)).toContain(token('access'))
    expect(localStorage.length).toBe(0)
  })
})
