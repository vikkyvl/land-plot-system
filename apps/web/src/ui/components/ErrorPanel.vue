<script setup lang="ts">
import { computed } from 'vue'

import type { Problem } from '../../domain/problem'
import { problemText, swapText } from '../composables/errorText'

const props = defineProps<{ problem: Problem }>()
const emit = defineEmits<{ swap: [first: number, second: number]; dismiss: [] }>()

const text = computed(() => problemText(props.problem))
const swap = computed(() => (props.problem.details?.kind === 'self_intersection' ? props.problem.details : null))
</script>

<template>
  <section
    class="error-panel"
    role="alert"
  >
    <div class="error-panel__heading">
      <svg
        width="22"
        height="22"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <circle
          cx="12"
          cy="12"
          r="9.5"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        />
        <path
          d="M12 7v6.5"
          stroke="currentColor"
          stroke-width="2.2"
          stroke-linecap="round"
        />
        <circle
          cx="12"
          cy="17"
          r="1.3"
          fill="currentColor"
        />
      </svg>
      <h2 class="error-panel__title">
        {{ text.title }}
      </h2>
    </div>
    <p class="error-panel__text">
      {{ text.text }}
    </p>
    <p class="error-panel__code">
      Код помилки: {{ problem.code }}
    </p>
    <div class="error-panel__actions">
      <button
        v-if="swap"
        type="button"
        class="error-panel__button error-panel__button--primary"
        @click="emit('swap', swap.swap[0], swap.swap[1])"
      >
        {{ swapText(swap) }}
      </button>
      <button
        type="button"
        class="error-panel__button"
        @click="emit('dismiss')"
      >
        Виправлю сам
      </button>
    </div>
  </section>
</template>

<style scoped>
.error-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: 22px;
  border: 1.5px solid var(--color-danger-border);
  border-radius: var(--radius-2xl);
  background: var(--color-danger-bg);
}

.error-panel__heading {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-danger-ink);
}

.error-panel__heading svg {
  flex: none;
}

.error-panel__title {
  margin: 0;
  font-size: var(--fs-panel-title);
  font-weight: 800;
}

.error-panel__text {
  margin: 0;
  color: var(--color-danger-ink-2);
  font-size: var(--fs-base);
  line-height: 1.5;
}

.error-panel__code {
  margin: 0;
  color: var(--color-danger-ink);
  font-size: var(--fs-sm);
}

.error-panel__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.error-panel__button {
  flex: 1 1 120px;
  min-height: var(--control-h);
  padding: var(--space-2) var(--space-3);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
  font-weight: 600;
  cursor: pointer;
}

.error-panel__button--primary {
  flex-basis: 160px;
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-weight: 700;
}
</style>
