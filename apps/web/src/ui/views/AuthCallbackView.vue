<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import { useSessionService } from '../composables/services'
import { useSessionStore } from '../stores/session'

const service = useSessionService()
const session = useSessionStore()
const router = useRouter()
const failed = ref(false)

onMounted(async () => {
  try {
    const returnTo = await service.completeSignIn()
    await session.refresh()
    await router.replace(returnTo)
  } catch {
    failed.value = true
  }
})
</script>

<template>
  <main class="callback">
    <div
      v-if="!failed"
      class="callback__progress"
      role="status"
    >
      <span
        class="callback__spinner"
        aria-hidden="true"
      />
      Завершуємо вхід…
    </div>
    <div
      v-else
      class="callback__error"
      role="alert"
    >
      <p>Не вдалося завершити вхід.</p>
      <RouterLink to="/login">
        Спробувати ще раз
      </RouterLink>
    </div>
  </main>
</template>

<style scoped>
.callback {
  display: grid;
  min-height: 60dvh;
  place-items: center;
  padding: var(--space-6);
}

.callback__progress {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--color-ink-3);
  font-weight: 600;
}

.callback__spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--color-primary-soft);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.callback__error {
  color: var(--color-danger-ink);
  text-align: center;
}

@keyframes spin {
  to {
    transform: rotate(1turn);
  }
}

@media (prefers-reduced-motion: reduce) {
  .callback__spinner {
    animation-duration: 2.4s;
  }
}
</style>
