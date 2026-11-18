import type { Profile } from '../domain/profile'
import type { SignInMethod } from '../domain/session'
import type { AuthSession } from '../port/authSession'
import type { ProfileGateway } from '../port/profileGateway'

import { safeReturnPath } from './returnPath'

export class SessionService {
  private readonly auth: AuthSession
  private readonly profiles: ProfileGateway

  constructor(auth: AuthSession, profiles: ProfileGateway) {
    this.auth = auth
    this.profiles = profiles
  }

  isSignedIn(): Promise<boolean> {
    return this.auth.isSignedIn()
  }

  signIn(method: SignInMethod, returnTo?: string | null): Promise<void> {
    return this.auth.signIn(method, safeReturnPath(returnTo))
  }

  async completeSignIn(): Promise<string> {
    return safeReturnPath(await this.auth.completeSignIn())
  }

  signOut(): Promise<void> {
    return this.auth.signOut()
  }

  async currentProfile(): Promise<Profile | null> {
    if (!(await this.auth.isSignedIn())) {
      return null
    }

    return this.profiles.get()
  }
}
