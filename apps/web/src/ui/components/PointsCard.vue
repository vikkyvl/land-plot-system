<script setup lang="ts">
import type { Coordinate } from '../../domain/coordinate'
import { formatCoordinate } from '../composables/format'

defineProps<{ points: readonly Coordinate[]; flagged: readonly number[] }>()
</script>

<template>
  <section class="points-card">
    <h2 class="points-card__title">
      Точки межі · {{ points.length }}
    </h2>
    <ol class="points-card__list">
      <li
        v-for="(point, index) in points"
        :key="index"
        :class="{ 'points-card__item--flagged': flagged.includes(index) }"
      >
        {{ formatCoordinate(point) }}
      </li>
    </ol>
  </section>
</template>

<style scoped>
.points-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--space-5) var(--space-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
}

.points-card__title {
  margin: 0;
  font-size: var(--fs-base);
  font-weight: 700;
}

.points-card__list {
  max-height: 320px;
  margin: 0;
  padding-left: var(--space-5);
  overflow-y: auto;
  color: var(--color-ink-2);
  font-size: var(--fs-control);
  font-variant-numeric: tabular-nums;
  line-height: 1.8;
}

.points-card__item--flagged {
  color: var(--color-danger);
  font-weight: 700;
}
</style>
