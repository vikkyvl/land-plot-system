import { describe, expect, it } from 'vitest'
import type { RouteLocationNormalized } from 'vue-router'

import { guardRoute } from './guard'

function route(fullPath: string, meta: RouteLocationNormalized['meta'], query = {}): RouteLocationNormalized {
  return { fullPath, meta, query } as RouteLocationNormalized
}

const signedIn = { isSignedIn: async () => true }
const signedOut = { isSignedIn: async () => false }

describe('guardRoute', () => {
  it('lets anyone open public routes', async () => {
    expect(await guardRoute(route('/', {}), signedOut)).toBe(true)
  })

  it('sends a guest from a protected route to the login with the way back', async () => {
    expect(await guardRoute(route('/plots/1?tab=history', { requiresAuth: true }), signedOut)).toEqual({
      name: 'login',
      query: { redirect: '/plots/1?tab=history' },
    })
    expect(await guardRoute(route('/plots', { requiresAuth: true }), signedIn)).toBe(true)
  })

  it('sends a signed-in user from the login to the safe redirect', async () => {
    expect(await guardRoute(route('/login', { guestOnly: true }, { redirect: '/plots' }), signedIn)).toBe('/plots')
    expect(await guardRoute(route('/login', { guestOnly: true }, { redirect: '//evil.example' }), signedIn)).toBe('/')
    expect(await guardRoute(route('/login', { guestOnly: true }), signedOut)).toBe(true)
  })
})
