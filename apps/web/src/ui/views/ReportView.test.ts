import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { MapLayers } from '../../domain/mapLayer'
import type { Plot } from '../../domain/plot'
import type { Report } from '../../domain/report'
import type { ReportDetails } from '../../domain/reportDetails'
import type { SettingsService } from '../../service/settingsService'
import type { ReportService } from '../../service/reportService'
import type { SessionService } from '../../service/sessionService'
import {
  mapLayersKey,
  settingsServiceKey,
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
  accuracy: null,
  terrain: null,
  climate: null,
  boundary: [],
  generatedFrom: { climateYears: null, terrainAt: null },
  latest: null,
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
    download: vi.fn(async () => ({ content: new Blob(['%PDF-1.7']), fileName: 'Звіт-Поле-біля-ставка-0014.pdf' })),
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
          [settingsServiceKey as symbol]: { layer: () => 'map' } as unknown as SettingsService,
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

const carpathians: Report = {
  ...report,
  accuracy: {
    marginSquareMeters: 8040,
    marginHectares: 0.804,
    percent: 3.9,
    confidence: 0.95,
    scaleShare: 0.22,
    obstructionShare: 0.78,
    worstSide: 1,
    groundShiftMeters: 4,
    basis: 'capture',
  },
  terrain: {
    status: 'available',
    slopeDegrees: 16.74,
    surfaceSquareMeters: 25_165,
    surfaceHectares: 2.5165,
    surfaceIncreasePercent: 4.42,
    minElevationMeters: 1301,
    maxElevationMeters: 1532,
    reliefMeters: 231,
    coarse: false,
  },
  climate: {
    status: 'available',
    firstYear: 2016,
    lastYear: 2025,
    seasons: [
      {
        season: 'winter',
        meanCelsius: -4.2,
        meanMinCelsius: -7.3,
        meanMaxCelsius: -1.2,
        precipitationMillimeters: 300,
        frostDays: 85.2,
        heatDays: 0,
      },
    ],
    lastSpringFrost: { month: 5, day: 13 },
    firstAutumnFrost: { month: 10, day: 3 },
    growingDays: 180.4,
  },
}

describe('ReportView blocks', () => {
  it('shows accuracy, terrain and climate with their sources', async () => {
    const { wrapper } = await mountReport({
      report: carpathians,
      plot: null,
      points,
      obstructions: [{ side: 1, reason: 'trees' }],
    })
    const text = wrapper.text()

    expect(text).toContain('Точність вимірювання')
    expect(text).toContain('± 0,80 га (3,9%)')
    expect(text).toContain('Найбільше впливає: дерева на стороні 2–3')
    expect(text).toContain('Середній ухил16,7°')
    expect(text).toMatch(/1\s301–1\s532 м/u)
    expect(text).toContain('Кадастрова (юридична) площа — це горизонтальна проєкція')
    expect(text).not.toContain('Оцінка груба')
    expect(wrapper.get('.report-climate__table').text()).toContain('Зима')
    expect(wrapper.get('.report-climate__table').text()).toContain('-4,2 °C')
    expect(text).toContain('Останній весняний заморозок: 13 травня')
    expect(text).toContain('180 днів із середньою температурою ≥ 5 °C')
    expect(wrapper.get('.report__sources').text()).toBe(
      'Дані: Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5, Copernicus DEM GLO-90',
    )
  })

  it('offers to refresh unavailable blocks and marks a coarse estimate', async () => {
    const { wrapper, service } = await mountReport({
      report: {
        ...carpathians,
        terrain: { ...(carpathians.terrain as NonNullable<Report['terrain']>), coarse: true },
        climate: { status: 'unavailable', firstYear: 2016, lastYear: 2025, seasons: [], lastSpringFrost: null, firstAutumnFrost: null, growingDays: 0 },
      },
      plot: null,
      points,
      obstructions: [],
    })

    expect(wrapper.text()).toContain('Оцінка груба')
    expect(wrapper.text()).toContain('Тимчасово недоступно')
    await wrapper.get('.report-block__refresh').trigger('click')
    await flushPromises()
    expect(service.create).toHaveBeenCalledWith('plot-1')
  })

  it('says there is no data for an older report', async () => {
    const { wrapper } = await mountReport({ report, plot: null, points, obstructions: [] })

    expect(wrapper.findAll('.report-block__muted').map((item) => item.text())).toEqual([
      'Даних немає для цього звіту.',
      'Даних немає для цього звіту.',
      'Даних немає для цього звіту.',
    ])
    expect(wrapper.find('.report__sources').exists()).toBe(false)
  })
})

describe('ReportView', () => {
  it('shows the report from its own data and the boundary of its version', async () => {
    const { wrapper } = await mountReport({ report, plot: { version: 1 } as Plot, points , obstructions: [] })

    expect(wrapper.get('.report__title').text()).toBe('Звіт · Поле біля ставка')
    expect(wrapper.get('.report__meta').text()).toContain('Звіт № 0014 · Сформовано 26.10.2026')
    expect(wrapper.get('.report__tiles').text()).toContain('2,41')
    expect(wrapper.get('.report__tiles').text()).toContain('24 100')
    expect(wrapper.findAll('tbody tr')).toHaveLength(3)
    expect(wrapper.find('.report__banner').exists()).toBe(false)
    expect(wrapper.get('.report__back').attributes('href')).toBe('/plots/plot-1')
  })

  it('shows the banner for a changed plot and refreshes the report', async () => {
    const { wrapper, router, service } = await mountReport({ report, plot: { version: 2 } as Plot, points , obstructions: [] })

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
    const { wrapper } = await mountReport({ report: { ...report, plotDeleted: true }, plot: null, points: null, obstructions: [] })

    expect(wrapper.get('.report__deleted').text()).toContain('Ділянку видалено')
    expect(wrapper.find('tbody').exists()).toBe(false)
    expect(wrapper.find('.report__banner').exists()).toBe(false)
    expect(button(wrapper, 'Завантажити GeoJSON')?.attributes('disabled')).toBeDefined()
    expect(wrapper.get('.report__back').attributes('href')).toBe('/reports')
  })

  it('downloads the PDF from the backend in the current language and copies the link', async () => {
    vi.spyOn(URL, 'createObjectURL').mockImplementation(() => 'blob:report')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    const names: string[] = []
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      names.push(this.download)
    })
    const writeText = vi.fn(async () => undefined)
    Object.assign(navigator, { clipboard: { writeText } })
    const { wrapper, service } = await mountReport({ report, plot: { version: 1 } as Plot, points, obstructions: [] })

    await button(wrapper, 'Завантажити PDF')?.trigger('click')
    await flushPromises()
    expect(service.download).toHaveBeenCalledWith(report.id, 'pdf', 'uk')
    expect(names).toEqual(['Звіт-Поле-біля-ставка-0014.pdf'])

    await button(wrapper, 'Поділитися')?.trigger('click')
    await flushPromises()
    expect(writeText).toHaveBeenCalledWith(window.location.href)
  })

  it('always offers a new report, shows the climate years and links a newer report', async () => {
    const { wrapper, service, router } = await mountReport({
      report: {
        ...report,
        generatedFrom: { climateYears: { first: 2016, last: 2025 }, terrainAt: report.createdAt },
        latest: { id: 'report-newer', createdAt: new Date(2026, 10, 19) },
      },
      plot: { version: 1 } as Plot,
      points,
      obstructions: [],
    })

    expect(wrapper.get('.report__meta').text()).toContain('Сформовано 26.10.2026 · клімат за 2016–2025')
    expect(wrapper.get('.report__refresh').text()).toContain('Створить новий звіт з актуальними даними, цей лишиться без змін')
    expect(wrapper.get('.report__newer').text()).toBe('Є новіший звіт від 19.11.2026')
    expect(wrapper.get('.report__newer').attributes('href')).toBe('/reports/report-newer')

    await wrapper.get('.report__refresh-button').trigger('click')
    await flushPromises()
    expect(service.create).toHaveBeenCalledWith('plot-1')
    expect(router.currentRoute.value.fullPath).toBe('/reports/report-new')
  })

  it('explains why an older report has no GeoJSON and still downloads its PDF', async () => {
    vi.spyOn(URL, 'createObjectURL').mockImplementation(() => 'blob:report')
    vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => undefined)
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { wrapper, service } = await mountReport({ report, plot: { version: 1 } as Plot, points, obstructions: [] })

    expect(wrapper.get('.report__boundary-note').text()).toBe(
      'Цей звіт створено до збереження меж у звіті. Натисніть «Оновити звіт», щоб завантажити PDF і GeoJSON',
    )
    const geoJson = button(wrapper, 'Завантажити GeoJSON')
    expect(geoJson?.attributes('disabled')).toBeDefined()
    expect(geoJson?.attributes('aria-describedby')).toBe('report-boundary-note')

    await button(wrapper, 'Завантажити PDF')?.trigger('click')
    await flushPromises()
    expect(service.download).toHaveBeenCalledWith(report.id, 'pdf', 'uk')
    expect(wrapper.find('.report__failure').exists()).toBe(false)
  })

  it('says the boundary is unavailable for an older report of a deleted plot', async () => {
    const { wrapper } = await mountReport({
      report: { ...report, plotDeleted: true },
      plot: null,
      points: null,
      obstructions: [],
    })

    expect(wrapper.get('.report__boundary-note').text()).toBe('Межа цього звіту недоступна')
    expect(button(wrapper, 'Завантажити GeoJSON')?.attributes('disabled')).toBeDefined()
    expect(wrapper.find('.report__refresh').exists()).toBe(false)
  })

  it('shows no note when the report keeps its boundary', async () => {
    const { wrapper } = await mountReport({
      report: { ...report, boundary: points },
      plot: { version: 1 } as Plot,
      points,
      obstructions: [],
    })

    expect(wrapper.find('.report__boundary-note').exists()).toBe(false)
    expect(button(wrapper, 'Завантажити GeoJSON')?.attributes('disabled')).toBeUndefined()
  })
})

