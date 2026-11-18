<script setup lang="ts">
import { ref } from 'vue'
import { useRoute } from 'vue-router'

import type { SignInMethod } from '../../domain/session'
import { useSessionStore } from '../stores/session'

const session = useSessionStore()
const route = useRoute()
const pending = ref<SignInMethod | null>(null)
const failed = ref(false)

async function signIn(method: SignInMethod): Promise<void> {
  pending.value = method
  failed.value = false
  const redirect = route.query['redirect']

  try {
    await session.signIn(method, typeof redirect === 'string' ? redirect : null)
  } catch {
    failed.value = true
    pending.value = null
  }
}
</script>

<template>
  <main class="login">
    <section
      class="login__card"
      aria-labelledby="login-title"
    >
      <div class="login__intro">
        <p class="login__eyebrow">
          {{ $t('login.eyebrow') }}
        </p>
        <h1
          id="login-title"
          class="login__title"
        >
          {{ $t('login.title') }}
        </h1>
        <p class="login__text">
          {{ $t('login.text') }}
        </p>
      </div>

      <div class="login__actions">
        <button
          type="button"
          class="login__button login__button--primary"
          :disabled="pending !== null"
          @click="signIn('password')"
        >
          {{ $t('login.signIn') }}
        </button>
        <button
          type="button"
          class="login__button"
          :disabled="pending !== null"
          @click="signIn('registration')"
        >
          {{ $t('login.register') }}
        </button>

        <div
          class="login__divider"
          aria-hidden="true"
        >
          {{ $t('login.or') }}
        </div>

        <button
          type="button"
          class="login__button"
          :disabled="pending !== null"
          @click="signIn('google')"
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 20 20"
            aria-hidden="true"
          >
            <circle
              cx="10"
              cy="10"
              r="8"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
            />
            <path
              d="M10 10 H16"
              stroke="currentColor"
              stroke-width="1.8"
            />
          </svg>
          {{ $t('login.google') }}
        </button>
      </div>

      <p
        v-if="failed"
        class="login__error"
        role="alert"
      >
        {{ $t('login.unavailable') }}
      </p>

      <p class="login__note">
        {{ $t('login.guestNote') }}
      </p>
    </section>
  </main>
</template>

<style scoped>
.login {
  display: flex;
  justify-content: center;
  padding: var(--space-3) calc(var(--space-6) + env(safe-area-inset-right)) calc(var(--space-7) + env(safe-area-inset-bottom))
    calc(var(--space-6) + env(safe-area-inset-left));
}

.login__card {
  display: flex;
  flex-direction: column;
  gap: var(--space-7);
  width: 100%;
  max-width: 390px;
}

.login__intro {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.login__eyebrow {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: 700;
}

.login__title {
  margin: 0;
  font-size: var(--fs-h1);
  font-weight: 800;
  letter-spacing: -0.02em;
  line-height: 1.15;
}

.login__text,
.login__note {
  margin: 0;
  color: var(--color-muted);
  line-height: 1.5;
}

.login__text {
  font-size: var(--fs-base);
}

.login__actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.login__button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: var(--control-h-lg);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
  font-weight: 700;
  cursor: pointer;
}

.login__button--primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: var(--fs-md);
}

.login__button--primary:hover:not(:disabled) {
  border-color: var(--color-primary-hover);
  background: var(--color-primary-hover);
}

.login__button:disabled {
  cursor: progress;
  opacity: 0.7;
}

.login__divider {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--color-muted);
  font-size: var(--fs-sm);
}

.login__divider::before,
.login__divider::after {
  flex: 1;
  height: 1px;
  background: var(--color-divider);
  content: '';
}

.login__error {
  margin: 0;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-danger-border);
  border-radius: var(--radius-md);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink);
  font-size: var(--fs-sm);
}

.login__note {
  font-size: var(--fs-sm);
  text-align: center;
}
</style>
