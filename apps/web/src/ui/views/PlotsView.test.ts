import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { DomainError } from '../../domain/error'
import type { Plot } from '../../domain/plot'
import type { PlotService } from '../../service/plotService'
import type { SessionService } from '../../service/sessionService'
import {
  confirmationServiceKey,
  plotServiceKey,
  sessionServiceKey,
} from '../composables/services'

import PlotsView from './PlotsView.vue'

function plot(id: string, name: string, hectares: number, status: Plot['status'] = 'CONFIRMED'): Plot {
  return {
    id,
    ownerId: 'owner-1',
    name,
    points: [
      { latitude: 50.45, longitude: 30.52 },
      { latitude: 50.451, longitude: 30.523 },
      { latitude: 50.449, longitude: 30.524 },
    ],
    measurement: { area: { squareMeters: hectares * 10000, ares: hectares * 100, hectares }, perimeterMeters: 600 },
    version: 2,
    status,
    createdAt: new Date(2026, 9, 25),
    updatedAt: new Date(2026, 9, 27),
  }
}

async function mountPlots(list: () => Promise<Plot[]>, path = '/plots') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<p>calc</p>' } },
      { path: '/login', name: 'login', component: { template: '<p>login</p>' } },
      {
        path: '/plots',
        component: PlotsView,
        children: [
          { path: '', component: { template: '<p>empty</p>' } },
          { path: ':id', component: { template: '<p>detail</p>' } },
        ],
      },
    ],
  })
  await router.push(path)
  const service = { list: vi.fn(list) }
  const wrapper = mount({ template: '<RouterView />' }, {
    global: {
      plugins: [createPinia(), router],
      provide: {
          [confirmationServiceKey as symbol]: { check: vi.fn(async () => ({ state: 'gone' })) },
        [plotServiceKey as symbol]: service as unknown as PlotService,
        [sessionServiceKey as symbol]: {} as SessionService,
      },
    },
  })
  await flushPromises()

  return { wrapper, router }
}

describe('PlotsView', () => {
  it('lists plots with date, version, area, the pending badge and the total', async () => {
    const { wrapper } = await mountPlots(async () => [
      plot('a', 'Поле біля ставка', 2.48),
      plot('b', 'Дача в Ірпені', 0.12, 'PENDING'),
    ])

    const cards = wrapper.findAll('.plots__card')
    expect(cards).toHaveLength(2)
    expect(cards[0]?.text()).toContain('Поле біля ставка')
    expect(cards[0]?.text()).toContain('27.10.2026 · версія 2')
    expect(cards[0]?.text()).toContain('2,48 га')
    expect(cards[1]?.text()).toContain('Обробляється…')
    expect(wrapper.get('.plots__summary').text()).toContain('Усього 2 ділянки')
    expect(wrapper.get('.plots__summary').text()).toContain('2,60 га')
  })

  it('filters by name and marks the selected plot', async () => {
    const { wrapper } = await mountPlots(
      async () => [plot('a', 'Поле біля ставка', 2.48), plot('b', 'Дача в Ірпені', 0.12)],
      '/plots/b',
    )

    expect(wrapper.get('[aria-current="page"]').text()).toContain('Дача в Ірпені')
    await wrapper.get('#plot-search').setValue('ірп')

    expect(wrapper.findAll('.plots__card')).toHaveLength(1)
  })

  it('shows the empty state', async () => {
    const { wrapper } = await mountPlots(async () => [])

    expect(wrapper.get('.plots__empty').text()).toContain('Ще немає жодної ділянки')
  })

  it('shows an alert with the request id when the list fails', async () => {
    const { wrapper } = await mountPlots(async () => {
      throw new DomainError({ kind: 'unavailable', code: 'service_unavailable', message: '', requestId: 'req-7' })
    })

    expect(wrapper.get('[role="alert"]').text()).toContain('Сервіс тимчасово недоступний')
    expect(wrapper.get('[role="alert"]').text()).toContain('req-7')
  })

  it('goes to the login with the way back when the session has expired', async () => {
    const { router } = await mountPlots(async () => {
      throw new DomainError({ kind: 'unauthorized', code: 'unauthorized', message: '' })
    })

    expect(router.currentRoute.value.fullPath).toBe('/login?redirect=/plots')
  })
})
