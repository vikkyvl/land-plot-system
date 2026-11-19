import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { SessionService } from '../../service/sessionService'
import { sessionServiceKey } from '../composables/services'

import AuthCallbackView from './AuthCallbackView.vue'

async function mountCallback(completeSignIn: () => Promise<string>) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/auth/callback', component: AuthCallbackView },
      { path: '/plots/:id', component: { template: '<p>plot</p>' } },
      { path: '/login', component: { template: '<p>login</p>' } },
    ],
  })
  await router.push('/auth/callback?code=1&state=2')
  const service = {
    completeSignIn: vi.fn(completeSignIn),
    isSignedIn: vi.fn(async () => true),
    currentProfile: vi.fn(async () => null),
  } as unknown as SessionService

  const wrapper = mount(AuthCallbackView, {
    global: { plugins: [createPinia(), router], provide: { [sessionServiceKey as symbol]: service } },
  })
  await flushPromises()

  return { wrapper, router }
}

describe('AuthCallbackView', () => {
  it('finishes the sign-in and returns to the saved page', async () => {
    const { router } = await mountCallback(async () => '/plots/1')

    expect(router.currentRoute.value.fullPath).toBe('/plots/1')
  })

  it('shows an alert with a way back to the login when the callback fails', async () => {
    const { wrapper, router } = await mountCallback(async () => {
      throw new Error('state mismatch')
    })

    expect(router.currentRoute.value.path).toBe('/auth/callback')
    expect(wrapper.get('[role="alert"]').text()).toContain('Не вдалося завершити вхід')
  })
})
