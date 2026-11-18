<script setup lang="ts">
import { ref } from 'vue'

import type { Failure } from '../composables/useFailure'

import BaseDialog from './BaseDialog.vue'

const props = defineProps<{ name: string; busy: boolean; failure: Failure | null }>()
const emit = defineEmits<{ confirm: [name: string]; close: [] }>()

const value = ref(props.name)
const inputId = `rename-${Math.random().toString(36).slice(2)}`
</script>

<template>
  <BaseDialog
    :title="$t('dialogs.renameTitle')"
    @close="emit('close')"
  >
    <form
      class="rename-dialog"
      @submit.prevent="!busy && emit('confirm', value)"
    >
      <div class="rename-dialog__field">
        <label
          class="rename-dialog__label"
          :for="inputId"
        >{{ $t('dialogs.newName') }}</label>
        <input
          :id="inputId"
          v-model="value"
          class="rename-dialog__input"
          type="text"
          maxlength="200"
          autocomplete="off"
        >
      </div>
      <p
        v-if="failure"
        class="rename-dialog__error"
        role="alert"
      >
        {{ failure.text }}
        <small v-if="failure.requestId">{{ $t('app.requestId', { id: failure.requestId }) }}</small>
      </p>
      <div class="rename-dialog__actions">
        <button
          type="button"
          class="rename-dialog__button ui-secondary"
          @click="emit('close')"
        >
          {{ $t('app.cancel') }}
        </button>
        <button
          type="submit"
          class="rename-dialog__button rename-dialog__button--primary ui-primary"
          :disabled="busy"
        >
          {{ busy ? $t('dialogs.saving') : $t('dialogs.save') }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>

<style scoped>
.rename-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.rename-dialog__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.rename-dialog__label {
  font-size: var(--fs-control);
  font-weight: 600;
}

.rename-dialog__input {
  height: var(--control-h);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
}

.rename-dialog__input:focus-visible {
  border-color: var(--color-primary);
  outline: 2px solid var(--color-primary-soft);
}

.rename-dialog__error {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  color: var(--color-danger-ink);
  font-size: var(--fs-control);
}

.rename-dialog__error small {
  font-size: var(--fs-2xs);
}

.rename-dialog__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.rename-dialog__button {
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

.rename-dialog__button--primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
}
</style>
