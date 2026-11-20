<script setup lang="ts">
import { useOnline } from '../composables/useOnline'

const online = useOnline()
</script>

<template>
  <div
    v-if="!online"
    class="offline-banner"
    role="status"
  >
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      aria-hidden="true"
    >
      <path
        d="M3 9 a14 14 0 0 1 18 0 M6.5 12.5 a9 9 0 0 1 11 0 M10 16 a4 4 0 0 1 4 0 M4 4 L20 20"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
      />
    </svg>
    <span><b>{{ $t('app.offline') }}</b> {{ $t('app.offlineDetails') }}</span>
  </div>
</template>

<style scoped>
.offline-banner {
  position: sticky;
  z-index: 2500;
  top: 0;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: calc(var(--space-2) + env(safe-area-inset-top)) calc(var(--space-4) + env(safe-area-inset-right))
    var(--space-2) calc(var(--space-4) + env(safe-area-inset-left));
  border-bottom: 1px solid var(--color-warning-border);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
  font-size: var(--fs-control);
}

.offline-banner svg {
  flex: none;
  color: var(--color-warning-icon);
}
</style>
