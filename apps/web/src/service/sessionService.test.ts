import { describe, expect, it, vi } from 'vitest'

import type { Profile } from '../domain/profile'
import type { AuthSession } from '../port/authSession'
import type { ProfileGateway } from '../port/profileGateway'

import { SessionService } from './sessionService'

const profile: Profile = {
  subject: 'user-1',
  email: 'taras.melnyk@example.com',
  name: null,
  scopes: [],
  createdAt: new Date(0),
  updatedAt: new Date(0),
}

function fakes(signedIn: boolean, returnTo = '/plots') {
  const auth = {
    accessToken: vi.fn(async () => null),
    renewAccessToken: vi.fn(async () => null),
    isSignedIn: vi.fn(async () => signedIn),
    signIn: vi.fn(async () => undefined),
    completeSignIn: vi.fn(async () => returnTo),
    signOut: vi.fn(async () => undefined),
  } satisfies AuthSession
  const profiles = { get: vi.fn(async () => profile) } satisfies ProfileGateway

  return { auth, profiles, service: new SessionService(auth, profiles) }
}

describe('SessionService', () => {
  it('passes only a safe return path to the sign-in', async () => {
    const { auth, service } = fakes(false)

    await service.signIn('google', '//evil.example', 'uk')
    await service.signIn('registration', '/plots/1', 'en')

    expect(auth.signIn.mock.calls).toEqual([
      ['google', '/', 'uk'],
      ['registration', '/plots/1', 'en'],
    ])
  })

  it('returns a safe path after the callback', async () => {
    expect(await fakes(true, '/plots/1').service.completeSignIn()).toBe('/plots/1')
    expect(await fakes(true, 'https://evil.example').service.completeSignIn()).toBe('/')
  })

  it('loads the profile only for a signed-in user', async () => {
    const signedOut = fakes(false)
    const signedIn = fakes(true)

    expect(await signedOut.service.currentProfile()).toBeNull()
    expect(signedOut.profiles.get).not.toHaveBeenCalled()
    expect(await signedIn.service.currentProfile()).toEqual(profile)
  })

  it('signs out', async () => {
    const { auth, service } = fakes(true)

    await service.signOut()

    expect(auth.signOut).toHaveBeenCalledTimes(1)
  })
})
