<script setup lang="ts">
import { locales, type Locale } from '../../domain/locale'
import { useLocale } from '../composables/useLocale'

const { locale, choose } = useLocale()

const labels: Readonly<Record<Locale, string>> = { uk: 'УК', en: 'EN' }
const names: Readonly<Record<Locale, string>> = { uk: 'Українська', en: 'English' }
</script>

<template>
  <div
    class="locale-switch"
    role="group"
    :aria-label="$t('app.language')"
  >
    <button
      v-for="option in locales"
      :key="option"
      type="button"
      class="locale-switch__option ui-control"
      :lang="option"
      :aria-pressed="locale === option"
      :aria-label="names[option]"
      @click="choose(option)"
    >
      {{ labels[option] }}
    </button>
  </div>
</template>

<style scoped>
.locale-switch {
  display: inline-flex;
  padding: 2px;
  border-radius: var(--radius-sm);
  background: var(--color-segmented-bg);
}

.locale-switch__option {
  min-width: 40px;
  height: 32px;
  padding: 0 var(--space-2);
  border: none;
  border-radius: calc(var(--radius-sm) - 2px);
  background: transparent;
  color: var(--color-ink-3);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: 600;
  cursor: pointer;
}

.locale-switch__option[aria-pressed='true'] {
  background: var(--color-surface);
  color: var(--color-ink);
  font-weight: 800;
}

@media (pointer: coarse) {
  .locale-switch__option {
    height: var(--touch-min);
    min-width: var(--touch-min);
  }
}
</style>
