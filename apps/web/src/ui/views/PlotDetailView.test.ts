import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { pollInterval } from '../stores/confirmations'
import { createMemoryHistory, createRouter } from 'vue-router'

import { DomainError } from '../../domain/error'
import type { MapLayers } from '../../domain/mapLayer'
import type { Plot, PlotVersion } from '../../domain/plot'
import type { SettingsService } from '../../service/settingsService'
import type { PlotService } from '../../service/plotService'
import type { SessionService } from '../../service/sessionService'
import {
  confirmationServiceKey,
  mapLayersKey,
  settingsServiceKey,
  plotServiceKey,
  sessionServiceKey,
} from '../composables/services'

import PlotDetailView from './PlotDetailView.vue'

const points = [
  { latitude: 50.45, longitude: 30.52 },
  { latitude: 50.451, longitude: 30.523 },
  { latitude: 50.449, longitude: 30.524 },
]

function plot(status: Plot['status'] = 'CONFIRMED'): Plot {
  return {
    id: 'plot-1',
    ownerId: 'owner-1',
    name: 'Поле біля ставка',
    points,
    measurement: { area: { squareMeters: 24810, ares: 248.1, hectares: 2.481 }, perimeterMeters: 612 },
    capture: null,
    obstructions: [],
    accuracy: null,
    version: 2,
    status,
    createdAt: new Date(2026, 9, 25),
    updatedAt: new Date(2026, 9, 27),
  }
}

const versions: PlotVersion[] = [
  {
    version: 2,
    name: 'Поле біля ставка',
    points,
    measurement: { area: { squareMeters: 24810, ares: 248.1, hectares: 2.481 }, perimeterMeters: 612 },
    obstructions: [],
    accuracy: null,
    createdAt: new Date(2026, 9, 27),
  },
  {
    version: 1,
    name: 'Поле',
    points: points.slice(0, 3),
    measurement: { area: { squareMeters: 24100, ares: 241, hectares: 2.41 }, perimeterMeters: 598 },
    obstructions: [],
    accuracy: null,
    createdAt: new Date(2026, 9, 25),
  },
]

const layers = {
  map: { id: 'map', tileUrl: 'http://tiles.test/{z}/{x}/{y}.png', attribution: 'map', maxZoom: 19 },
  satellite: { id: 'satellite', tileUrl: 'http://tiles.test/s/{z}/{x}/{y}', attribution: 'sat', maxZoom: 19 },
} satisfies MapLayers

afterEach(() => {
  document.body.innerHTML = ''
})

async function mountDetail(
  overrides: Partial<Record<keyof PlotService, unknown>> = {},
  check: () => Promise<unknown> = async () => ({ state: 'gone' }),
) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/plots', component: { template: '<p>list</p>' } },
      { path: '/plots/:id', component: PlotDetailView, props: true },
      { path: '/reports/:id', component: { template: '<p>report</p>' } },
      { path: '/login', name: 'login', component: { template: '<p>login</p>' } },
    ],
  })
  await router.push('/plots/plot-1')
  const service = {
    get: vi.fn(async () => plot()),
    versions: vi.fn(async () => versions),
    rename: vi.fn(async (_id: string, name: string) => ({ ...plot('PENDING'), name })),
    remove: vi.fn(async () => undefined),
    createReport: vi.fn(async () => ({ id: 'report-1' })),
    list: vi.fn(async () => []),
    ...overrides,
  }
  const wrapper = mount(PlotDetailView, {
    attachTo: document.body,
    props: { id: 'plot-1' },
    global: {
      plugins: [createPinia(), router],
      provide: {
          [confirmationServiceKey as symbol]: { check: vi.fn(check) },
        [plotServiceKey as symbol]: service as unknown as PlotService,
        [sessionServiceKey as symbol]: {} as SessionService,
        [mapLayersKey as symbol]: layers,
        [settingsServiceKey as symbol]: { layer: () => 'map' } as unknown as SettingsService,
      },
      stubs: { PlotViewMap: true, Teleport: true },
    },
  })
  await flushPromises()

  return { wrapper, router, service }
}

function button(wrapper: Awaited<ReturnType<typeof mountDetail>>['wrapper'], label: string) {
  return wrapper.findAll('button').find((item) => item.text() === label)
}

describe('PlotDetailView', () => {
  it('shows the area, the actions and the version history', async () => {
    const { wrapper } = await mountDetail()

    expect(wrapper.get('.detail__title').text()).toBe('Поле біля ставка')
    expect(wrapper.get('.detail__meta').text()).toContain('Створено 25.10.2026 · 3 точки')
    expect(wrapper.get('.detail__tiles').text()).toContain('2,48 га')
    expect(wrapper.get('.detail__tiles').text()).toContain('24 810')
    const rows = wrapper.findAll('tbody tr').map((row) => row.text())
    expect(rows[0]).toContain('27.10.2026')
    expect(rows[1]).toContain('2,41 га')
    expect(wrapper.get('.detail__version--current').text()).toContain('Версія 2 · поточна')
    expect(wrapper.get('.detail__edit').attributes('href')).toBe('/plots/plot-1/edit')
  })

  it('disables changes while the plot is pending', async () => {
    const { wrapper } = await mountDetail({ get: vi.fn(async () => plot('PENDING')) })

    expect(wrapper.text()).toContain('Обробляється…')
    for (const label of ['Змінити межу', 'Перейменувати', 'Видалити', 'Сформувати звіт']) {
      expect(button(wrapper, label)?.attributes('disabled'), label).toBeDefined()
    }
  })

  it('renames through the dialog', async () => {
    const { wrapper, service } = await mountDetail()

    await button(wrapper, 'Перейменувати')?.trigger('click')
    await wrapper.get('[role="dialog"] input').setValue('Нова назва')
    await wrapper.get('[role="dialog"] form').trigger('submit')
    await flushPromises()

    expect(service.rename).toHaveBeenCalledWith('plot-1', 'Нова назва')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(wrapper.get('.detail__title').text()).toBe('Нова назва')
  })

  it('deletes after confirmation and goes back to the list', async () => {
    const { wrapper, router, service } = await mountDetail()

    await button(wrapper, 'Видалити')?.trigger('click')
    await wrapper.get('[role="dialog"] input').setValue('Поле біля ставка')
    await wrapper.get('[role="dialog"] form').trigger('submit')
    await flushPromises()

    expect(service.remove).toHaveBeenCalledWith('plot-1')
    expect(router.currentRoute.value.path).toBe('/plots')
  })

  it('keeps the dialog open with the server error when deletion is refused', async () => {
    const { wrapper } = await mountDetail({
      remove: vi.fn(async () => {
        throw new DomainError({ kind: 'conflict', code: 'plot_not_confirmed', message: '', requestId: 'req-9' })
      }),
    })

    await button(wrapper, 'Видалити')?.trigger('click')
    await wrapper.get('[role="dialog"] input').setValue('Поле біля ставка')
    await wrapper.get('[role="dialog"] form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="dialog"] [role="alert"]').text()).toContain('Ділянка ще обробляється')
    expect(wrapper.get('[role="dialog"] [role="alert"]').text()).toContain('req-9')
  })

  it('tells that a deleted plot is gone', async () => {
    const { wrapper } = await mountDetail({
      get: vi.fn(async () => {
        throw new DomainError({ kind: 'gone', code: 'plot_deleted', message: '' })
      }),
    })

    expect(wrapper.get('[role="alert"]').text()).toContain('Ділянку видалено')
  })

  it('creates a report and opens it', async () => {
    const { wrapper, router, service } = await mountDetail()

    await button(wrapper, 'Сформувати звіт')?.trigger('click')
    await flushPromises()

    expect(service.createReport).toHaveBeenCalledWith('plot-1')
    expect(router.currentRoute.value.path).toBe('/reports/report-1')
  })

  it('follows a pending plot until it is confirmed and refreshes the history', async () => {
    vi.useFakeTimers()
    try {
      const { wrapper, service } = await mountDetail({ get: vi.fn(async () => plot('PENDING')) }, async () => ({
        state: 'confirmed',
        plot: plot('CONFIRMED'),
      }))
      expect(wrapper.text()).toContain('Обробляється…')

      await vi.advanceTimersByTimeAsync(pollInterval)
      await flushPromises()

      expect(wrapper.text()).not.toContain('Обробляється…')
      expect(button(wrapper, 'Змінити межу')).toBeUndefined()
      expect(wrapper.get('.detail__edit').attributes('href')).toBe('/plots/plot-1/edit')
      expect(service.versions).toHaveBeenCalledTimes(2)
    } finally {
      vi.useRealTimers()
    }
  })
})
