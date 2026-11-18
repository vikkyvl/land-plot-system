<script setup lang="ts">
import { computed, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'

import type { Report } from '../../domain/report'
import { reportNumber } from '../../domain/reportDetails'
import { formatDate, formatHectares } from '../composables/format'
import { useReportService } from '../composables/services'
import { useFailure, type Failure } from '../composables/useFailure'

const service = useReportService()
const handle = useFailure()
const route = useRoute()
const reports = shallowRef<readonly Report[]>([])
const loading = ref(true)
const failure = ref<Failure | null>(null)
const query = ref('')

const plotId = computed(() => (typeof route.query['plot_id'] === 'string' ? route.query['plot_id'] : undefined))
const visible = computed(() => {
  const needle = query.value.trim().toLocaleLowerCase('uk-UA')

  return needle === ''
    ? reports.value
    : reports.value.filter((report) => report.plotName.toLocaleLowerCase('uk-UA').includes(needle))
})

async function load(): Promise<void> {
  loading.value = true
  failure.value = null
  try {
    reports.value = await service.list({ plotId: plotId.value })
  } catch (error) {
    failure.value = await handle(error)
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(plotId, load)
</script>

<template>
  <main class="reports">
    <div class="reports__head">
      <h1 class="reports__title">
        {{ $t('reports.title') }}
      </h1>
      <RouterLink
        v-if="plotId"
        class="reports__all"
        to="/reports"
      >
        {{ $t('reports.all') }}
      </RouterLink>
    </div>
    <p
      v-if="plotId && reports[0]"
      class="reports__filter"
    >
      {{ $t('reports.ofPlot', { name: reports[0].plotName }) }}
    </p>

    <label
      class="visually-hidden"
      for="report-search"
    >{{ $t('reports.searchLabel') }}</label>
    <input
      id="report-search"
      v-model="query"
      class="reports__search"
      type="search"
      :placeholder="$t('reports.searchPlaceholder')"
    >

    <div
      v-if="loading"
      class="reports__items"
      aria-busy="true"
      :aria-label="$t('reports.loading')"
    >
      <div
        v-for="index in 3"
        :key="index"
        class="reports__skeleton"
      />
    </div>
    <div
      v-else-if="failure"
      class="reports__failure"
      role="alert"
    >
      <p>{{ failure.text }}</p>
      <small v-if="failure.requestId">{{ $t('app.requestId', { id: failure.requestId }) }}</small>
      <button
        type="button"
        class="reports__retry"
        @click="load"
      >
        {{ $t('app.retry') }}
      </button>
    </div>
    <div
      v-else-if="reports.length === 0"
      class="reports__empty"
    >
      <p>{{ $t('reports.empty') }}</p>
      <RouterLink
        class="reports__link"
        to="/plots"
      >
        {{ $t('reports.toPlots') }}
      </RouterLink>
    </div>
    <template v-else>
      <ul class="reports__items">
        <li
          v-for="report in visible"
          :key="report.id"
        >
          <RouterLink
            class="reports__card"
            :to="`/reports/${report.id}`"
          >
            <span
              class="reports__icon"
              aria-hidden="true"
            >
              <svg
                width="26"
                height="26"
                viewBox="0 0 24 24"
              >
                <path
                  d="M7 3 H14 L18 7 V21 H7 Z"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linejoin="round"
                />
                <path
                  d="M10 12 H15 M10 16 H15"
                  stroke="currentColor"
                  stroke-width="2"
                />
              </svg>
            </span>
            <span class="reports__card-text">
              <span class="reports__card-name">{{ report.plotName }}</span>
              <span class="reports__card-meta">{{
                $t('reports.cardMeta', {
                  number: reportNumber(report),
                  date: formatDate(report.createdAt),
                  version: report.plotVersion,
                })
              }}</span>
              <span
                v-if="report.plotDeleted"
                class="reports__badge"
              >{{ $t('reports.plotDeleted') }}</span>
            </span>
            <span class="reports__card-area">{{ formatHectares(report.measurement.area.hectares) }}</span>
          </RouterLink>
        </li>
      </ul>
      <p
        v-if="visible.length === 0"
        class="reports__nothing"
      >
        {{ $t('reports.nothingFound', { query }) }}
      </p>
    </template>
  </main>
</template>

<style scoped>
.reports {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-width: 720px;
  margin: 0 auto;
  padding: var(--space-7) var(--space-6);
}

.reports__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.reports__title {
  margin: 0;
  font-size: var(--fs-h2);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.reports__all,
.reports__link {
  font-size: var(--fs-control);
  font-weight: 700;
}

.reports__filter,
.reports__nothing {
  margin: 0;
  color: var(--color-muted);
}

.reports__search {
  width: 100%;
  height: var(--control-h-search);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font: inherit;
  font-size: var(--fs-base);
}

.reports__items {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.reports__card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-surface);
  color: var(--color-ink);
  text-decoration: none;
}

.reports__card:hover {
  border-color: var(--color-border-strong);
  color: var(--color-ink);
}

.reports__icon {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 60px;
  height: 60px;
  border-radius: var(--radius-xs);
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

.reports__card-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: flex-start;
  gap: 3px;
  min-width: 0;
}

.reports__card-name {
  font-size: var(--fs-md);
  font-weight: 700;
  overflow-wrap: anywhere;
}

.reports__card-meta {
  color: var(--color-muted);
  font-size: var(--fs-sm);
}

.reports__badge {
  padding: 2px var(--space-2);
  border: 1px solid var(--color-danger-border-soft);
  border-radius: var(--radius-sm);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink);
  font-size: var(--fs-xs);
  font-weight: 700;
}

.reports__card-area {
  font-size: var(--fs-lg);
  font-weight: 800;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.reports__skeleton {
  height: 90px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-surface-muted);
}

.reports__failure,
.reports__empty {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-5);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  background: var(--color-surface);
}

.reports__failure {
  border-color: var(--color-danger-border);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink);
}

.reports__failure p,
.reports__empty p {
  margin: 0;
}

.reports__retry {
  min-height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}

@media (max-width: 767.98px) {
  .reports {
    gap: 10px;
    padding: calc(18px + env(safe-area-inset-top)) var(--space-5) var(--space-5);
  }
}

.reports__all {
  display: inline-flex;
  align-items: center;
}

@media (pointer: coarse) {
  .reports__all {
    min-height: var(--touch-min);
  }
}

.reports {
  padding-right: max(var(--space-6), env(safe-area-inset-right));
  padding-left: max(var(--space-6), env(safe-area-inset-left));
}

@media (max-width: 767.98px) {
  .reports {
    padding-right: max(var(--space-5), env(safe-area-inset-right));
    padding-left: max(var(--space-5), env(safe-area-inset-left));
  }
}
</style>
