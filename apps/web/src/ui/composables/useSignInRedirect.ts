import { useRoute, useRouter } from 'vue-router'

export function useSignInRedirect(): (returnTo?: string) => Promise<void> {
  const router = useRouter()
  const route = useRoute()

  return async (returnTo?: string) => {
    await router.push({ name: 'login', query: { redirect: returnTo ?? route.fullPath } })
  }
}
