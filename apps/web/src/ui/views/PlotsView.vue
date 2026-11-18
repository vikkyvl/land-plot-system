<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'

import LocaleSwitch from '../components/LocaleSwitch.vue'
import PlotStatusBadge from '../components/PlotStatusBadge.vue'
import PlotThumbnail from '../components/PlotThumbnail.vue'
import { formatDate, formatHectares, formatPlotCount, initials } from '../composables/format'
import { useFailure, type Failure } from '../composables/useFailure'
import { useFollowedPlots } from '../composables/useFollowedPlots'
import { maxStaggered, plotTransitionName } from '../composables/viewTransitions'
import { useConfirmationsStore } from '../stores/confirmations'
import { usePlotsStore } from '../stores/plots'
import { useSessionStore } from '../stores/session'

const store = usePlotsStore()
const session = useSessionStore()
const { visible, plots, loading, loaded, query, totalHectares } = storeToRefs(store)
const { userName } = storeToRefs(session)
const route = useRoute()
const handle = useFailure()
const failure = ref<Failure | null>(null)
const menuOpen = ref(false)

const confirmations = useConfirmationsStore()
const followed = useFollowedPlots()

watch(plots, (list) => {
  for (const plot of list) {
    if (plot.status === 'PENDING') {
      followed.follow(plot.id)
    }
  }
})

watch(
  () => confirmations.entries,
  (entries) => {
    for (const [id, entry] of Object.entries(entries)) {
      const current = plots.value.find((plot) => plot.id === id)
      if (current === undefined) {
        continue
      }
      if ((entry.phase === 'confirmed' || entry.phase === 'reverted') && entry.plot !== null && entry.plot !== current) {
        store.replace(entry.plot)
      } else if (entry.phase === 'cancelled') {
        store.forget(id)
      }
    }
  },
)

const selectedId = computed(() => (typeof route.params['id'] === 'string' ? route.params['id'] : null))

async function load(): Promise<void> {
  failure.value = null
  try {
    await store.load()
  } catch (error) {
    failure.value = await handle(error)
  }
}

onMounted(load)
</script>

<template>
  <main
    class="plots"
    :class="{ 'plots--selected': selectedId !== null }"
  >
    <section
      class="plots__list"
      aria-labelledby="plots-title"
    >
      <div class="plots__head">
        <h1
          id="plots-title"
          class="plots__title"
        >
          {{ $t('plots.title') }}
        </h1>
        <RouterLink
          class="plots__new plots__new--web ui-primary"
          to="/"
        >
          {{ $t('plots.new') }}
        </RouterLink>
        <div class="plots__profile">
          <button
            type="button"
            class="plots__avatar"
            :aria-label="$t('plots.profileMenu', { initials: initials(userName ?? '') })"
            :aria-expanded="menuOpen"
            @click="menuOpen = !menuOpen"
          >
            {{ initials(userName ?? '') }}
          </button>
          <div
            v-if="menuOpen"
            class="plots__menu"
          >
            <span>{{ userName }}</span>
            <LocaleSwitch />
            <button
              type="button"
              class="plots__menu-action"
              @click="session.signOut()"
            >
              {{ $t('app.signOut') }}
            </button>
          </div>
        </div>
      </div>

      <label
        class="visually-hidden"
        for="plot-search"
      >{{ $t('plots.searchLabel') }}</label>
      <input
        id="plot-search"
        v-model="query"
        class="plots__search"
        type="search"
        :placeholder="$t('plots.searchPlaceholder')"
      >

      <div
        v-if="loading && !loaded"
        class="plots__items"
        aria-busy="true"
        :aria-label="$t('plots.loading')"
      >
        <div
          v-for="index in 3"
          :key="index"
          class="plots__skeleton motion-shimmer"
        />
      </div>
      <div
        v-else-if="failure"
        class="plots__failure"
        role="alert"
      >
        <p>{{ failure.text }}</p>
        <small v-if="failure.requestId">{{ $t('app.requestId', { id: failure.requestId }) }}</small>
        <button
          type="button"
          class="plots__retry ui-secondary"
          @click="load"
        >
          {{ $t('app.retry') }}
        </button>
      </div>
      <div
        v-else-if="loaded && plots.length === 0"
        class="plots__empty"
      >
        <p>{{ $t('plots.empty') }}</p>
        <RouterLink
          class="plots__new ui-primary"
          to="/"
        >
          {{ $t('plots.new') }}
        </RouterLink>
      </div>
      <template v-else>
        <TransitionGroup
          tag="ul"
          name="plots-item"
          class="plots__items"
        >
          <li
            v-for="(plot, index) in visible"
            :key="plot.id"
            class="plots__item"
            :style="{ '--order': Math.min(index, maxStaggered) }"
          >
            <RouterLink
              class="plots__card ui-card"
              :class="{ 'plots__card--selected': plot.id === selectedId }"
              :to="`/plots/${plot.id}`"
              :aria-current="plot.id === selectedId ? 'page' : undefined"
            >
              <PlotThumbnail
                :points="plot.points"
                :style="{ viewTransitionName: plot.id === selectedId ? undefined : plotTransitionName(plot.id) }"
              />
              <span class="plots__card-text">
                <span class="plots__card-name">{{ plot.name }}</span>
                <span class="plots__card-meta">
                  <span class="plots__card-updated">{{ $t('plots.updated') }} </span>{{ formatDate(plot.updatedAt) }} ·
                  {{ $t('plots.versionShort', { n: plot.version }) }}
                </span>
                <PlotStatusBadge :status="plot.status" />
              </span>
              <span class="plots__card-area">{{ formatHectares(plot.measurement.area.hectares) }}</span>
            </RouterLink>
          </li>
        </TransitionGroup>
        <p
          v-if="visible.length === 0"
          class="plots__nothing"
        >
          {{ $t('plots.nothingFound', { query }) }}
        </p>
        <p class="plots__summary">
          <span>{{ $t('plots.total', { count: formatPlotCount(plots.length) }) }}</span>
          <b>{{ formatHectares(totalHectares) }}</b>
        </p>
      </template>

      <RouterLink
        class="plots__new plots__new--mobile ui-primary"
        to="/"
      >
        <svg
          width="20"
          height="20"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <path
            d="M12 5v14M5 12h14"
            stroke="currentColor"
            stroke-width="2.4"
            stroke-linecap="round"
          />
        </svg>
        {{ $t('plots.new') }}
      </RouterLink>
    </section>

    <section class="plots__detail">
      <RouterView />
    </section>
  </main>
</template>

<style scoped>
.plots {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-6);
  max-width: var(--container-max);
  margin: 0 auto;
  padding: var(--space-7) var(--space-6);
}

.plots__list {
  display: flex;
  flex: 1 1 380px;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.plots__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.plots__title {
  margin: 0;
  font-size: var(--fs-h2);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.plots__new {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  height: var(--touch-min);
  padding: 0 var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: var(--fs-control);
  font-weight: 700;
  text-decoration: none;
}

.plots__new--mobile,
.plots__profile,
.plots__search,
.plots__summary {
  display: none;
}

.plots__search {
  width: 100%;
  height: var(--control-h-search);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font: inherit;
  font-size: var(--fs-base);
}

.plots__items {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.plots__items {
  position: relative;
}

.plots__item {
  animation: motion-fade-up var(--motion-base) var(--ease-out) backwards;
  animation-delay: calc(var(--order) * var(--motion-stagger));
}

.plots-item-move {
  transition: transform var(--motion-base) var(--ease-out);
}

.plots-item-leave-active {
  position: absolute;
  right: 0;
  left: 0;
  transition:
    opacity var(--motion-base) var(--ease-out),
    transform var(--motion-base) var(--ease-out);
}

.plots-item-leave-to {
  opacity: 0;
  transform: scaleY(0.4);
  transform-origin: top;
}

.plots__card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 15px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-surface);
  color: var(--color-ink);
  text-decoration: none;
}

.plots__card--selected,
.plots__card--selected:hover {
  padding: 14px;
  border: 2px solid var(--color-primary);
}

.plots__card-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  min-width: 0;
}

.plots__card-name {
  font-size: var(--fs-md);
  font-weight: 700;
  overflow-wrap: anywhere;
}

.plots__card-meta {
  color: var(--color-muted);
  font-size: var(--fs-sm);
}

.plots__card-area {
  font-size: var(--fs-panel-title);
  font-weight: 800;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.plots__skeleton {
  height: 92px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: linear-gradient(90deg, var(--color-surface) 0%, var(--color-surface-muted) 50%, var(--color-surface) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.4s ease-in-out infinite;
}

.plots__failure,
.plots__empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-5);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-surface);
}

.plots__failure {
  border-color: var(--color-danger-border);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink);
}

.plots__failure p,
.plots__empty p,
.plots__nothing {
  margin: 0;
}

.plots__nothing {
  color: var(--color-muted);
}

.plots__retry {
  min-height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

.plots__detail {
  flex: 999 1 560px;
  min-width: 0;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .plots__skeleton {
    animation: none;
  }
}

@media (max-width: 767.98px) {
  .plots {
    padding: calc(18px + env(safe-area-inset-top)) var(--space-5) var(--space-5);
  }

  .plots--selected .plots__list,
  .plots:not(.plots--selected) .plots__detail {
    display: none;
  }

  .plots__detail {
    flex-basis: 100%;
  }

  .plots__list {
    gap: 10px;
  }

  .plots__new--web,
  .plots__card-updated {
    display: none;
  }

  .plots__profile {
    position: relative;
    display: block;
  }

  .plots__avatar {
    width: var(--touch-min);
    height: var(--touch-min);
    border: none;
    border-radius: 50%;
    background: var(--color-primary-soft);
    color: var(--color-primary);
    font: inherit;
    font-size: var(--fs-base);
    font-weight: 800;
    cursor: pointer;
  }

  .plots__menu {
    position: absolute;
    z-index: 900;
    top: calc(100% + var(--space-2));
    right: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-width: 200px;
    padding: var(--space-3);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    box-shadow: var(--shadow-dropdown);
    font-size: var(--fs-control);
  }

  .plots__menu-action {
    min-height: var(--touch-min);
    border: 1.5px solid var(--color-border-strong);
    border-radius: var(--radius-md);
    background: var(--color-surface);
    font: inherit;
    font-weight: 700;
    cursor: pointer;
  }

  .plots__search {
    display: block;
    margin-top: var(--space-2);
  }

  .plots__items {
    gap: 10px;
  }

  .plots__card,
  .plots__card--selected {
    padding: 14px;
    border: 1px solid var(--color-border);
  }

  .plots__card-area {
    font-size: var(--fs-lg);
  }

  .plots__summary {
    display: flex;
    justify-content: space-between;
    margin: 0;
    padding: 14px var(--space-4);
    border-radius: var(--radius-xl);
    background: var(--color-primary-soft);
    color: var(--color-primary-hover);
    font-size: var(--fs-control);
  }

  .plots__new--mobile {
    display: flex;
    height: var(--control-h-lg);
    margin-top: var(--space-3);
    border-radius: var(--radius-lg);
    font-size: var(--fs-md);
  }
}

.plots {
  padding-right: max(var(--space-6), env(safe-area-inset-right));
  padding-left: max(var(--space-6), env(safe-area-inset-left));
}

@media (max-width: 767.98px) {
  .plots {
    padding-right: max(var(--space-5), env(safe-area-inset-right));
    padding-left: max(var(--space-5), env(safe-area-inset-left));
  }
}
</style>
