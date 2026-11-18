import { describe, expect, it } from 'vitest'

import { ErrorCode } from '../domain/errorCode'

import { ReportsClient } from './reportsClient'
import { FakeServer, gatewayUrl } from './testing/fakeServer'

const plotId = '7f1c2a9e-0000-4000-8000-000000000001'

const reportDto = {
  id: 'report-14',
  plot_id: plotId,
  plot_name: 'Поле біля ставка',
  plot_version: 2,
  area_m2: 24812.4,
  area_ares: 248.124,
  area_hectares: 2.48124,
  perimeter_m: 648.2,
  created_at: '2026-10-27T12:30:00Z',
  plot_deleted: true,
}

function reports(server: FakeServer): ReportsClient {
  return new ReportsClient(server.client())
}

describe('ReportsClient', () => {
  it('maps a report including the deleted plot mark', async () => {
    const server = new FakeServer().reply({ status: 200, body: reportDto })

    expect(await reports(server).get('report-14')).toEqual({
      id: 'report-14',
      plotId,
      plotName: 'Поле біля ставка',
      plotVersion: 2,
      measurement: { area: { squareMeters: 24812.4, ares: 248.124, hectares: 2.48124 }, perimeterMeters: 648.2 },
      createdAt: new Date('2026-10-27T12:30:00Z'),
      plotDeleted: true,
      accuracy: null,
      terrain: null,
      climate: null,
    })
  })

  it('maps the terrain and climate blocks', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: {
        ...reportDto,
        terrain: {
          status: 'available',
          slope_degrees: 16.7,
          surface_area_m2: 25_900,
          surface_area_hectares: 2.59,
          surface_increase_percent: 4.42,
          min_elevation_m: 1301,
          max_elevation_m: 1532,
          relief_m: 231,
          coarse: false,
          source: 'Open-Meteo.com (CC BY 4.0), Copernicus DEM GLO-90',
        },
        climate: {
          status: 'unavailable',
          first_year: 2016,
          last_year: 2025,
          seasons: [],
          last_spring_frost: '04-05',
          first_autumn_frost: null,
          growing_days: 0,
          source: 'Open-Meteo.com (CC BY 4.0), Copernicus C3S ERA5',
        },
      },
    })

    const report = await reports(server).get('report-14')

    expect(report.terrain).toEqual({
      status: 'available',
      slopeDegrees: 16.7,
      surfaceSquareMeters: 25_900,
      surfaceHectares: 2.59,
      surfaceIncreasePercent: 4.42,
      minElevationMeters: 1301,
      maxElevationMeters: 1532,
      reliefMeters: 231,
      coarse: false,
    })
    expect(report.climate).toEqual({
      status: 'unavailable',
      firstYear: 2016,
      lastYear: 2025,
      seasons: [],
      lastSpringFrost: { month: 4, day: 5 },
      firstAutumnFrost: null,
      growingDays: 0,
    })
  })

  it('lists reports with the plot filter and limit in the query', async () => {
    const server = new FakeServer().reply({ status: 200, body: [reportDto] })

    const list = await reports(server).list({ plotId, limit: 10 })

    expect(list).toHaveLength(1)
    expect(server.requests[0]?.url).toBe(`${gatewayUrl}/api/v1/reports?plot_id=${plotId}&limit=10`)
  })

  it('lists all reports without query parameters', async () => {
    const server = new FakeServer().reply({ status: 200, body: [] })

    await reports(server).list()

    expect(server.requests[0]?.url).toBe(`${gatewayUrl}/api/v1/reports`)
  })

  it('creates a report for a plot', async () => {
    const server = new FakeServer().reply({ status: 201, body: reportDto })

    await reports(server).create(plotId)

    expect(server.requests[0]).toMatchObject({ method: 'POST', body: { plot_id: plotId } })
  })

  it('reports 400 invalid_query', async () => {
    const server = new FakeServer().reply({ status: 400, body: { code: 'invalid_query', message: 'limit must be 1..100' } })

    await expect(reports(server).list({ limit: 500 })).rejects.toMatchObject({
      kind: 'invalid_request',
      code: ErrorCode.InvalidQuery,
    })
  })
})
