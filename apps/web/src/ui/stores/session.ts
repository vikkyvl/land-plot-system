import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { displayName, type Profile } from '../../domain/profile'
import type { SignInMethod } from '../../domain/session'
import { useSessionService } from '../composables/services'

export const useSessionStore = defineStore('session', () => {
  const service = useSessionService()
  const profile = ref<Profile | null>(null)
  const signedIn = ref(false)
  const userName = computed(() => (profile.value === null ? null : displayName(profile.value)))

  async function refresh(): Promise<void> {
    signedIn.value = await service.isSignedIn()
    profile.value = null
    if (signedIn.value) {
      profile.value = await service.currentProfile().catch(() => null)
    }
  }

  function signIn(method: SignInMethod, returnTo?: string | null): Promise<void> {
    return service.signIn(method, returnTo)
  }

  async function signOut(): Promise<void> {
    await service.signOut()
  }

  return { profile, signedIn, userName, refresh, signIn, signOut }
})
