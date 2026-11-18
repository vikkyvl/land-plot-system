import { describe, expect, it } from 'vitest'

import { displayName, hasScope, Scope, type Profile } from './profile'

const profile: Profile = {
  subject: 'user-1',
  email: 'taras.melnyk@example.com',
  name: 'Тарас Мельник',
  scopes: ['openid', Scope.PlotsRead],
  createdAt: new Date(0),
  updatedAt: new Date(0),
}

describe('profile', () => {
  it('checks scopes from the profile', () => {
    expect(hasScope(profile, Scope.PlotsRead)).toBe(true)
    expect(hasScope(profile, Scope.PlotsWrite)).toBe(false)
  })

  it('shows the name, then the email, then the subject', () => {
    expect(displayName(profile)).toBe('Тарас Мельник')
    expect(displayName({ ...profile, name: null })).toBe('taras.melnyk@example.com')
    expect(displayName({ ...profile, name: null, email: null })).toBe('user-1')
  })
})
