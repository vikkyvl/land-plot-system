import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import type { Report } from '../../domain/report'
import type { ReportService } from '../../service/reportService'
import type { SessionService } from '../../service/sessionService'
import { reportServiceKey, sessionServiceKey } from '../composables/services'

import ReportsView from './ReportsView.vue'

function report(id: string, name: string, deleted = false): Report {
  return {
    id,
    plotId: 'plot-1',
    plotName: name,
    plotVersion: 2,
    measurement: { area: { squareMeters: 24810, ares: 248.1, hectares: 2.481 }, perimeterMeters: 612 },
    createdAt: new Date(2026, 9, 26),
    plotDeleted: deleted,
    accuracy: null,
    terrain: null,
    climate: null,
    boundary: [],
    generatedFrom: { climateYears: null, terrainAt: null },
    latest: null,
  }
}

async function mountReports(list: () => Promise<Report[]>, path = '/reports') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/reports', component: ReportsView },
      { path: '/reports/:id', component: { template: '<p>report</p>' } },
      { path: '/plots', component: { template: '<p>plots</p>' } },
      { path: '/login', name: 'login', component: { template: '<p>login</p>' } },
    ],
  })
  await router.push(path)
  const service = { list: vi.fn(list) }
  const wrapper = mount(
    { template: '<RouterView />' },
    {
      global: {
        plugins: [createPinia(), router],
        provide: {
          [reportServiceKey as symbol]: service as unknown as ReportService,
          [sessionServiceKey as symbol]: {} as SessionService,
        },
      },
    },
  )
  await flushPromises()

  return { wrapper, service }
}

describe('ReportsView', () => {
  it('lists reports with number, date, version, area and the deleted badge', async () => {
    const { wrapper } = await mountReports(async () => [
      report('0014abcd', 'Поле біля ставка'),
      report('0015abcd', 'Дача в Ірпені', true),
    ])

    const cards = wrapper.findAll('.reports__card')
    expect(cards[0]?.text()).toContain('№ 0014 · 26.10.2026 · версія 2')
    expect(cards[0]?.text()).toContain('2,48 га')
    expect(cards[0]?.attributes('href')).toBe('/reports/0014abcd')
    expect(cards[0]?.find('.reports__badge').exists()).toBe(false)
    expect(cards[1]?.get('.reports__badge').text()).toBe('Ділянку видалено')
  })

  it('filters by the plot from the query', async () => {
    const { wrapper, service } = await mountReports(async () => [report('0014abcd', 'Поле')], '/reports?plot_id=plot-1')

    expect(service.list).toHaveBeenCalledWith({ plotId: 'plot-1' })
    expect(wrapper.text()).toContain('Звіти ділянки «Поле»')
  })

  it('shows the empty state', async () => {
    const { wrapper } = await mountReports(async () => [])

    expect(wrapper.get('.reports__empty').text()).toContain('Звітів ще немає')
  })
})
