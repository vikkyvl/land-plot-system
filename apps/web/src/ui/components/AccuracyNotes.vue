<script setup lang="ts">
import { computed } from 'vue'

import { scaleDominates, type Accuracy } from '../../domain/accuracy'
import type { Obstruction } from '../../domain/survey'
import { groundShift, mainSource } from '../composables/accuracyText'

const props = defineProps<{
  accuracy: Accuracy
  obstructions: readonly Obstruction[]
  pointCount: number
  zoomHint?: boolean
}>()

const assumed = computed(() => props.accuracy.basis === 'assumed')
const showZoomHint = computed(() => props.zoomHint === true && !assumed.value && scaleDominates(props.accuracy))
</script>

<template>
  <div class="accuracy-notes">
    <p
      v-if="assumed"
      class="accuracy-notes__assumed"
    >
      {{ $t('accuracy.approximateNote') }}
    </p>
    <p class="accuracy-notes__source">
      {{ mainSource(accuracy, obstructions, pointCount) }}
    </p>
    <p
      v-if="assumed"
      class="accuracy-notes__hint"
    >
      {{ $t('accuracy.refineHint') }}
    </p>
    <p
      v-else-if="showZoomHint"
      class="accuracy-notes__hint"
    >
      {{ $t('accuracy.zoomHint') }}
    </p>
    <p class="accuracy-notes__shift">
      {{ groundShift(accuracy) }}
    </p>
  </div>
</template>

<style scoped>
.accuracy-notes {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin-top: var(--space-2);
}

.accuracy-notes p {
  margin: 0;
}

.accuracy-notes__source {
  color: var(--color-ink-2);
  font-size: var(--fs-sm);
}

.accuracy-notes__assumed {
  color: var(--color-ink-3);
  font-size: var(--fs-sm);
}

.accuracy-notes__hint {
  color: var(--color-primary);
  font-size: var(--fs-sm);
  font-weight: 600;
}

.accuracy-notes__shift {
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
}
</style>
