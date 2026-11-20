import { describe, expect, it } from 'vitest'

import { ProfileClient } from './profileClient'
import { FakeServer, gatewayUrl } from './testing/fakeServer'

describe('ProfileClient', () => {
  it('maps the profile and fills missing optional fields', async () => {
    const server = new FakeServer().reply({
      status: 200,
      body: {
        sub: 'user-1',
        email: 'taras.melnyk@example.com',
        scopes: ['openid', 'plots:read'],
        created_at: '2026-10-01T10:00:00Z',
        updated_at: '2026-10-02T10:00:00Z',
      },
    })

    const profile = await new ProfileClient(server.client()).get()

    expect(server.requests[0]?.url).toBe(`${gatewayUrl}/api/v1/users/profile`)
    expect(profile).toEqual({
      subject: 'user-1',
      email: 'taras.melnyk@example.com',
      name: null,
      scopes: ['openid', 'plots:read'],
      createdAt: new Date('2026-10-01T10:00:00Z'),
      updatedAt: new Date('2026-10-02T10:00:00Z'),
    })
  })
})
