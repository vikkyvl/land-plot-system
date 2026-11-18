<script setup lang="ts">
import type { Accuracy } from '../../domain/accuracy'
import type { Obstruction } from '../../domain/survey'
import { formatMargin } from '../composables/accuracyText'
import { formatPercent } from '../composables/surroundingsText'

import AccuracyNotes from './AccuracyNotes.vue'

defineProps<{ accuracy: Accuracy; obstructions: readonly Obstruction[]; pointCount: number }>()

const percentOfWhole = 100
</script>

<template>
  <div class="report-accuracy">
    <p class="report-accuracy__margin">
      {{ $t('surroundings.marginOf', { margin: formatMargin(accuracy), percent: formatPercent(accuracy.percent) }) }}
      <span class="report-accuracy__confidence">{{
        $t('surroundings.confidence', { value: formatPercent(accuracy.confidence * percentOfWhole) })
      }}</span>
    </p>
    <dl class="report-accuracy__shares">
      <div
        v-for="share in [
          { label: $t('surroundings.shareScale'), value: accuracy.scaleShare },
          { label: $t('surroundings.shareObstructions'), value: accuracy.obstructionShare },
        ]"
        :key="share.label"
        class="report-accuracy__share"
      >
        <dt>{{ share.label }}</dt>
        <dd>
          <span
            class="report-accuracy__bar"
            :style="{ '--share': share.value }"
            aria-hidden="true"
          />
          {{ formatPercent(share.value * percentOfWhole) }}
        </dd>
      </div>
    </dl>
    <AccuracyNotes
      :accuracy="accuracy"
      :obstructions="obstructions"
      :point-count="pointCount"
    />
  </div>
</template>

<style scoped>
.report-accuracy {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.report-accuracy__margin {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-2);
  margin: 0;
  font-size: var(--fs-title-md);
  font-weight: 800;
}

.report-accuracy__confidence {
  color: var(--color-ink-3);
  font-size: var(--fs-control);
  font-weight: 600;
}

.report-accuracy__shares {
  display: grid;
  gap: var(--space-2);
  margin: 0;
}

.report-accuracy__share {
  display: grid;
  grid-template-columns: minmax(120px, 180px) 1fr;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--fs-control);
}

.report-accuracy__share dt {
  color: var(--color-ink-2);
}

.report-accuracy__share dd {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.report-accuracy__bar {
  flex: 1;
  height: 8px;
  border-radius: 4px;
  background: linear-gradient(
    to right,
    var(--color-primary) calc(var(--share) * 100%),
    var(--color-primary-soft) calc(var(--share) * 100%)
  );
}
</style>
