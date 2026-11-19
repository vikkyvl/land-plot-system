import type { AccessTokenProvider } from '../port/authSession'

import { networkError, toDomainError } from './errors'

export type HttpMethod = 'GET' | 'POST' | 'PATCH' | 'DELETE'

export type QueryValue = string | number | undefined

export interface RequestOptions {
  readonly body?: unknown
  readonly query?: Readonly<Record<string, QueryValue>>
  readonly authenticated?: boolean
}

export interface DownloadedFile {
  readonly content: Blob
  readonly fileName: string | null
}

export interface HttpClientOptions {
  readonly baseUrl: string
  readonly session: AccessTokenProvider
  readonly fetch?: typeof fetch
}

const apiPrefix = '/api/v1'
const requestIdHeader = 'X-Request-ID'
const retryAfterHeader = 'Retry-After'
const statusNoContent = 204
const statusUnauthorized = 401
const dispositionHeader = 'Content-Disposition'
const encodedFileName = /filename\*=UTF-8''([^;]+)/iu
const plainFileName = /filename="([^"]+)"/iu

export class HttpClient {
  private readonly baseUrl: string
  private readonly session: AccessTokenProvider
  private readonly fetch: typeof fetch
  private renewal: Promise<string | null> | null = null

  constructor(options: HttpClientOptions) {
    this.baseUrl = options.baseUrl.replace(/\/+$/, '')
    this.session = options.session
    this.fetch = options.fetch ?? globalThis.fetch.bind(globalThis)
  }

  async request<T>(method: HttpMethod, path: string, options: RequestOptions = {}): Promise<T> {
    const response = await this.checked(method, path, options)

    if (response.status === statusNoContent) {
      return undefined as T
    }

    return (await response.json()) as T
  }

  async download(path: string, options: RequestOptions = {}): Promise<DownloadedFile> {
    const response = await this.checked('GET', path, options)

    return { content: await response.blob(), fileName: fileNameOf(response.headers.get(dispositionHeader)) }
  }

  private async checked(method: HttpMethod, path: string, options: RequestOptions): Promise<Response> {
    const authenticated = options.authenticated ?? true
    const url = this.url(path, options.query)
    const token = authenticated ? await this.session.accessToken() : null
    let response = await this.send(method, url, options.body, token)

    if (response.status === statusUnauthorized && authenticated) {
      const freshToken = await this.freshToken(token)
      if (freshToken !== null) {
        response = await this.send(method, url, options.body, freshToken)
      }
    }

    if (!response.ok) {
      throw toDomainError(response.status, await readJson(response), {
        requestId: response.headers.get(requestIdHeader),
        retryAfter: response.headers.get(retryAfterHeader),
      })
    }

    return response
  }

  private async freshToken(rejectedToken: string | null): Promise<string | null> {
    if (this.renewal !== null) {
      return this.renewal
    }

    const currentToken = await this.session.accessToken()
    if (currentToken !== null && currentToken !== rejectedToken) {
      return currentToken
    }

    if (this.renewal === null) {
      this.renewal = this.session.renewAccessToken().finally(() => {
        this.renewal = null
      })
    }

    return this.renewal
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

function fileNameOf(disposition: string | null): string | null {
  const encoded = encodedFileName.exec(disposition ?? '')?.[1]
  if (encoded !== undefined) {
    try {
      return decodeURIComponent(encoded)
    } catch {
      return null
    }
  }

  return plainFileName.exec(disposition ?? '')?.[1] ?? null
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    return null
  }
}
