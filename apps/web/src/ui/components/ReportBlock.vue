<script setup lang="ts">
defineProps<{
  title: string
  state: 'ready' | 'no-data' | 'unavailable'
  canRefresh: boolean
  refreshing: boolean
}>()

const emit = defineEmits<{ refresh: [] }>()
const titleId = `report-block-${Math.random().toString(36).slice(2)}`
</script>

<template>
  <section
    class="report-block"
    :aria-labelledby="titleId"
  >
    <h2
      :id="titleId"
      class="report-block__title"
    >
      {{ title }}
    </h2>
    <p
      v-if="state === 'no-data'"
      class="report-block__muted"
    >
      {{ $t('surroundings.noData') }}
    </p>
    <div
      v-else-if="state === 'unavailable'"
      class="report-block__unavailable"
      role="status"
    >
      <p>{{ $t('surroundings.unavailable') }}</p>
      <button
        v-if="canRefresh"
        type="button"
        class="report-block__refresh ui-secondary"
        :disabled="refreshing"
        @click="emit('refresh')"
      >
        {{ refreshing ? $t('report.refreshing') : $t('report.refresh') }}
      </button>
    </div>
    <slot v-else />
  </section>
</template>

<style scoped>
.report-block {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-5);
  border-top: 1px solid var(--color-border);
  animation: motion-fade-up var(--motion-base) var(--ease-out) backwards;
  animation-delay: calc(var(--order, 0) * 2 * var(--motion-stagger) + var(--motion-fast));
}

.report-block__title {
  margin: 0;
  font-size: var(--fs-panel-title);
  font-weight: 800;
}

.report-block__muted {
  margin: 0;
  color: var(--color-ink-3);
  font-size: var(--fs-control);
}

.report-block__unavailable {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-xs);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
  font-size: var(--fs-control);
}

.report-block__unavailable p {
  flex: 1 1 220px;
  margin: 0;
}

.report-block__refresh {
  min-height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-primary);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
</style>
