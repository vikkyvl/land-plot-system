import type { Profile } from '../domain/profile'

export interface ProfileGateway {
  get(): Promise<Profile>
}
