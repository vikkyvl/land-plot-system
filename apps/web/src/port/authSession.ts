export interface AuthSession {
  accessToken(): Promise<string | null>
  renewAccessToken(): Promise<string | null>
}
