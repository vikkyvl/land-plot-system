<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, ref, shallowRef, watch } from 'vue'
import { useRouter } from 'vue-router'

import { reportToGeoJson } from '../../domain/geoJson'
import { hasScope, Scope } from '../../domain/profile'
import { isOutdated, isPlotDeleted, reportNumber, type ReportDetails } from '../../domain/reportDetails'
import PlotViewMap from '../components/PlotViewMap.vue'
import { copyText, downloadJson, safeFileName } from '../composables/fileExport'
import { formatAres, formatDate, formatHectareNumber, formatMeters, formatSquareMeters } from '../composables/format'
import { useMapLayers, useMapPreferencesService, useReportService } from '../composables/services'
import { useFailure, type Failure } from '../composables/useFailure'
import { useSessionStore } from '../stores/session'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ id: string }>()

const service = useReportService()
const layers = useMapLayers()
const preferences = useMapPreferencesService()
const toasts = useToastStore()
const router = useRouter()
const handle = useFailure()
const { profile } = storeToRefs(useSessionStore())

const details = shallowRef<ReportDetails | null>(null)
const loading = ref(true)
const failure = ref<Failure | null>(null)
const refreshing = ref(false)

const layer = computed(() => layers[preferences.layer()])
const report = computed(() => details.value?.report ?? null)
const outdated = computed(() => details.value !== null && isOutdated(details.value))
const deleted = computed(() => details.value !== null && isPlotDeleted(details.value))
const points = computed(() => details.value?.points ?? null)
const canReport = computed(() => profile.value === null || hasScope(profile.value, Scope.ReportsWrite))
const owner = computed(() => profile.value?.email ?? profile.value?.name ?? null)
const backLink = computed(() =>
  report.value !== null && !deleted.value
    ? { to: `/plots/${report.value.plotId}`, label: 'До ділянки' }
    : { to: '/reports', label: 'До звітів' },
)

async function load(id: string): Promise<void> {
  loading.value = true
  failure.value = null
  details.value = null
  try {
    details.value = await service.details(id)
  } catch (error) {
    failure.value = await handle(error)
  } finally {
    loading.value = false
  }
}

watch(() => props.id, load, { immediate: true })

async function refresh(): Promise<void> {
  if (report.value === null) {
    return
  }
  refreshing.value = true
  try {
    const created = await service.create(report.value.plotId)
    toasts.show('Сформовано новий звіт для поточної версії')
    await router.push(`/reports/${created.id}`)
  } catch (error) {
    await handle(error, { toast: true })
  } finally {
    refreshing.value = false
  }
}

function download(): void {
  if (report.value === null || points.value === null) {
    return
  }
  downloadJson(`${safeFileName(report.value.plotName)}-${reportNumber(report.value)}.geojson`, reportToGeoJson(report.value, points.value))
}

async function share(): Promise<void> {
  toasts.show(
    (await copyText(window.location.href)) ? 'Посилання на звіт скопійовано' : 'Не вдалося скопіювати посилання',
  )
}
</script>

<template>
  <main
    class="report"
    :aria-busy="loading"
  >
    <div
      v-if="loading"
      class="report__skeleton"
      aria-label="Завантаження звіту"
    >
      <div class="report__skeleton-line" />
      <div class="report__skeleton-block" />
    </div>

    <div
      v-else-if="failure"
      class="report__failure"
      role="alert"
    >
      <p>{{ failure.text }}</p>
      <small v-if="failure.requestId">Код запиту: {{ failure.requestId }}</small>
      <RouterLink to="/reports">
        До звітів
      </RouterLink>
    </div>

    <template v-else-if="report">
      <div class="report__bar">
        <RouterLink
          class="report__back-icon"
          :to="backLink.to"
          :aria-label="backLink.label"
        >
          <svg
            width="22"
            height="22"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              d="M15 5 L8 12 L15 19"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
        </RouterLink>
        <span class="report__bar-title">Звіт № {{ reportNumber(report) }}</span>
      </div>

      <header class="report__head">
        <div class="report__heading">
          <RouterLink
            class="report__back"
            :to="backLink.to"
          >
            ← {{ backLink.label }}
          </RouterLink>
          <h1 class="report__title">
            Звіт · {{ report.plotName }}
          </h1>
        </div>
        <div class="report__actions">
          <button
            type="button"
            class="report__button report__download"
            :disabled="points === null"
            :title="points === null ? 'Контур ділянки недоступний' : undefined"
            @click="download"
          >
            Завантажити GeoJSON
          </button>
          <button
            type="button"
            class="report__button report__button--primary report__share"
            @click="share"
          >
            <span class="report__share-web">Поділитися посиланням</span>
            <span class="report__share-mobile">Поділитися звітом</span>
          </button>
        </div>
      </header>

      <div
        v-if="outdated && details?.plot"
        class="report__banner"
        role="status"
      >
        <svg
          class="report__banner-icon"
          width="22"
          height="22"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            cx="12"
            cy="12"
            r="9.5"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          />
          <path
            d="M12 7v6.5"
            stroke="currentColor"
            stroke-width="2.2"
            stroke-linecap="round"
          />
          <circle
            cx="12"
            cy="17"
            r="1.3"
            fill="currentColor"
          />
        </svg>
        <div class="report__banner-text">
          <b>Ділянку змінено після формування звіту.</b>
          <span>Звіт побудовано для версії {{ report.plotVersion }}. Поточна версія {{ details.plot.version }}.</span>
        </div>
        <button
          v-if="canReport"
          type="button"
          class="report__banner-button"
          :disabled="refreshing"
          @click="refresh"
        >
          {{ refreshing ? 'Формуємо…' : 'Оновити звіт' }}
        </button>
      </div>

      <div
        v-if="deleted"
        class="report__deleted"
        role="status"
      >
        <span class="report__badge">Ділянку видалено</span>
        <span>Звіт збережено з даними на момент формування. Контур і координати більше недоступні.</span>
      </div>

      <article class="report__card">
        <div class="report__meta">
          <span class="report__meta-number">Звіт № {{ reportNumber(report) }} · сформовано {{ formatDate(report.createdAt) }}</span>
          <span
            v-if="owner"
            class="report__meta-owner"
          >Власник · {{ owner }}</span>
        </div>
        <div class="report__card-heading">
          <span class="report__card-name">{{ report.plotName }}</span>
          <span class="report__card-date">Сформовано {{ formatDate(report.createdAt) }}</span>
        </div>

        <PlotViewMap
          v-if="points"
          class="report__map"
          :points="points"
          :layer="layer"
        />

        <dl class="report__tiles">
          <div class="report__tile">
            <dt>Гектари</dt>
            <dd>{{ formatHectareNumber(report.measurement.area.hectares) }}</dd>
          </div>
          <div class="report__tile">
            <dt>Ари</dt>
            <dd>{{ formatAres(report.measurement.area.ares) }}</dd>
          </div>
          <div class="report__tile">
            <dt>м²</dt>
            <dd>{{ formatSquareMeters(report.measurement.area.squareMeters) }}</dd>
          </div>
          <div class="report__tile">
            <dt>Периметр</dt>
            <dd>{{ formatMeters(report.measurement.perimeterMeters) }}</dd>
          </div>
        </dl>

        <section
          v-if="points"
          class="report__coordinates"
          aria-labelledby="coordinates-title"
        >
          <h2
            id="coordinates-title"
            class="report__coordinates-title"
          >
            Координати точок межі (WGS 84)
          </h2>
          <div class="report__table-wrap">
            <table class="report__table">
              <thead>
                <tr>
                  <th scope="col">
                    №
                  </th>
                  <th scope="col">
                    Широта
                  </th>
                  <th scope="col">
                    Довгота
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="(point, index) in points"
                  :key="index"
                >
                  <td>{{ index + 1 }}</td>
                  <td>{{ point.latitude.toFixed(5) }}</td>
                  <td>{{ point.longitude.toFixed(5) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>

        <p class="report__note">
          <span class="report__note-web">Площу розраховано за координатами, переведеними в метри, за формулою Гаусса. Результат
            наближений і не є кадастровим вимірюванням.</span>
          <span class="report__note-mobile">Розрахунок наближений і не є кадастровим вимірюванням.</span>
        </p>
      </article>
    </template>
  </main>
</template>

<style scoped>
.report {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  max-width: 960px;
  margin: 0 auto;
  padding: var(--space-7) var(--space-6);
}

.report__bar,
.report__card-heading,
.report__share-mobile,
.report__note-mobile {
  display: none;
}

.report__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.report__heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.report__back {
  font-size: var(--fs-control);
  font-weight: 600;
  text-decoration: none;
}

.report__title {
  margin: 0;
  font-size: var(--fs-h2);
  font-weight: 800;
  letter-spacing: -0.02em;
  overflow-wrap: anywhere;
}

.report__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.report__button {
  height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  cursor: pointer;
}

.report__button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.report__button--primary {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-weight: 700;
}

.report__banner {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-3);
  padding: 14px var(--space-4);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-lg);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
}

.report__banner-icon {
  flex: none;
  color: var(--color-warning-icon);
}

.report__banner-text {
  display: flex;
  flex: 1 1 260px;
  flex-direction: column;
  gap: 2px;
  font-size: var(--fs-control);
}

.report__banner-button {
  flex: none;
  height: var(--control-h-sm);
  margin-left: auto;
  padding: 0 14px;
  border: 1.5px solid var(--color-warning-border-strong);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  color: var(--color-warning-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 700;
  cursor: pointer;
}

.report__deleted {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-3);
  padding: 14px var(--space-4);
  border: 1px solid var(--color-danger-border-soft);
  border-radius: var(--radius-lg);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink-2);
  font-size: var(--fs-control);
}

.report__badge {
  padding: 2px var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-danger-ink);
  font-size: var(--fs-xs);
  font-weight: 800;
}

.report__card {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: var(--space-7);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
}

.report__meta {
  display: flex;
  flex-wrap: wrap;
  justify-content: space-between;
  gap: var(--space-2);
  color: var(--color-muted);
  font-size: var(--fs-control);
}

.report__map {
  height: 260px;
}

.report__tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.report__tile {
  padding: var(--space-3) 14px;
  border-radius: var(--radius-md);
  background: var(--color-surface-muted);
}

.report__tile dt {
  color: var(--color-muted);
  font-size: var(--fs-xs);
}

.report__tile dd {
  margin: 0;
  font-size: var(--fs-h3);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.report__coordinates {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.report__coordinates-title {
  margin: 0;
  font-size: var(--fs-md);
  font-weight: 700;
}

.report__table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.report__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-control);
  font-variant-numeric: tabular-nums;
}

.report__table th {
  padding: 10px 14px;
  background: var(--color-surface-muted);
  font-weight: 700;
  text-align: left;
}

.report__table td {
  padding: 9px 14px;
  border-top: 1px solid var(--color-border);
}

.report__note {
  margin: 0;
  color: var(--color-muted);
  font-size: var(--fs-sm);
  line-height: 1.5;
}

.report__failure {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
  color: var(--color-danger-ink);
}

.report__failure p {
  margin: 0;
}

.report__skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.report__skeleton-line,
.report__skeleton-block {
  height: 32px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-muted);
}

.report__skeleton-block {
  height: 420px;
  border-radius: var(--radius-2xl);
}

@media (max-width: 767.98px) {
  .report {
    gap: 14px;
    padding: calc(14px + env(safe-area-inset-top)) var(--space-5) var(--space-6);
  }

  .report__bar {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    margin: 0 calc(-1 * var(--space-3));
  }

  .report__back-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: var(--touch-min);
    height: var(--touch-min);
    color: var(--color-ink);
  }

  .report__bar-title {
    font-size: var(--fs-lg);
    font-weight: 700;
  }

  .report__head {
    display: contents;
  }

  .report__heading,
  .report__meta,
  .report__coordinates,
  .report__share-web,
  .report__note-web {
    display: none;
  }

  .report__share-mobile,
  .report__note-mobile {
    display: inline;
  }

  .report__banner {
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-3) 14px;
  }

  .report__banner-icon {
    display: none;
  }

  .report__banner-text {
    flex: none;
  }

  .report__banner-button {
    margin-left: 0;
  }

  .report__card {
    gap: 14px;
    padding: 18px;
  }

  .report__card-heading {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .report__card-name {
    font-size: var(--fs-panel-title);
    font-weight: 800;
  }

  .report__card-date {
    color: var(--color-muted);
    font-size: var(--fs-sm);
  }

  .report__map {
    height: 150px;
    border-radius: var(--radius-md);
  }

  .report__tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-2);
  }

  .report__tile {
    padding: 10px var(--space-3);
  }

  .report__tile dd {
    font-size: var(--fs-tile);
  }

  .report__note {
    font-size: var(--fs-xs);
  }

  .report__actions {
    display: flex;
    flex-direction: column-reverse;
    gap: 10px;
    order: 10;
    margin-top: var(--space-2);
  }

  .report__button {
    height: var(--control-h);
    border-radius: var(--radius-lg);
    font-size: var(--fs-base);
    font-weight: 700;
  }

  .report__share {
    height: var(--control-h-lg);
    font-size: var(--fs-md);
  }
}

.report__back {
  display: inline-flex;
  align-items: center;
}

@media (pointer: coarse) {
  .report__back,
  .report__banner-button {
    min-height: var(--touch-min);
  }
}

.report {
  padding-right: max(var(--space-6), env(safe-area-inset-right));
  padding-left: max(var(--space-6), env(safe-area-inset-left));
}

@media (max-width: 767.98px) {
  .report {
    padding-right: max(var(--space-5), env(safe-area-inset-right));
    padding-left: max(var(--space-5), env(safe-area-inset-left));
  }
}
</style>
