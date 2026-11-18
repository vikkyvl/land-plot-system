import type { SignInMethod } from '../domain/session'

export interface AccessTokenProvider {
  accessToken(): Promise<string | null>
  renewAccessToken(): Promise<string | null>
}

export interface AuthSession extends AccessTokenProvider {
  isSignedIn(): Promise<boolean>
  signIn(method: SignInMethod, returnTo: string): Promise<void>
  completeSignIn(): Promise<string>
  signOut(): Promise<void>
}
