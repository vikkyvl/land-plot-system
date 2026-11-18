import type { AuthSession } from '../../port/authSession'
import { HttpClient } from '../http'

export const gatewayUrl = 'http://gateway.test'

export interface RecordedRequest {
  readonly method: string
  readonly url: string
  readonly authorization: string | null
  readonly contentType: string | null
  readonly body: unknown
}

export interface FakeReply {
  readonly status: number
  readonly body?: unknown
  readonly headers?: Record<string, string>
}

export class FakeSession implements AuthSession {
  renewals = 0
  private token: string | null
  private readonly renewedToken: string | null

  constructor(token: string | null, renewedToken: string | null = null) {
    this.token = token
    this.renewedToken = renewedToken
  }

  async accessToken(): Promise<string | null> {
    return this.token
  }

  async renewAccessToken(): Promise<string | null> {
    this.renewals += 1
    this.token = this.renewedToken

    return this.renewedToken
  }
}

export class FakeServer {
  readonly requests: RecordedRequest[] = []
  private readonly replies: FakeReply[] = []

  reply(...replies: FakeReply[]): this {
    this.replies.push(...replies)

    return this
  }

  readonly fetch = async (input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> => {
    const headers = new Headers(init.headers)
    this.requests.push({
      method: init.method ?? 'GET',
      url: String(input),
      authorization: headers.get('Authorization'),
      contentType: headers.get('Content-Type'),
      body: typeof init.body === 'string' ? JSON.parse(init.body) : undefined,
    })

    const next = this.replies.shift()
    if (next === undefined) {
      throw new Error('unexpected request')
    }

    const body = next.body === undefined ? null : JSON.stringify(next.body)

    return new Response(body, { status: next.status, headers: next.headers })
  }

  client(session: AuthSession = new FakeSession(fakeToken('first'))): HttpClient {
    return new HttpClient({ baseUrl: `${gatewayUrl}/`, session, fetch: this.fetch })
  }
}

export function fakeToken(label: string): string {
  return ['fake', label, 'access', 'token'].join('-')
}
