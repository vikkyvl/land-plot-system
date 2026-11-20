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
      boundary: [],
      generatedFrom: { climateYears: null, terrainAt: null },
      latest: null,
    })
  })

  it('reads the boundary, what the report was generated from and the latest report', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: {
        ...reportDto,
        boundary: [{ lat: 48.14, lon: 24.48 }],
        generated_from: { climate_first_year: 2016, climate_last_year: 2025, terrain_at: '2026-11-18T12:00:00Z' },
        latest_report_id: 'report-15',
        latest_report_created_at: '2026-11-19T08:00:00Z',
      },
    })

    const report = await reports(server).get('report-14')

    expect(report.boundary).toEqual([{ latitude: 48.14, longitude: 24.48 }])
    expect(report.generatedFrom).toEqual({
      climateYears: { first: 2016, last: 2025 },
      terrainAt: new Date('2026-11-18T12:00:00Z'),
    })
    expect(report.latest).toEqual({ id: 'report-15', createdAt: new Date('2026-11-19T08:00:00Z') })
  })

  it('downloads the PDF in the current language with its file name', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: '%PDF-1.7',
      headers: {
        'Content-Type': 'application/pdf',
        'Content-Disposition': `attachment; filename="____-14.pdf"; filename*=UTF-8''%D0%97%D0%B2%D1%96%D1%82-14.pdf`,
      },
    })

    const file = await reports(server).download('report-14', 'pdf', 'uk')

    expect(server.requests[0]?.url).toBe(`${gatewayUrl}/api/v1/reports/report-14/pdf?lang=uk`)
    expect(file.fileName).toBe('Звіт-14.pdf')
    expect(await file.content.text()).toContain('%PDF-1.7')
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
