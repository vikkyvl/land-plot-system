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
    <RouterLink
      class="app-header__brand"
      to="/"
    >
      Arqetra
    </RouterLink>
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
        class="app-header__action"
        @click="session.signOut()"
      >
        Вийти
      </button>
    </div>
    <RouterLink
      v-else-if="route.name !== 'login'"
      class="app-header__action"
      :to="loginTarget"
    >
      Увійти
    </RouterLink>
  </header>
</template>

<style scoped>
.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: calc(var(--space-4) + env(safe-area-inset-top)) calc(var(--space-5) + env(safe-area-inset-right))
    var(--space-3) calc(var(--space-5) + env(safe-area-inset-left));
}

.app-header__brand {
  color: var(--color-ink);
  font-size: var(--fs-lg);
  font-weight: 800;
  text-decoration: none;
}

.app-header__account {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.app-header__user {
  overflow: hidden;
  color: var(--color-ink-3);
  font-size: var(--fs-sm);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-header__action {
  display: inline-flex;
  align-items: center;
  min-height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: 700;
  text-decoration: none;
  cursor: pointer;
}
</style>
