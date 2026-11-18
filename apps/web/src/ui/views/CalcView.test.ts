import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { DomainError } from '../../domain/error'
import type { MapLayers } from '../../domain/mapLayer'
import type { PlotPreview } from '../../domain/plot'
import type { CalculatorService } from '../../service/calculatorService'
import type { MapPreferencesService } from '../../service/mapPreferencesService'
import type { PlaceSearchService } from '../../service/placeSearchService'
import type { SessionService } from '../../service/sessionService'
import {
  calculatorServiceKey,
  mapLayersKey,
  mapPreferencesServiceKey,
  placeSearchServiceKey,
  sessionServiceKey,
} from '../composables/services'

import CalcView from './CalcView.vue'

const square = [
  { latitude: 50.45021, longitude: 30.52011 },
  { latitude: 50.45048, longitude: 30.52342 },
  { latitude: 50.44962, longitude: 30.5245 },
  { latitude: 50.44871, longitude: 30.52388 },
]

const valid: PlotPreview = {
  valid: true,
  measurement: { area: { squareMeters: 24810.4, ares: 248.104, hectares: 2.48104 }, perimeterMeters: 611.7 },
  problems: [],
}

const crossed: PlotPreview = {
  valid: false,
  measurement: null,
  problems: [
    {
      code: 'self_intersection',
      message: 'polygon sides intersect',
      details: { kind: 'self_intersection', segments: [[1, 2], [3, 0]], swap: [2, 3] },
    },
  ],
}

const layers: MapLayers = {
  map: { id: 'map', tileUrl: 'http://tiles.test/{z}/{x}/{y}.png', attribution: 'map', maxZoom: 19 },
  satellite: { id: 'satellite', tileUrl: 'http://tiles.test/s/{z}/{x}/{y}', attribution: 'sat', maxZoom: 19 },
}

async function mountCalc(preview: () => Promise<PlotPreview>) {
  const calculator = {
    canPreview: (points: readonly unknown[]) => points.length >= 3,
    preview: vi.fn(preview),
    loadDraft: vi.fn(async () => ({ name: '', points: square, saveAfterSignIn: false })),
    keepDraft: vi.fn(async () => undefined),
    save: vi.fn(),
    saveAfterSignIn: vi.fn(async () => undefined),
    savePendingDraft: vi.fn(async () => null),
  }
  const session = { isSignedIn: vi.fn(async () => false), currentProfile: vi.fn(async () => null) }
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: CalcView },
      { path: '/login', name: 'login', component: { template: '<p>login</p>' } },
    ],
  })
  await router.push('/')

  const wrapper = mount(CalcView, {
    global: {
      plugins: [createPinia(), router],
      provide: {
        [calculatorServiceKey as symbol]: calculator as unknown as CalculatorService,
        [sessionServiceKey as symbol]: session as unknown as SessionService,
        [placeSearchServiceKey as symbol]: { search: vi.fn(async () => []) } as unknown as PlaceSearchService,
        [mapPreferencesServiceKey as symbol]: {
          layer: () => 'map',
          chooseLayer: vi.fn(),
        } as unknown as MapPreferencesService,
        [mapLayersKey as symbol]: layers,
      },
      stubs: { PlotMap: { template: '<div class="plot-map-stub"><slot /></div>' } },
    },
  })
  await flushPromises()
  await vi.advanceTimersByTimeAsync(0)
  await flushPromises()

  return { wrapper, router, calculator }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('CalcView', () => {
  it('shows the server area in the agreed format', async () => {
    const { wrapper } = await mountCalc(async () => valid)

    expect(wrapper.get('.calc__value').text()).toBe('2,48 га')
    expect(wrapper.get('.calc__tiles').text()).toContain('24 810')
    expect(wrapper.get('.calc__tiles').text()).toContain('248,1')
    expect(wrapper.get('.calc__tiles').text()).toContain('612 м')
    expect(wrapper.get('.calc__status').text()).toBe('Розрахунок сервера')
    expect(wrapper.get('.calc__save').attributes('disabled')).toBeUndefined()
  })

  it('shows the error panel with the swap button and blocks saving on self-intersection', async () => {
    const { wrapper, calculator } = await mountCalc(async () => crossed)

    const panel = wrapper.get('.error-panel')
    expect(panel.attributes('role')).toBe('alert')
    expect(panel.text()).toContain('Сторони 2–3 і 4–1 перетинаються')
    expect(panel.text()).toContain('Код помилки: self_intersection')
    expect(wrapper.get('.calc__value').text()).toBe('— га')
    expect(wrapper.get('.calc__save').attributes('disabled')).toBeDefined()
    expect(wrapper.get('.checks-card').text()).toContain('Без самоперетину')

    await panel.findAll('button').find((button) => button.text() === 'Поміняти точки 3 і 4')?.trigger('click')
    await vi.advanceTimersByTimeAsync(300)
    expect(calculator.preview).toHaveBeenLastCalledWith([square[0], square[1], square[3], square[2]])
  })

  it('hides the panel on «Виправлю сам»', async () => {
    const { wrapper } = await mountCalc(async () => crossed)

    await wrapper.findAll('.error-panel button').find((button) => button.text() === 'Виправлю сам')?.trigger('click')

    expect(wrapper.find('.error-panel').exists()).toBe(false)
  })

  it('shows the outage panel with the seconds from Retry-After on 429', async () => {
    const { wrapper } = await mountCalc(async () => {
      throw new DomainError({ kind: 'rate_limited', code: 'rate_limited', message: '', retryAfterSeconds: 7 })
    })

    expect(wrapper.get('.outage-panel').text()).toContain('Розрахунок тимчасово недоступний')
    expect(wrapper.get('.outage-panel').text()).toContain('Спробуйте через 7 с.')
    expect(wrapper.get('.calc__save').attributes('disabled')).toBeDefined()
  })

  it('sends a guest to the login with the way back and keeps the draft', async () => {
    const { wrapper, router, calculator } = await mountCalc(async () => valid)

    await wrapper.get('.calc__save').trigger('click')
    await flushPromises()

    expect(calculator.saveAfterSignIn).toHaveBeenCalled()
    expect(router.currentRoute.value.fullPath).toBe('/login?redirect=/')
  })
})
