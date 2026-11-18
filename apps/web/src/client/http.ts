import type { AuthSession } from '../port/authSession'

import { networkError, toDomainError } from './errors'

export type HttpMethod = 'GET' | 'POST' | 'PATCH' | 'DELETE'

export type QueryValue = string | number | undefined

export interface RequestOptions {
  readonly body?: unknown
  readonly query?: Readonly<Record<string, QueryValue>>
  readonly authenticated?: boolean
}

export interface HttpClientOptions {
  readonly baseUrl: string
  readonly session: AuthSession
  readonly fetch?: typeof fetch
}

const apiPrefix = '/api/v1'
const requestIdHeader = 'X-Request-ID'
const statusNoContent = 204
const statusUnauthorized = 401

export class HttpClient {
  private readonly baseUrl: string
  private readonly session: AuthSession
  private readonly fetch: typeof fetch

  constructor(options: HttpClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '')
    this.session = options.session
    this.fetch = options.fetch ?? globalThis.fetch.bind(globalThis)
  }

  async request<T>(method: HttpMethod, path: string, options: RequestOptions = {}): Promise<T> {
    const authenticated = options.authenticated ?? true
    const url = this.url(path, options.query)
    const token = authenticated ? await this.session.accessToken() : null
    let response = await this.send(method, url, options.body, token)

    if (response.status === statusUnauthorized && authenticated) {
      const renewedToken = await this.session.renewAccessToken()
      if (renewedToken !== null) {
        response = await this.send(method, url, options.body, renewedToken)
      }
    }

    if (!response.ok) {
      throw toDomainError(response.status, await readJson(response), response.headers.get(requestIdHeader))
    }

    if (response.status === statusNoContent) {
      return undefined as T
    }

    return (await response.json()) as T
  }

  private url(path: string, query: RequestOptions['query']): string {
    const url = new URL(`${this.baseUrl}${apiPrefix}${path}`)
    for (const [name, value] of Object.entries(query ?? {})) {
      if (value !== undefined) {
        url.searchParams.set(name, String(value))
      }
    }

    return url.toString()
  }

  private async send(method: HttpMethod, url: string, body: unknown, token: string | null): Promise<Response> {
    const headers = new Headers({ Accept: 'application/json' })
    if (body !== undefined) {
      headers.set('Content-Type', 'application/json')
    }
    if (token !== null) {
      headers.set('Authorization', `Bearer ${token}`)
    }

    try {
      return await this.fetch(url, {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body),
      })
    } catch {
      throw networkError()
    }
  }
}

export function pathOf(...segments: string[]): string {
  return segments.map((segment) => `/${encodeURIComponent(segment)}`).join('')
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    return null
  }
}
