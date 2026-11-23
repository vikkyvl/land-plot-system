<script setup lang="ts">
import type { PreviewOutage } from '../stores/calculator'

defineProps<{ outage: PreviewOutage }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <section
    class="outage-panel"
    role="alert"
  >
    <h2 class="outage-panel__title">
      {{ $t('outage.title') }}
    </h2>
    <p
      v-if="outage.rateLimited"
      class="outage-panel__text"
    >
      {{ $t('outage.rateLimited') }}
      <template v-if="outage.secondsLeft !== null">
        {{ $t('outage.retryIn', { seconds: outage.secondsLeft }) }}
      </template>
    </p>
    <p
      v-else
      class="outage-panel__text"
    >
      {{ $t('outage.noAnswer') }}
    </p>
    <button
      v-if="!outage.rateLimited"
      type="button"
      class="outage-panel__button ui-secondary"
      @click="emit('retry')"
    >
      {{ $t('app.retry') }}
    </button>
  </section>
</template>

<style scoped>
.outage-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-5);
  border: 1.5px solid var(--color-warning-border);
  border-radius: var(--radius-2xl);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
}

.outage-panel__title {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: 800;
}

.outage-panel__text {
  margin: 0;
  font-size: var(--fs-control);
  line-height: 1.5;
}

.outage-panel__button {
  align-self: flex-start;
  min-height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-warning-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 700;
  cursor: pointer;
}
</style>
