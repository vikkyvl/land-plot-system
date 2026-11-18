<script setup lang="ts">
import { computed } from 'vue'

import type { Coordinate } from '../../domain/coordinate'

const props = defineProps<{ points: readonly Coordinate[]; size?: number }>()

const box = 64
const padding = 12

const outline = computed(() => {
  if (props.points.length < 3) {
    return ''
  }
  const middle = props.points.reduce((sum, point) => sum + point.latitude, 0) / props.points.length
  const stretch = Math.cos((middle * Math.PI) / 180)
  const flat = props.points.map((point) => ({ x: point.longitude * stretch, y: -point.latitude }))
  const xs = flat.map((point) => point.x)
  const ys = flat.map((point) => point.y)
  const width = Math.max(...xs) - Math.min(...xs)
  const height = Math.max(...ys) - Math.min(...ys)
  const scale = (box - 2 * padding) / Math.max(width, height, Number.EPSILON)
  const left = (box - width * scale) / 2
  const top = (box - height * scale) / 2

  return flat
    .map((point) => `${(left + (point.x - Math.min(...xs)) * scale).toFixed(1)},${(top + (point.y - Math.min(...ys)) * scale).toFixed(1)}`)
    .join(' ')
})
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
      v-if="outline"
      class="plot-thumbnail__shape"
      :points="outline"
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
