<script setup lang="ts">
import { computed } from 'vue'

import { ErrorCode } from '../../domain/errorCode'
import type { Problem } from '../../domain/problem'

const props = defineProps<{ problems: readonly Problem[] }>()

type CheckState = 'passed' | 'failed' | 'unknown'

const pointCodes: readonly string[] = [
  ErrorCode.InsufficientPoints,
  ErrorCode.DuplicatePoint,
  ErrorCode.InvalidCoordinate,
  ErrorCode.TooManyPoints,
]

const checks = computed(() => {
  const codes = new Set(props.problems.map((problem) => problem.code))
  const failed = (...list: string[]) => list.some((code) => codes.has(code))
  const pointsBroken = pointCodes.some((code) => codes.has(code))
  const state = (broken: boolean, known = true): CheckState => (broken ? 'failed' : known ? 'passed' : 'unknown')

  const rows: { label: string; state: CheckState }[] = [
    { label: 'Щонайменше 3 різні точки', state: state(failed(ErrorCode.InsufficientPoints, ErrorCode.DuplicatePoint)) },
    { label: 'Координати в допустимих межах', state: state(failed(ErrorCode.InvalidCoordinate)) },
    { label: 'Без самоперетину', state: state(failed(ErrorCode.SelfIntersection), !pointsBroken) },
  ]
  if (codes.has(ErrorCode.TooManyPoints)) {
    rows.push({ label: 'Не більше 1000 точок', state: 'failed' })
  }
  if (codes.has(ErrorCode.ZeroArea)) {
    rows.push({ label: 'Площа більша за нуль', state: 'failed' })
  }

  return rows
})

const marks: Readonly<Record<CheckState, string>> = { passed: '✓', failed: '✕', unknown: '–' }
const hints: Readonly<Record<CheckState, string>> = {
  passed: 'виконано',
  failed: 'не виконано',
  unknown: 'не перевірено',
}
</script>

<template>
  <section class="checks-card">
    <h2 class="checks-card__title">
      Інші перевірки
    </h2>
    <ul class="checks-card__list">
      <li
        v-for="check in checks"
        :key="check.label"
        :class="`checks-card__item checks-card__item--${check.state}`"
      >
        <span aria-hidden="true">{{ marks[check.state] }}</span>
        {{ check.label }}
        <span class="visually-hidden">: {{ hints[check.state] }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.checks-card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: 18px var(--space-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
}

.checks-card__title {
  margin: 0;
  font-size: var(--fs-base);
  font-weight: 700;
}

.checks-card__list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  color: var(--color-ink-2);
  font-size: var(--fs-control);
  list-style: none;
}

.checks-card__item--failed {
  color: var(--color-danger);
  font-weight: 700;
}

.checks-card__item--unknown {
  color: var(--color-muted);
}
</style>
