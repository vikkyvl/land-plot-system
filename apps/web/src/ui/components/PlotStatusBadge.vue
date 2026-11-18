<script setup lang="ts">
import { PlotStatus } from '../../domain/plot'

defineProps<{ status: PlotStatus }>()
</script>

<template>
  <span
    v-if="status === PlotStatus.Pending"
    class="status-badge"
  >
    <span
      class="status-badge__spinner"
      aria-hidden="true"
    />
    {{ $t('plots.processing') }}
  </span>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-segmented-bg);
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
  font-weight: 600;
  white-space: nowrap;
}

.status-badge__spinner {
  width: 10px;
  height: 10px;
  border: 2px solid var(--color-border-strong);
  border-top-color: var(--color-ink-3);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(1turn);
  }
}

@media (prefers-reduced-motion: reduce) {
  .status-badge__spinner {
    animation-duration: 2.4s;
  }
}
</style>
