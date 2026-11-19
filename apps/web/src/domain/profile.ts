export const Scope = {
  PlotsRead: 'plots:read',
  PlotsWrite: 'plots:write',
  ReportsRead: 'reports:read',
  ReportsWrite: 'reports:write',
} as const

export type Scope = (typeof Scope)[keyof typeof Scope]

export interface Profile {
  readonly subject: string
  readonly email: string | null
  readonly name: string | null
  readonly scopes: readonly string[]
  readonly createdAt: Date
  readonly updatedAt: Date
}

export function hasScope(profile: Profile, scope: Scope): boolean {
  return profile.scopes.includes(scope)
}

export function displayName(profile: Profile): string {
  return profile.name || profile.email || profile.subject
}
