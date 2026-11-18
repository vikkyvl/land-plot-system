<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'

import type { Place } from '../../domain/place'
import { usePlaceSearchService } from '../composables/services'

defineProps<{ placeholder: string }>()
const emit = defineEmits<{ select: [place: Place] }>()

const searchDelay = 400
const service = usePlaceSearchService()
const query = ref('')
const places = ref<Place[]>([])
const active = ref(-1)
const open = ref(false)
const failed = ref(false)
const listId = `place-search-${Math.random().toString(36).slice(2)}`
let timer: ReturnType<typeof setTimeout> | null = null
let controller: AbortController | null = null

const expanded = computed(() => open.value && (places.value.length > 0 || failed.value))

function cancel(): void {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
  controller?.abort()
  controller = null
}

function onInput(): void {
  cancel()
  failed.value = false
  timer = setTimeout(search, searchDelay)
}

async function search(): Promise<void> {
  timer = null
  controller = new AbortController()
  const current = controller
  try {
    const found = await service.search(query.value, current.signal)
    if (current === controller) {
      places.value = found
      active.value = found.length > 0 ? 0 : -1
      open.value = true
    }
  } catch {
    if (current === controller && !current.signal.aborted) {
      places.value = []
      failed.value = true
      open.value = true
    }
  }
}

function choose(place: Place): void {
  query.value = place.name
  open.value = false
  emit('select', place)
}

function onKeydown(event: KeyboardEvent): void {
  if (!expanded.value) {
    return
  }
  if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
    event.preventDefault()
    const step = event.key === 'ArrowDown' ? 1 : -1
    active.value = (active.value + step + places.value.length) % places.value.length
  } else if (event.key === 'Enter' && places.value[active.value]) {
    event.preventDefault()
    choose(places.value[active.value] as Place)
  } else if (event.key === 'Escape') {
    open.value = false
  }
}

onBeforeUnmount(cancel)
</script>

<template>
  <div class="place-search">
    <label
      class="visually-hidden"
      :for="`${listId}-input`"
    >Пошук місця</label>
    <input
      :id="`${listId}-input`"
      v-model="query"
      class="place-search__input"
      type="search"
      role="combobox"
      autocomplete="off"
      :placeholder="placeholder"
      :aria-expanded="expanded"
      :aria-controls="listId"
      :aria-activedescendant="active >= 0 && expanded ? `${listId}-${active}` : undefined"
      @input="onInput"
      @keydown="onKeydown"
      @blur="open = false"
      @focus="open = places.length > 0"
    >
    <div
      v-show="expanded"
      class="place-search__dropdown"
    >
      <ul
        :id="listId"
        class="place-search__list"
        role="listbox"
        aria-label="Знайдені місця"
      >
        <li
          v-for="(place, index) in places"
          :id="`${listId}-${index}`"
          :key="`${place.coordinate.latitude},${place.coordinate.longitude}`"
          class="place-search__option"
          :class="{ 'place-search__option--active': index === active }"
          role="option"
          :aria-selected="index === active"
          @mousedown.prevent="choose(place)"
        >
          <strong>{{ place.name }}</strong><span v-if="place.description">, {{ place.description }}</span>
        </li>
      </ul>
      <p
        v-if="failed"
        class="place-search__note"
        role="alert"
      >
        Пошук зараз недоступний.
      </p>
      <p class="place-search__note">
        Пошук: © OpenStreetMap, Nominatim
      </p>
    </div>
  </div>
</template>

<style scoped>
.place-search {
  position: relative;
  flex: 1 1 260px;
  min-width: 0;
}

.place-search__input {
  width: 100%;
  height: var(--control-h-search);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
}

.place-search__input:focus-visible {
  border-color: var(--color-primary);
  outline: none;
}

.place-search__dropdown {
  position: absolute;
  z-index: 1000;
  top: calc(100% + var(--space-1));
  right: 0;
  left: 0;
  overflow: hidden;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  box-shadow: var(--shadow-dropdown);
}

.place-search__list {
  margin: 0;
  padding: var(--space-2) 0 0;
  list-style: none;
}

.place-search__option {
  padding: var(--space-3) var(--space-4);
  font-size: var(--fs-base);
  cursor: pointer;
}

.place-search__option--active {
  background: var(--color-primary-soft);
}

.place-search__note {
  margin: 0;
  padding: var(--space-2) var(--space-4) var(--space-3);
  color: var(--color-muted);
  font-size: var(--fs-xs);
  text-align: right;
}
</style>
