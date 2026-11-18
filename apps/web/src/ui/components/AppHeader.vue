<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import { useSessionStore } from '../stores/session'

const session = useSessionStore()
const { signedIn, userName } = storeToRefs(session)
const route = useRoute()
const loginTarget = computed(() => ({ name: 'login', query: { redirect: route.fullPath } }))
</script>

<template>
  <header class="app-header">
    <div class="app-header__inner">
      <RouterLink
        class="app-header__brand"
        to="/"
      >
        <svg
          width="28"
          height="28"
          viewBox="0 0 28 28"
          aria-hidden="true"
        >
          <path
            d="M5 20 L10 6 L23 9 L21 22 Z"
            fill="none"
            stroke="currentColor"
            stroke-width="2.4"
            stroke-linejoin="round"
          />
          <circle
            cx="10"
            cy="6"
            r="2.6"
            fill="currentColor"
          />
        </svg>
        <span>Arqetra</span>
      </RouterLink>
      <nav
        class="app-header__nav"
        aria-label="Розділи"
      >
        <RouterLink
          class="app-header__tab"
          to="/"
        >
          Розрахунок
        </RouterLink>
      </nav>
      <div
        v-if="signedIn"
        class="app-header__account"
      >
        <span
          v-if="userName"
          class="app-header__user"
        >{{ userName }}</span>
        <button
          type="button"
          class="app-header__button"
          @click="session.signOut()"
        >
          Вийти
        </button>
      </div>
      <RouterLink
        v-else-if="route.name !== 'login'"
        class="app-header__button app-header__button--primary"
        :to="loginTarget"
      >
        Увійти
      </RouterLink>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.app-header__inner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3) var(--space-7);
  max-width: var(--container-max);
  margin: 0 auto;
  padding: 14px calc(var(--space-6) + env(safe-area-inset-right)) 14px calc(var(--space-6) + env(safe-area-inset-left));
}

.app-header__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-primary);
  text-decoration: none;
}

.app-header__brand span {
  color: var(--color-ink);
  font-size: var(--fs-h3);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.app-header__nav {
  display: flex;
  flex: 1;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.app-header__tab {
  padding: 10px 14px;
  border-radius: var(--radius-xs);
  color: var(--color-ink-2);
  font-weight: 600;
  text-decoration: none;
}

.app-header__tab.router-link-exact-active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  font-weight: 700;
}

.app-header__account {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: var(--color-ink-2);
  font-size: var(--fs-control);
}

.app-header__user {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-header__button {
  display: inline-flex;
  align-items: center;
  height: var(--control-h-sm);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
}

.app-header__button--primary {
  padding: 0 var(--space-4);
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-weight: 700;
}

@media (max-width: 767.98px) {
  .app-header {
    border-bottom: none;
    background: var(--color-bg);
  }

  .app-header__inner {
    justify-content: space-between;
    padding: calc(18px + env(safe-area-inset-top)) calc(var(--space-5) + env(safe-area-inset-right)) 14px
      calc(var(--space-5) + env(safe-area-inset-left));
  }

  .app-header__brand span {
    font-size: var(--fs-h3);
  }

  .app-header__nav,
  .app-header__user {
    display: none;
  }

  .app-header__button,
  .app-header__button--primary {
    height: var(--touch-min);
    padding: 0 var(--space-1);
    border: none;
    background: transparent;
    color: var(--color-primary);
    font-size: var(--fs-base);
    font-weight: 700;
  }
}
</style>
