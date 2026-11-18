<script setup lang="ts">
import { ref } from 'vue'

import { useFocusTrap } from '../composables/useFocusTrap'

defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()

const dialog = ref<HTMLElement | null>(null)
const titleId = `dialog-${Math.random().toString(36).slice(2)}`
const onKeydown = useFocusTrap(dialog, () => emit('close'))
</script>

<template>
  <Teleport to="body">
    <div
      class="dialog-overlay"
      @mousedown.self="emit('close')"
    >
      <div
        ref="dialog"
        class="dialog"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        @keydown="onKeydown"
      >
        <h2
          :id="titleId"
          class="dialog__title"
        >
          {{ title }}
        </h2>
        <slot />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  z-index: 2000;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  background: var(--overlay);
}

.dialog {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  width: 100%;
  max-width: 460px;
  padding: var(--space-7);
  border-radius: var(--radius-dialog);
  background: var(--color-surface);
  box-shadow: var(--shadow-dialog);
}

.dialog__title {
  margin: 0;
  font-size: var(--fs-h3);
  font-weight: 800;
  overflow-wrap: anywhere;
}
</style>
