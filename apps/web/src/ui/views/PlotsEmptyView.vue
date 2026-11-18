<template>
  <div class="plots-empty">
    <p>Оберіть ділянку зі списку, щоб побачити карту, площу та історію вимірювань.</p>
  </div>
</template>

<style scoped>
.plots-empty {
  display: grid;
  min-height: 320px;
  place-items: center;
  padding: var(--space-6);
  border: 1px dashed var(--color-border-strong);
  border-radius: var(--radius-2xl);
  color: var(--color-muted);
  text-align: center;
}

.plots-empty p {
  max-width: 360px;
  margin: 0;
}
</style>
