<script setup lang="ts">
import type { MapLayerId } from '../../domain/mapLayer'

defineProps<{ modelValue: MapLayerId }>()
const emit = defineEmits<{ 'update:modelValue': [layer: MapLayerId] }>()

const options: readonly { id: MapLayerId; label: string }[] = [
  { id: 'map', label: 'Карта' },
  { id: 'satellite', label: 'Супутник' },
]
</script>

<template>
  <div
    class="layer-switch"
    role="group"
    aria-label="Шар карти"
  >
    <button
      v-for="option in options"
      :key="option.id"
      type="button"
      class="layer-switch__option"
      :aria-pressed="modelValue === option.id"
      @click="emit('update:modelValue', option.id)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.layer-switch {
  display: flex;
  padding: var(--space-1);
  border-radius: var(--radius-md);
  background: var(--color-segmented-bg);
}

.layer-switch__option {
  height: var(--control-h-segment);
  padding: 0 14px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-ink-3);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  cursor: pointer;
}

.layer-switch__option[aria-pressed='true'] {
  background: var(--color-surface);
  color: var(--color-ink);
  font-weight: 700;
}
</style>
