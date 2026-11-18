import { describe, expect, it } from 'vitest'

import { ConfigError, readConfig } from './env'

describe('readConfig', () => {
  it('uses local defaults when nothing is set', () => {
    expect(readConfig({})).toEqual({
      gatewayUrl: 'http://localhost:8080',
      oidcAuthority: 'http://localhost:8180/realms/land-plot',
      oidcClientId: 'land-plot-web',
    })
  })

  it('reads and trims the variables', () => {
    expect(
      readConfig({
        VITE_GATEWAY_URL: ' https://api.example.test/ ',
        VITE_OIDC_AUTHORITY: 'https://id.example.test/realms/land-plot',
        VITE_OIDC_CLIENT_ID: 'web',
      }),
    ).toEqual({
      gatewayUrl: 'https://api.example.test',
      oidcAuthority: 'https://id.example.test/realms/land-plot',
      oidcClientId: 'web',
    })
  })

  it.each(['localhost:8080', 'ftp://files.example.test', 'not a url'])('rejects %j as the gateway URL', (value) => {
    expect(() => readConfig({ VITE_GATEWAY_URL: value })).toThrow(ConfigError)
  })
})
