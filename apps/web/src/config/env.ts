export interface AppConfig {
  readonly gatewayUrl: string
  readonly oidcAuthority: string
  readonly oidcClientId: string
}

export type RawEnvironment = Readonly<Record<string, string | boolean | undefined>>

const defaults = {
  VITE_GATEWAY_URL: 'http://localhost:8080',
  VITE_OIDC_AUTHORITY: 'http://localhost:8180/realms/land-plot',
  VITE_OIDC_CLIENT_ID: 'land-plot-web',
} as const

type VariableName = keyof typeof defaults

export class ConfigError extends Error {
  constructor(name: VariableName, value: string) {
    super(`${name} must be an absolute http(s) URL, got "${value}"`)
    this.name = 'ConfigError'
  }
}

export function readConfig(environment: RawEnvironment): AppConfig {
  return {
    gatewayUrl: url(environment, 'VITE_GATEWAY_URL'),
    oidcAuthority: url(environment, 'VITE_OIDC_AUTHORITY'),
    oidcClientId: text(environment, 'VITE_OIDC_CLIENT_ID'),
  }
}

function text(environment: RawEnvironment, name: VariableName): string {
  const value = environment[name]

  return typeof value === 'string' && value.trim() !== '' ? value.trim() : defaults[name]
}

function url(environment: RawEnvironment, name: VariableName): string {
  const value = text(environment, name)
  if (!URL.canParse(value) || !['http:', 'https:'].includes(new URL(value).protocol)) {
    throw new ConfigError(name, value)
  }

  return value.replace(/\/+$/, '')
}
