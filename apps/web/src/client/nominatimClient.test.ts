import { describe, expect, it } from 'vitest'

import { ErrorCode } from '../domain/errorCode'

import { minimumRequestInterval, NominatimClient } from './nominatimClient'
import { FakeServer } from './testing/fakeServer'

const nominatimUrl = 'http://nominatim.test'

const bilaTserkva = {
  name: 'Біла Церква',
  display_name: 'Біла Церква, Білоцерківська міська громада, Київська область, Україна',
  lat: '49.7957',
  lon: '30.1310',
  boundingbox: ['49.7581', '49.8364', '30.0613', '30.1907'],
}

function clock() {
  let time = 10_000
  const waits: number[] = []

  return {
    waits,
    now: () => time,
    wait: async (milliseconds: number) => {
      waits.push(milliseconds)
      time += milliseconds
    },
    advance: (milliseconds: number) => {
      time += milliseconds
    },
  }
}

function client(server: FakeServer, time = clock()) {
  return new NominatimClient({ baseUrl: `${nominatimUrl}/`, fetch: server.fetch, now: time.now, wait: time.wait })
}

describe('NominatimClient', () => {
  it('searches Ukrainian places in Ukrainian and maps them to the domain', async () => {
    const server = new FakeServer().reply({ status: 200, body: [bilaTserkva] })

    const places = await client(server).search('Біла Церква')

    const url = new URL(server.requests[0]?.url ?? '')
    expect(url.origin + url.pathname).toBe(`${nominatimUrl}/search`)
    expect(Object.fromEntries(url.searchParams)).toEqual({
      q: 'Біла Церква',
      format: 'jsonv2',
      limit: '5',
      'accept-language': 'uk',
      countrycodes: 'ua',
    })
    expect(server.requests[0]?.authorization).toBeNull()
    expect(places).toEqual([
      {
        name: 'Біла Церква',
        description: 'Білоцерківська міська громада, Київська область, Україна',
        coordinate: { latitude: 49.7957, longitude: 30.131 },
        bounds: { south: 49.7581, west: 30.0613, north: 49.8364, east: 30.1907 },
      },
    ])
  })

  it('sends at most one request per second', async () => {
    const time = clock()
    const server = new FakeServer().reply({ status: 200, body: [] }, { status: 200, body: [] }, { status: 200, body: [] })
    const nominatim = client(server, time)

    await nominatim.search('Біла')
    time.advance(300)
    await nominatim.search('Біла Ц')
    time.advance(2000)
    await nominatim.search('Біла Церква')

    expect(time.waits).toEqual([minimumRequestInterval - 300])
    expect(server.requests).toHaveLength(3)
  })

  it('reports an unavailable service on an error status', async () => {
    const server = new FakeServer().reply({ status: 429, body: {} })

    await expect(client(server).search('Київ')).rejects.toMatchObject({
      kind: 'unavailable',
      code: ErrorCode.ServiceUnavailable,
    })
  })

  it('does not send a search that was cancelled while waiting for its turn', async () => {
    const server = new FakeServer()
    const controller = new AbortController()
    controller.abort()

    await expect(client(server).search('Київ', controller.signal)).rejects.toThrow()
    expect(server.requests).toHaveLength(0)
  })
})

describe('NominatimClient trace context', () => {
  it('never sends traceparent to the external geocoder', async () => {
    const server = new FakeServer().reply({ status: 200, body: [bilaTserkva] })

    await client(server).search('Біла Церква')

    expect(server.traceparents).toEqual([null])
  })
})
