import { describe, expect, it } from 'vitest'

import { ErrorCode } from '../domain/errorCode'

import { PlotsClient } from './plotsClient'
import { FakeServer, gatewayUrl } from './testing/fakeServer'

const plotId = '7f1c2a9e-0000-4000-8000-000000000001'

const plotDto = {
  id: plotId,
  owner_id: 'owner-1',
  name: 'Поле біля ставка',
  points: [
    { lat: 50.45, lon: 30.52 },
    { lat: 50.451, lon: 30.523 },
    { lat: 50.449, lon: 30.524 },
  ],
  area_m2: 24812.4,
  area_ares: 248.124,
  area_hectares: 2.48124,
  perimeter_m: 648.2,
  version: 3,
  status: 'CONFIRMED',
  created_at: '2026-10-14T09:00:00Z',
  updated_at: '2026-10-27T12:30:00Z',
}

const measurement = {
  area: { squareMeters: 24812.4, ares: 248.124, hectares: 2.48124 },
  perimeterMeters: 648.2,
}

function plots(server: FakeServer): PlotsClient {
  return new PlotsClient(server.client())
}

describe('PlotsClient', () => {
  it('maps a plot from the API to the domain', async () => {
    const server = new FakeServer().reply({ status: 200, body: plotDto })

    const plot = await plots(server).get(plotId)

    expect(server.requests[0]).toMatchObject({ method: 'GET', url: `${gatewayUrl}/api/v1/plots/${plotId}` })
    expect(plot).toEqual({
      id: plotId,
      ownerId: 'owner-1',
      name: 'Поле біля ставка',
      points: [
        { latitude: 50.45, longitude: 30.52 },
        { latitude: 50.451, longitude: 30.523 },
        { latitude: 50.449, longitude: 30.524 },
      ],
      measurement,
      capture: null,
      obstructions: [],
      accuracy: null,
      version: 3,
      status: 'CONFIRMED',
      createdAt: new Date('2026-10-14T09:00:00Z'),
      updatedAt: new Date('2026-10-27T12:30:00Z'),
    })
  })

  it('lists plots', async () => {
    const server = new FakeServer().reply({ status: 200, body: [plotDto] })

    const list = await plots(server).list()

    expect(list.map((plot) => plot.id)).toEqual([plotId])
  })

  it('creates a plot with points in the API format', async () => {
    const server = new FakeServer().reply({ status: 201, body: { ...plotDto, status: 'PENDING', version: 1 } })

    const plot = await plots(server).create({
      name: 'Поле',
      points: [
        { latitude: 50.45, longitude: 30.52 },
        { latitude: 50.451, longitude: 30.523 },
      ],
      capture: { zoom: 18, layer: 'satellite', pointer: 'touch' },
      obstructions: [{ side: 1, reason: 'trees' }],
    })

    expect(plot.status).toBe('PENDING')
    expect(server.requests[0]).toMatchObject({
      method: 'POST',
      url: `${gatewayUrl}/api/v1/plots`,
      body: {
        name: 'Поле',
        points: [
          { lat: 50.45, lon: 30.52 },
          { lat: 50.451, lon: 30.523 },
        ],
        capture: { zoom: 18, layer: 'satellite', pointer: 'touch' },
        obstructions: [{ side: 1, reason: 'trees' }],
      },
    })
  })

  it('sends only the changed fields on update', async () => {
    const server = new FakeServer().reply({ status: 200, body: plotDto })

    await plots(server).update(plotId, { name: 'Нова назва' })

    expect(server.requests[0]).toMatchObject({ method: 'PATCH', body: { name: 'Нова назва' } })
  })

  it('deletes a plot', async () => {
    const server = new FakeServer().reply({ status: 204 })

    await expect(plots(server).remove(plotId)).resolves.toBeUndefined()
    expect(server.requests[0]).toMatchObject({ method: 'DELETE', url: `${gatewayUrl}/api/v1/plots/${plotId}` })
  })

  it('reports 410 plot_deleted for a deleted plot', async () => {
    const server = new FakeServer().reply({ status: 410, body: { code: 'plot_deleted', message: 'plot was deleted' } })

    await expect(plots(server).versions(plotId)).rejects.toMatchObject({ kind: 'gone', code: ErrorCode.PlotDeleted })
  })

  it('maps the version history', async () => {
    const version = {
      version: 2,
      name: 'Поле',
      points: plotDto.points,
      area_m2: plotDto.area_m2,
      area_ares: plotDto.area_ares,
      area_hectares: plotDto.area_hectares,
      perimeter_m: plotDto.perimeter_m,
      created_at: '2026-10-20T08:00:00Z',
    }
    const server = new FakeServer().reply({ status: 200, body: [version] })

    const versions = await plots(server).versions(plotId)

    expect(server.requests[0]?.url).toBe(`${gatewayUrl}/api/v1/plots/${plotId}/versions`)
    expect(versions).toEqual([
      {
        version: 2,
        name: 'Поле',
        points: plotDto.points.map(({ lat, lon }) => ({ latitude: lat, longitude: lon })),
        measurement,
        obstructions: [],
        accuracy: null,
        createdAt: new Date('2026-10-20T08:00:00Z'),
      },
    ])
  })

  it('previews a valid polygon without a token', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: {
        area_m2: 24812.4,
        area_ares: 248.124,
        area_hectares: 2.48124,
        perimeter_m: 648.2,
        accuracy: {
          area_m2_margin: 520,
          area_hectares_margin: 0.052,
          percent: 2.1,
          confidence: 0.95,
          contributions: { scale: 0.3, obstructions: 0.7 },
          worst_side: 2,
          ground_shift_m: 4,
          basis: 'capture',
        },
        valid: true,
        errors: [],
      },
    })

    const preview = await plots(server).preview([{ latitude: 50.45, longitude: 30.52 }], {
      capture: null,
      obstructions: [{ side: 2, reason: 'unclear' }],
    })

    expect(server.requests[0]).toMatchObject({
      method: 'POST',
      url: `${gatewayUrl}/api/v1/plots/preview`,
      authorization: null,
      body: { points: [{ lat: 50.45, lon: 30.52 }], obstructions: [{ side: 2, reason: 'unclear' }] },
    })
    expect(server.requests[0]?.body).not.toHaveProperty('capture')
    expect(preview).toEqual({
      valid: true,
      measurement,
      accuracy: {
        marginSquareMeters: 520,
        marginHectares: 0.052,
        percent: 2.1,
        confidence: 0.95,
        scaleShare: 0.3,
        obstructionShare: 0.7,
        worstSide: 2,
        groundShiftMeters: 4,
        basis: 'capture',
      },
      problems: [],
    })
  })

  it('previews an invalid polygon with problems and their details', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: {
        area_m2: null,
        area_ares: null,
        area_hectares: null,
        perimeter_m: null,
        valid: false,
        errors: [
          { code: 'invalid_coordinate', message: 'latitude out of range', details: { index: 1 } },
          { code: 'duplicate_point', message: 'duplicate point', details: { indices: [0, 2] } },
          { code: 'too_many_points', message: 'too many points' },
        ],
      },
    })

    const preview = await plots(server).preview([], { capture: null, obstructions: [] })

    expect(preview).toEqual({
      valid: false,
      measurement: null,
      accuracy: null,
      problems: [
        { code: 'invalid_coordinate', message: 'latitude out of range', details: { kind: 'invalid_coordinate', index: 1 } },
        { code: 'duplicate_point', message: 'duplicate point', details: { kind: 'duplicate_point', indices: [0, 2] } },
        { code: 'too_many_points', message: 'too many points', details: null },
      ],
    })
  })

  it('reports 429 from the preview rate limit', async () => {
    const server = new FakeServer().reply({ status: 429, body: { code: 'rate_limited', message: 'too many requests' } })

    await expect(plots(server).preview([], { capture: null, obstructions: [] })).rejects.toMatchObject({ kind: 'rate_limited', code: ErrorCode.RateLimited })
  })
})
