<script setup lang="ts">
import { computed, ref } from 'vue'

import { obstructionReasons, type ObstructionReason } from '../../domain/survey'
import { useFocusTrap } from '../composables/useFocusTrap'

const props = defineProps<{
  side: number
  pointCount: number
  reason: ObstructionReason | null
  anchor: { readonly x: number; readonly y: number }
}>()

const emit = defineEmits<{
  insert: []
  mark: [reason: ObstructionReason | null]
  close: []
}>()

const menuWidth = 260
const menuHeight = 300
const anchorGap = 12

const menu = ref<HTMLElement | null>(null)
const titleId = `side-menu-${Math.random().toString(36).slice(2)}`
const onKeydown = useFocusTrap(menu, () => emit('close'))

const choices: readonly (ObstructionReason | null)[] = [null, ...obstructionReasons]
const position = computed(() => ({
  '--menu-left': `${Math.max(anchorGap, Math.min(props.anchor.x + anchorGap, window.innerWidth - menuWidth - anchorGap))}px`,
  '--menu-top': `${Math.max(anchorGap, Math.min(props.anchor.y + anchorGap, window.innerHeight - menuHeight - anchorGap))}px`,
}))
const from = computed(() => props.side + 1)
const to = computed(() => ((props.side + 1) % props.pointCount) + 1)

function choose(reason: ObstructionReason | null): void {
  emit('mark', reason)
}
</script>

<template>
  <Teleport to="body">
    <div
      class="side-menu__overlay"
      @mousedown.self="emit('close')"
    >
      <div
        ref="menu"
        class="side-menu"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="titleId"
        :style="position"
        @keydown="onKeydown"
      >
        <h2
          :id="titleId"
          class="side-menu__title"
        >
          {{ $t('sides.side', { from, to }) }}
        </h2>
        <button
          type="button"
          class="side-menu__insert ui-secondary"
          @click="emit('insert')"
        >
          {{ $t('sides.insertHere') }}
        </button>
        <p class="side-menu__label">
          {{ $t('sides.coveredBy') }}
        </p>
        <div class="side-menu__choices">
          <button
            v-for="choice in choices"
            :key="choice ?? 'none'"
            type="button"
            class="side-menu__choice ui-secondary"
            :aria-pressed="reason === choice"
            @click="choose(choice)"
          >
            {{ $t(`sides.${choice ?? 'none'}`) }}
          </button>
        </div>
        <button
          type="button"
          class="side-menu__cancel ui-secondary"
          @click="emit('close')"
        >
          {{ $t('sides.cancel') }}
        </button>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.side-menu__overlay {
  position: fixed;
  z-index: 2000;
  inset: 0;
  animation: motion-fade-in var(--motion-fast) var(--ease-out);
}

.side-menu {
  position: fixed;
  top: var(--menu-top);
  left: var(--menu-left);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  width: 260px;
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
  box-shadow: var(--shadow-dialog);
  animation: motion-fade-up var(--motion-fast) var(--ease-out);
}

.side-menu__title {
  margin: 0;
  font-size: var(--fs-base);
  font-weight: 800;
}

.side-menu__label {
  margin: var(--space-2) 0 0;
  color: var(--color-ink-3);
  font-size: var(--fs-sm);
}

.side-menu__choices {
  display: grid;
  grid-template-columns: 1fr;
  gap: var(--space-1);
}

.side-menu__insert,
.side-menu__choice,
.side-menu__cancel {
  min-height: 40px;
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}

.side-menu__insert {
  border-color: var(--color-primary);
  color: var(--color-primary);
  font-weight: 700;
}

.side-menu__choice[aria-pressed='true'] {
  border-color: var(--color-primary);
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

.side-menu__cancel {
  display: none;
  text-align: center;
}

@media (max-width: 767.98px) {
  .side-menu__overlay {
    background: var(--overlay);
  }

  .side-menu {
    top: auto;
    right: 0;
    bottom: 0;
    left: 0;
    width: auto;
    padding: var(--space-5) var(--space-5) calc(var(--space-5) + env(safe-area-inset-bottom));
    border-radius: var(--radius-2xl) var(--radius-2xl) 0 0;
    animation: motion-sheet-up var(--motion-base) var(--ease-spring);
  }

  .side-menu__choices {
    grid-template-columns: 1fr 1fr;
  }

  .side-menu__insert,
  .side-menu__choice,
  .side-menu__cancel {
    min-height: var(--touch-min);
    text-align: center;
  }

  .side-menu__cancel {
    display: block;
  }
}
</style>
