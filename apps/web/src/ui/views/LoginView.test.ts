import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { SessionService } from '../../service/sessionService'
import { sessionServiceKey } from '../composables/services'

import LoginView from './LoginView.vue'

async function mountLogin(signIn: SessionService['signIn']) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/login', name: 'login', component: LoginView }],
  })
  await router.push('/login?redirect=/plots/1')
  const service = { signIn: vi.fn(signIn) } as unknown as SessionService

  const wrapper = mount(LoginView, {
    global: { plugins: [createPinia(), router], provide: { [sessionServiceKey as symbol]: service } },
  })

  return { wrapper, signIn: service.signIn as ReturnType<typeof vi.fn> }
}

describe('LoginView', () => {
  it.each([
    ['Увійти', 'password'],
    ['Зареєструватися', 'registration'],
    ['Продовжити з Google', 'google'],
  ])('«%s» starts the %s sign-in and keeps the way back', async (label, method) => {
    const { wrapper, signIn } = await mountLogin(async () => undefined)

    await wrapper.findAll('button').find((button) => button.text() === label)?.trigger('click')

    expect(signIn).toHaveBeenCalledWith(method, '/plots/1')
    expect(wrapper.findAll('button').every((button) => button.attributes('disabled') !== undefined)).toBe(true)
  })

  it('shows an alert when Keycloak cannot be reached', async () => {
    const { wrapper } = await mountLogin(async () => {
      throw new Error('discovery failed')
    })

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Сервіс входу зараз недоступний')
    expect(wrapper.get('button').attributes('disabled')).toBeUndefined()
  })

  it('has no email or password fields', async () => {
    const { wrapper } = await mountLogin(async () => undefined)

    expect(wrapper.findAll('input')).toHaveLength(0)
  })
})
