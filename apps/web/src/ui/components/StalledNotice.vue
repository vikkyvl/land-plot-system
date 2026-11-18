<script setup lang="ts">
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div
    class="stalled"
    role="status"
  >
    <span>Обробка триває довше, ніж зазвичай. Сервер ще не підтвердив ділянку.</span>
    <button
      type="button"
      class="stalled__button"
      @click="emit('retry')"
    >
      Перевірити ще раз
    </button>
  </div>
</template>

<style scoped>
.stalled {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-3);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-md);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
  font-size: var(--fs-control);
}

.stalled span {
  flex: 1 1 220px;
}

.stalled__button {
  min-height: var(--control-h-sm);
  padding: 0 14px;
  border: 1.5px solid var(--color-warning-border-strong);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-warning-ink);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

@media (pointer: coarse) {
  .stalled__button {
    min-height: var(--touch-min);
  }
}
</style>
