import { describe, expect, it } from 'vitest'

import { DomainError } from '../domain/error'
import { ErrorCode } from '../domain/errorCode'

import { HttpClient } from './http'
import { FakeServer, FakeSession, fakeToken, gatewayUrl } from './testing/fakeServer'

const firstToken = fakeToken('first')
const renewedToken = fakeToken('renewed')

async function failure(promise: Promise<unknown>): Promise<DomainError> {
  const error = await promise.then(
    () => null,
    (reason: unknown) => reason,
  )
  expect(error).toBeInstanceOf(DomainError)

  return error as DomainError
}

describe('HttpClient', () => {
  it('sends the bearer token, JSON body and query to the gateway API', async () => {
    const server = new FakeServer().reply({ status: 200, body: { ok: true } })

    const result = await server.client().request('POST', '/things', { body: { name: 'a' }, query: { limit: 5, skip: undefined } })

    expect(result).toEqual({ ok: true })
    expect(server.requests).toEqual([
      {
        method: 'POST',
        url: `${gatewayUrl}/api/v1/things?limit=5`,
        authorization: `Bearer ${firstToken}`,
        contentType: 'application/json',
        body: { name: 'a' },
      },
    ])
  })

  it('does not send a token to public routes', async () => {
    const server = new FakeServer().reply({ status: 200, body: {} })

    await server.client().request('POST', '/public', { body: {}, authenticated: false })

    expect(server.requests[0]?.authorization).toBeNull()
  })

  it('returns undefined for 204 No Content', async () => {
    const server = new FakeServer().reply({ status: 204 })

    await expect(server.client().request('DELETE', '/things/1')).resolves.toBeUndefined()
  })

  it('renews the token once on 401 and repeats the request', async () => {
    const session = new FakeSession(firstToken, renewedToken)
    const server = new FakeServer().reply({ status: 401, body: { code: 'unauthorized' } }, { status: 200, body: [] })

    await expect(server.client(session).request('GET', '/plots')).resolves.toEqual([])

    expect(session.renewals).toBe(1)
    expect(server.requests.map((request) => request.authorization)).toEqual([
      `Bearer ${firstToken}`,
      `Bearer ${renewedToken}`,
    ])
  })

  it('reports unauthorized when the renewed token is rejected too', async () => {
    const session = new FakeSession(firstToken, renewedToken)
    const server = new FakeServer().reply(
      { status: 401, body: { code: 'unauthorized' } },
      { status: 401, body: { code: 'unauthorized', message: 'invalid token', request_id: 'req-2' } },
    )

    const error = await failure(server.client(session).request('GET', '/plots'))

    expect(session.renewals).toBe(1)
    expect(server.requests).toHaveLength(2)
    expect(error).toMatchObject({ kind: 'unauthorized', code: ErrorCode.Unauthorized, requestId: 'req-2' })
  })

  it('reports unauthorized without a second request when renewal fails', async () => {
    const session = new FakeSession(firstToken, null)
    const server = new FakeServer().reply({ status: 401, body: { code: 'unauthorized' } })

    const error = await failure(server.client(session).request('GET', '/plots'))

    expect(server.requests).toHaveLength(1)
    expect(error.kind).toBe('unauthorized')
  })

  it('does not renew the token on 403 insufficient_scope', async () => {
    const session = new FakeSession(firstToken, renewedToken)
    const server = new FakeServer().reply({
      status: 403,
      body: { code: 'insufficient_scope', message: 'scope plots:write is required', request_id: 'req-3' },
    })

    const error = await failure(server.client(session).request('POST', '/plots', { body: {} }))

    expect(session.renewals).toBe(0)
    expect(error).toMatchObject({ kind: 'forbidden', code: ErrorCode.InsufficientScope, requestId: 'req-3' })
  })

  it('reports a network error when the gateway cannot be reached', async () => {
    const http = new HttpClient({
      baseUrl: gatewayUrl,
      session: new FakeSession(firstToken),
      fetch: () => Promise.reject(new TypeError('Failed to fetch')),
    })

    const error = await failure(http.request('GET', '/plots'))

    expect(error).toMatchObject({ kind: 'network', code: ErrorCode.NetworkError })
  })

  it('takes the request id from the header when the body has none', async () => {
    const server = new FakeServer().reply({ status: 502, body: 'not json', headers: { 'X-Request-ID': 'req-4' } })

    const error = await failure(server.client().request('GET', '/plots'))

    expect(error).toMatchObject({ kind: 'unavailable', requestId: 'req-4' })
  })
})
