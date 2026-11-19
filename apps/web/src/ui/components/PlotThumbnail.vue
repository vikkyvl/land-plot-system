<script setup lang="ts">
import { computed } from 'vue'

import type { OutlinePoint } from '../../domain/plot'

const props = defineProps<{ outline: readonly OutlinePoint[]; size?: number }>()

const box = 64
const padding = 12
const minimumPoints = 3

const shape = computed(() =>
  props.outline.length < minimumPoints
    ? ''
    : props.outline
        .map((point) => `${(padding + point.x * (box - 2 * padding)).toFixed(1)},${(padding + point.y * (box - 2 * padding)).toFixed(1)}`)
        .join(' '),
)
</script>

<template>
  <svg
    class="plot-thumbnail"
    :width="size ?? 60"
    :height="size ?? 60"
    viewBox="0 0 64 64"
    aria-hidden="true"
  >
    <rect
      class="plot-thumbnail__ground"
      width="64"
      height="64"
      rx="10"
    />
    <polygon
      v-if="shape"
      class="plot-thumbnail__shape"
      :points="shape"
    />
  </svg>
</template>

<style scoped>
.plot-thumbnail {
  flex: none;
}

.plot-thumbnail__ground {
  fill: var(--color-map-bg);
}

.plot-thumbnail__shape {
  fill: var(--color-plot-fill);
  stroke: var(--color-plot-stroke);
  stroke-width: 2.5;
  stroke-linejoin: round;
}
</style>
