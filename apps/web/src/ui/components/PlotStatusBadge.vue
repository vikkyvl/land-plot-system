<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'

import { PlotStatus } from '../../domain/plot'

const props = defineProps<{ status: PlotStatus }>()

const confirmedShownFor = 2400
const justConfirmed = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

watch(
  () => props.status,
  (status, previous) => {
    if (previous === PlotStatus.Pending && status === PlotStatus.Confirmed) {
      justConfirmed.value = true
      if (timer !== null) {
        clearTimeout(timer)
      }
      timer = setTimeout(() => {
        justConfirmed.value = false
      }, confirmedShownFor)
    }
  },
)

onBeforeUnmount(() => {
  if (timer !== null) {
    clearTimeout(timer)
  }
})
</script>

<template>
  <Transition
    name="status-badge"
    mode="out-in"
  >
    <span
      v-if="status === PlotStatus.Pending"
      key="pending"
      class="status-badge motion-shimmer"
    >
      <span
        class="status-badge__spinner"
        aria-hidden="true"
      />
      {{ $t('plots.processing') }}
    </span>
    <span
      v-else-if="justConfirmed"
      key="confirmed"
      class="status-badge status-badge--confirmed"
    >
      <svg
        class="status-badge__check"
        width="12"
        height="12"
        viewBox="0 0 12 12"
        aria-hidden="true"
      >
        <path
          d="M2 6.5 L5 9.2 L10 3"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          pathLength="1"
        />
      </svg>
      {{ $t('plots.confirmed') }}
    </span>
  </Transition>
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

.status-badge--confirmed {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  animation: motion-pop var(--motion-base) var(--ease-spring);
}

.status-badge__check path {
  stroke-dasharray: 1;
  animation: motion-draw var(--motion-base) var(--ease-out) var(--motion-fast) both;
}

.status-badge__spinner {
  width: 10px;
  height: 10px;
  border: 2px solid var(--color-border-strong);
  border-top-color: var(--color-ink-3);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.status-badge-leave-active {
  transition: opacity var(--motion-fast) var(--ease-out);
}

.status-badge-leave-to {
  opacity: 0;
}

@keyframes spin {
  to {
    transform: rotate(1turn);
  }
}
</style>
