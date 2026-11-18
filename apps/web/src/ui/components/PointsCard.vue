<script setup lang="ts">
import type { Coordinate } from '../../domain/coordinate'
import { obstructionOn, obstructionReasons, type Obstruction, type ObstructionReason } from '../../domain/survey'
import { formatCoordinate } from '../composables/format'

const props = withDefaults(
  defineProps<{ points: readonly Coordinate[]; flagged: readonly number[]; obstructions?: readonly Obstruction[] }>(),
  { obstructions: () => [] },
)

const emit = defineEmits<{ mark: [side: number, reason: ObstructionReason | null] }>()

const minimumSidePoints = 3

function sideTo(index: number): number {
  return ((index + 1) % props.points.length) + 1
}

function choose(side: number, value: string): void {
  emit('mark', side, value === '' ? null : (value as ObstructionReason))
}
</script>

<template>
  <section class="points-card">
    <h2 class="points-card__title">
      {{ $t('problems.boundaryPoints', { n: points.length }) }}
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
    <template v-if="points.length >= minimumSidePoints">
      <h3 class="points-card__title">
        {{ $t('sides.title') }}
      </h3>
      <ul class="points-card__sides">
        <li
          v-for="(_, index) in points"
          :key="index"
          class="points-card__side"
        >
          <label :for="`side-${index}`">{{ $t('sides.side', { from: index + 1, to: sideTo(index) }) }}</label>
          <select
            :id="`side-${index}`"
            class="points-card__select"
            :aria-label="$t('sides.coveredBySide', { from: index + 1, to: sideTo(index) })"
            :value="obstructionOn({ obstructions }, index) ?? ''"
            @change="choose(index, ($event.target as HTMLSelectElement).value)"
          >
            <option value="">
              {{ $t('sides.none') }}
            </option>
            <option
              v-for="reason in obstructionReasons"
              :key="reason"
              :value="reason"
            >
              {{ $t(`sides.${reason}`) }}
            </option>
          </select>
        </li>
      </ul>
    </template>
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

.points-card__sides {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 320px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.points-card__side {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  color: var(--color-ink-2);
  font-size: var(--fs-control);
}

.points-card__select {
  min-height: 36px;
  padding: 0 var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
}
</style>
