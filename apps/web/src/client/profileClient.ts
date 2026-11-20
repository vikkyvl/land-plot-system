import type { Profile } from '../domain/profile'
import type { ProfileGateway } from '../port/profileGateway'

import { toProfile, type ProfileDto } from './dto'
import type { HttpClient } from './http'

const profilePath = '/users/profile'

export class ProfileClient implements ProfileGateway {
  private readonly http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  async get(): Promise<Profile> {
    return toProfile(await this.http.request<ProfileDto>('GET', profilePath))
  }
}
