<script setup lang="ts">
import { computed, ref } from 'vue'

import { formatVersionCount } from '../composables/format'
import type { Failure } from '../composables/useFailure'

import BaseDialog from './BaseDialog.vue'

const props = defineProps<{ name: string; versionCount: number | null; busy: boolean; failure: Failure | null }>()
const emit = defineEmits<{ confirm: []; close: [] }>()

const typed = ref('')
const inputId = `confirm-name-${Math.random().toString(36).slice(2)}`
const matches = computed(() => typed.value.trim() === props.name.trim())
const history = computed(() =>
  props.versionCount === null ? 'історію вимірювань' : `історію вимірювань (${formatVersionCount(props.versionCount)})`,
)
</script>

<template>
  <BaseDialog
    :title="`Видалити «${name}»?`"
    @close="emit('close')"
  >
    <p class="delete-dialog__text">
      Ділянку і {{ history }} буде видалено. Звіти по ній залишаться з позначкою «Ділянку видалено». Цю дію не можна
      скасувати.
    </p>
    <form
      class="delete-dialog__form"
      @submit.prevent="matches && !busy && emit('confirm')"
    >
      <div class="delete-dialog__field">
        <label
          class="delete-dialog__label"
          :for="inputId"
        >Введіть назву ділянки для підтвердження</label>
        <input
          :id="inputId"
          v-model="typed"
          class="delete-dialog__input"
          type="text"
          autocomplete="off"
        >
      </div>
      <p
        v-if="failure"
        class="delete-dialog__error"
        role="alert"
      >
        {{ failure.text }}
        <small v-if="failure.requestId">Код запиту: {{ failure.requestId }}</small>
      </p>
      <div class="delete-dialog__actions">
        <button
          type="button"
          class="delete-dialog__button"
          @click="emit('close')"
        >
          Скасувати
        </button>
        <button
          type="submit"
          class="delete-dialog__button delete-dialog__button--danger"
          :disabled="!matches || busy"
        >
          {{ busy ? 'Видаляємо…' : 'Видалити назавжди' }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>

<style scoped>
.delete-dialog__text {
  margin: 0;
  color: var(--color-ink-2);
  font-size: var(--fs-base);
  line-height: 1.55;
}

.delete-dialog__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.delete-dialog__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.delete-dialog__label {
  font-size: var(--fs-control);
  font-weight: 600;
}

.delete-dialog__input {
  height: var(--control-h);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
}

.delete-dialog__input:focus-visible {
  border-color: var(--color-primary);
  outline: 2px solid var(--color-primary-soft);
}

.delete-dialog__error {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  color: var(--color-danger-ink);
  font-size: var(--fs-control);
}

.delete-dialog__error small {
  font-size: var(--fs-2xs);
}

.delete-dialog__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
  margin-top: var(--space-1);
}

.delete-dialog__button {
  flex: 1 1 140px;
  height: var(--control-h);
  padding: 0 var(--space-5);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
  font-weight: 700;
  cursor: pointer;
}

.delete-dialog__button--danger {
  border-color: var(--color-danger);
  background: var(--color-danger);
  color: var(--color-on-primary);
}

.delete-dialog__button--danger:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
</style>
