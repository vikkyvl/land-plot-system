<script setup lang="ts">
import { storeToRefs } from 'pinia'

import { useToastStore } from '../stores/toast'

const store = useToastStore()
const { toasts } = storeToRefs(store)
</script>

<template>
  <div
    class="toast-host"
    role="status"
    aria-live="polite"
  >
    <button
      v-for="toast in toasts"
      :key="toast.id"
      type="button"
      class="toast-host__toast"
      :class="`toast-host__toast--${toast.tone}`"
      @click="store.dismiss(toast.id)"
    >
      {{ toast.text }}
    </button>
  </div>
</template>

<style scoped>
.toast-host {
  position: fixed;
  z-index: 3000;
  bottom: calc(var(--space-6) + var(--bottom-nav-height, 0px) + env(safe-area-inset-bottom));
  left: 50%;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: min(440px, calc(100% - 2 * var(--space-4)));
  transform: translateX(-50%);
  pointer-events: none;
}

.toast-host__toast {
  padding: var(--space-3) var(--space-4);
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-chip-dark);
  color: var(--color-on-primary);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  text-align: left;
  pointer-events: auto;
  cursor: pointer;
}

.toast-host__toast--error {
  background: var(--color-danger);
}
</style>
