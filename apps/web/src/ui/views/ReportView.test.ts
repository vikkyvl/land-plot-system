import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { MapLayers } from '../../domain/mapLayer'
import type { Plot } from '../../domain/plot'
import type { Report } from '../../domain/report'
import type { ReportDetails } from '../../domain/reportDetails'
import type { MapPreferencesService } from '../../service/mapPreferencesService'
import type { ReportService } from '../../service/reportService'
import type { SessionService } from '../../service/sessionService'
import {
  mapLayersKey,
  mapPreferencesServiceKey,
  reportServiceKey,
  sessionServiceKey,
} from '../composables/services'

import ReportView from './ReportView.vue'

const report: Report = {
  id: '0014abcd-0000-4000-8000-000000000001',
  plotId: 'plot-1',
  plotName: 'Поле біля ставка',
  plotVersion: 1,
  measurement: { area: { squareMeters: 24100, ares: 241, hectares: 2.41 }, perimeterMeters: 598 },
  createdAt: new Date(2026, 9, 26),
  plotDeleted: false,
}

const points = [
  { latitude: 50.45021, longitude: 30.52011 },
  { latitude: 50.45048, longitude: 30.52342 },
  { latitude: 50.44962, longitude: 30.5245 },
]

const layers = {
  map: { id: 'map', tileUrl: 'http://tiles.test/{z}/{x}/{y}.png', attribution: 'map', maxZoom: 19 },
  satellite: { id: 'satellite', tileUrl: 'http://tiles.test/s/{z}/{x}/{y}', attribution: 'sat', maxZoom: 19 },
} satisfies MapLayers

afterEach(() => {
  vi.restoreAllMocks()
})

async function mountReport(details: ReportDetails) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/reports', component: { template: '<p>reports</p>' } },
      { path: '/reports/:id', component: ReportView, props: true },
      { path: '/plots/:id', component: { template: '<p>plot</p>' } },
      { path: '/login', name: 'login', component: { template: '<p>login</p>' } },
    ],
  })
  await router.push(`/reports/${report.id}`)
  const service = {
    details: vi.fn(async () => details),
    create: vi.fn(async () => ({ ...report, id: 'report-new' })),
  }
  const wrapper = mount(
    { template: '<RouterView />' },
    {
      global: {
        plugins: [createPinia(), router],
        provide: {
          [reportServiceKey as symbol]: service as unknown as ReportService,
          [sessionServiceKey as symbol]: {} as SessionService,
          [mapLayersKey as symbol]: layers,
          [mapPreferencesServiceKey as symbol]: { layer: () => 'map' } as unknown as MapPreferencesService,
        },
        stubs: { PlotViewMap: true },
      },
    },
  )
  await flushPromises()

  return { wrapper, router, service }
}

function button(wrapper: Awaited<ReturnType<typeof mountReport>>['wrapper'], text: string) {
  return wrapper.findAll('button').find((item) => item.text().includes(text))
}

describe('ReportView', () => {
  it('shows the report from its own data and the boundary of its version', async () => {
    const { wrapper } = await mountReport({ report, plot: { version: 1 } as Plot, points })

    expect(wrapper.get('.report__title').text()).toBe('Звіт · Поле біля ставка')
    expect(wrapper.get('.report__meta').text()).toContain('Звіт № 0014 · сформовано 26.10.2026')
    expect(wrapper.get('.report__tiles').text()).toContain('2,41')
    expect(wrapper.get('.report__tiles').text()).toContain('24 100')
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    expect(wrapper.find('.report__banner').exists()).toBe(false)
    expect(wrapper.get('.report__back').attributes('href')).toBe('/plots/plot-1')
  })

  it('shows the banner for a changed plot and refreshes the report', async () => {
    const { wrapper, router, service } = await mountReport({ report, plot: { version: 2 } as Plot, points })

    const banner = wrapper.get('.report__banner')
    expect(banner.attributes('role')).toBe('status')
    expect(banner.text()).toContain('Ділянку змінено після формування звіту.')
    expect(banner.text()).toContain('Звіт побудовано для версії 1. Поточна версія 2.')

    await button(wrapper, 'Оновити звіт')?.trigger('click')
    await flushPromises()

    expect(service.create).toHaveBeenCalledWith('plot-1')
    expect(router.currentRoute.value.path).toBe('/reports/report-new')
  })

  it('keeps the report of a deleted plot without map, coordinates or GeoJSON', async () => {
    const { wrapper } = await mountReport({ report: { ...report, plotDeleted: true }, plot: null, points: null })

    expect(wrapper.get('.report__deleted').text()).toContain('Ділянку видалено')
    expect(wrapper.find('tbody').exists()).toBe(false)
    expect(wrapper.find('.report__banner').exists()).toBe(false)
    expect(button(wrapper, 'Завантажити GeoJSON')?.attributes('disabled')).toBeDefined()
    expect(wrapper.get('.report__back').attributes('href')).toBe('/reports')
  })

  it('downloads GeoJSON built on the client and copies the link', async () => {
    const created: Blob[] = []
    vi.spyOn(URL, 'createObjectURL').mockImplementation((blob) => {
      created.push(blob as Blob)

      return 'blob:report'
    })
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const writeText = vi.fn(async () => undefined)
    Object.assign(navigator, { clipboard: { writeText } })
    const { wrapper } = await mountReport({ report, plot: { version: 1 } as Plot, points })

    await button(wrapper, 'Завантажити GeoJSON')?.trigger('click')
    const feature = JSON.parse(await (created[0] as Blob).text())
    expect(click).toHaveBeenCalled()
    expect(feature.geometry.coordinates[0]).toHaveLength(4)
    expect(feature.properties.report_id).toBe(report.id)

    await button(wrapper, 'Поділитися')?.trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith(window.location.href)
  })
})
