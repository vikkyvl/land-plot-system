<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, ref, shallowRef, watch } from 'vue'
import { useRouter } from 'vue-router'

import { isPending, type Plot, type PlotVersion } from '../../domain/plot'
import { hasScope, Scope } from '../../domain/profile'
import DeletePlotDialog from '../components/DeletePlotDialog.vue'
import PlotStatusBadge from '../components/PlotStatusBadge.vue'
import PlotViewMap from '../components/PlotViewMap.vue'
import RenamePlotDialog from '../components/RenamePlotDialog.vue'
import {
  formatAres,
  formatDate,
  formatHectares,
  formatMeters,
  formatPointCount,
  formatSquareMeters,
} from '../composables/format'
import { useMapLayers, useMapPreferencesService, usePlotService } from '../composables/services'
import { useFailure, type Failure } from '../composables/useFailure'
import { usePlotsStore } from '../stores/plots'
import { useSessionStore } from '../stores/session'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ id: string }>()

const service = usePlotService()
const layers = useMapLayers()
const preferences = useMapPreferencesService()
const plotsStore = usePlotsStore()
const toasts = useToastStore()
const router = useRouter()
const handle = useFailure()
const { profile } = storeToRefs(useSessionStore())

const plot = shallowRef<Plot | null>(null)
const versions = shallowRef<readonly PlotVersion[] | null>(null)
const loading = ref(true)
const failure = ref<Failure | null>(null)
const dialog = ref<'rename' | 'delete' | null>(null)
const busy = ref(false)
const dialogFailure = ref<Failure | null>(null)
const reporting = ref(false)

const layer = computed(() => layers[preferences.layer()])
const pending = computed(() => plot.value !== null && isPending(plot.value))
const canWrite = computed(() => profile.value === null || hasScope(profile.value, Scope.PlotsWrite))
const canReport = computed(() => profile.value === null || hasScope(profile.value, Scope.ReportsWrite))
const pendingHint = 'Ділянка ще обробляється'

async function load(id: string): Promise<void> {
  loading.value = true
  failure.value = null
  plot.value = null
  versions.value = null
  try {
    const [loadedPlot, loadedVersions] = await Promise.all([service.get(id), service.versions(id)])
    plot.value = loadedPlot
    versions.value = loadedVersions
  } catch (error) {
    failure.value = await handle(error)
  } finally {
    loading.value = false
  }
}

watch(() => props.id, load, { immediate: true })

function open(kind: 'rename' | 'delete'): void {
  dialogFailure.value = null
  dialog.value = kind
}

async function rename(name: string): Promise<void> {
  if (plot.value === null) {
    return
  }
  busy.value = true
  try {
    const renamed = await service.rename(plot.value.id, name)
    plot.value = renamed
    plotsStore.replace(renamed)
    dialog.value = null
    toasts.show('Назву змінено. Сервер підтверджує зміни.')
  } catch (error) {
    dialogFailure.value = await handle(error)
  } finally {
    busy.value = false
  }
}

async function remove(): Promise<void> {
  if (plot.value === null) {
    return
  }
  const { id, name } = plot.value
  busy.value = true
  try {
    await service.remove(id)
    plotsStore.forget(id)
    dialog.value = null
    toasts.show(`Ділянку «${name}» видалено`)
    await router.push('/plots')
  } catch (error) {
    dialogFailure.value = await handle(error)
  } finally {
    busy.value = false
  }
}

async function createReport(): Promise<void> {
  if (plot.value === null) {
    return
  }
  reporting.value = true
  try {
    await service.createReport(plot.value.id)
    toasts.show('Звіт сформовано')
  } catch (error) {
    await handle(error, { toast: true })
  } finally {
    reporting.value = false
  }
}
</script>

<template>
  <article
    class="detail"
    :aria-busy="loading"
  >
    <div
      v-if="loading"
      class="detail__skeleton"
      aria-label="Завантаження ділянки"
    >
      <div class="detail__skeleton-line" />
      <div class="detail__skeleton-map" />
      <div class="detail__skeleton-line detail__skeleton-line--short" />
    </div>

    <div
      v-else-if="failure"
      class="detail__failure"
      role="alert"
    >
      <p>{{ failure.text }}</p>
      <small v-if="failure.requestId">Код запиту: {{ failure.requestId }}</small>
      <RouterLink to="/plots">
        До списку ділянок
      </RouterLink>
    </div>

    <template v-else-if="plot">
      <div class="detail__bar">
        <RouterLink
          class="detail__back"
          to="/plots"
          aria-label="Назад до списку"
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
        <span class="detail__bar-title">{{ plot.name }}</span>
        <button
          v-if="canWrite"
          type="button"
          class="detail__trash"
          aria-label="Видалити ділянку"
          :disabled="pending"
          :title="pending ? pendingHint : undefined"
          @click="open('delete')"
        >
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <path
              d="M5 7 H19 M10 7 V4 H14 V7 M7 7 L8 20 H16 L17 7"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linejoin="round"
            />
          </svg>
        </button>
      </div>

      <header class="detail__head">
        <div class="detail__heading">
          <h2 class="detail__title">
            {{ plot.name }}
          </h2>
          <span class="detail__meta">
            Створено {{ formatDate(plot.createdAt) }} · {{ formatPointCount(plot.points.length) }}
            <PlotStatusBadge :status="plot.status" />
          </span>
        </div>
        <div class="detail__actions">
          <template v-if="canWrite">
            <RouterLink
              v-if="!pending"
              class="detail__button detail__edit"
              :to="`/plots/${plot.id}/edit`"
            >
              Змінити межу
            </RouterLink>
            <button
              v-else
              type="button"
              class="detail__button detail__edit"
              disabled
              :title="pendingHint"
            >
              Змінити межу
            </button>
            <button
              type="button"
              class="detail__button detail__rename"
              :disabled="pending"
              :title="pending ? pendingHint : undefined"
              @click="open('rename')"
            >
              Перейменувати
            </button>
            <button
              type="button"
              class="detail__button detail__button--danger detail__delete"
              :disabled="pending"
              :title="pending ? pendingHint : undefined"
              @click="open('delete')"
            >
              Видалити
            </button>
          </template>
          <button
            v-if="canReport"
            type="button"
            class="detail__button detail__button--primary detail__report"
            :disabled="pending || reporting"
            :title="pending ? pendingHint : undefined"
            @click="createReport"
          >
            {{ reporting ? 'Формуємо…' : 'Сформувати звіт' }}
          </button>
        </div>
        <p
          v-if="pending"
          class="detail__pending"
        >
          {{ pendingHint }}. Зміни і звіт стануть доступні після підтвердження.
        </p>
      </header>

      <PlotViewMap
        class="detail__map"
        :points="plot.points"
        :layer="layer"
      />

      <div class="detail__summary">
        <p class="detail__hectares">
          {{ formatHectares(plot.measurement.area.hectares) }}
        </p>
        <p class="detail__line">
          {{ formatSquareMeters(plot.measurement.area.squareMeters) }} м² ·
          {{ formatAres(plot.measurement.area.ares) }} ари · периметр {{ formatMeters(plot.measurement.perimeterMeters) }}
        </p>
      </div>

      <dl class="detail__tiles">
        <div class="detail__tile">
          <dt>Площа</dt>
          <dd>{{ formatHectares(plot.measurement.area.hectares) }}</dd>
        </div>
        <div class="detail__tile">
          <dt>у м²</dt>
          <dd>{{ formatSquareMeters(plot.measurement.area.squareMeters) }}</dd>
        </div>
        <div class="detail__tile">
          <dt>Периметр</dt>
          <dd>{{ formatMeters(plot.measurement.perimeterMeters) }}</dd>
        </div>
      </dl>

      <section
        class="detail__history"
        aria-labelledby="history-title"
      >
        <h3
          id="history-title"
          class="detail__history-title"
        >
          Історія вимірювань
        </h3>
        <p
          v-if="versions && versions.length === 0"
          class="detail__muted"
        >
          Підтверджених версій ще немає.
        </p>
        <template v-else-if="versions">
          <div class="detail__table-wrap">
            <table class="detail__table">
              <thead>
                <tr>
                  <th scope="col">
                    Версія
                  </th>
                  <th scope="col">
                    Дата
                  </th>
                  <th scope="col">
                    Площа
                  </th>
                  <th scope="col">
                    Периметр
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="version in versions"
                  :key="version.version"
                >
                  <td>{{ version.version }}</td>
                  <td>{{ formatDate(version.createdAt) }}</td>
                  <td>{{ formatHectares(version.measurement.area.hectares) }}</td>
                  <td>{{ formatMeters(version.measurement.perimeterMeters) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <ul class="detail__versions">
            <li
              v-for="version in versions"
              :key="version.version"
              class="detail__version"
              :class="{ 'detail__version--current': version.version === plot.version }"
            >
              <span class="detail__version-text">
                <span class="detail__version-name">Версія {{ version.version
                }}<template v-if="version.version === plot.version"> · поточна</template></span>
                <span class="detail__version-meta">{{ formatDate(version.createdAt) }},
                  {{ formatPointCount(version.points.length) }}</span>
              </span>
              <span class="detail__version-area">{{ formatHectares(version.measurement.area.hectares) }}</span>
            </li>
          </ul>
        </template>
      </section>

      <button
        v-if="canReport"
        type="button"
        class="detail__button detail__button--primary detail__report-mobile"
        :disabled="pending || reporting"
        @click="createReport"
      >
        {{ reporting ? 'Формуємо…' : 'Сформувати звіт' }}
      </button>
    </template>

    <RenamePlotDialog
      v-if="dialog === 'rename' && plot"
      :name="plot.name"
      :busy="busy"
      :failure="dialogFailure"
      @confirm="rename"
      @close="dialog = null"
    />
    <DeletePlotDialog
      v-if="dialog === 'delete' && plot"
      :name="plot.name"
      :version-count="versions?.length ?? null"
      :busy="busy"
      :failure="dialogFailure"
      @confirm="remove"
      @close="dialog = null"
    />
  </article>
</template>

<style scoped>
.detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding: var(--space-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
}

.detail__bar,
.detail__summary,
.detail__versions {
  display: none;
}

.detail__head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
}

.detail__heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  min-width: 0;
}

.detail__title {
  margin: 0;
  font-size: var(--fs-title-lg);
  font-weight: 800;
  overflow-wrap: anywhere;
}

.detail__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
  color: var(--color-muted);
  font-size: var(--fs-control);
}

.detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.detail__button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: var(--touch-min);
  padding: 0 var(--space-4);
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
}

.detail__button:hover {
  color: var(--color-ink);
}

.detail__button.detail__report-mobile {
  display: none;
}

.detail__button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.detail__button--danger,
.detail__button--danger:hover {
  border-color: var(--color-danger-border-soft);
  color: var(--color-danger);
}

.detail__button--primary,
.detail__button--primary:hover {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-weight: 700;
}

.detail__pending {
  flex-basis: 100%;
  margin: 0;
  color: var(--color-muted);
  font-size: var(--fs-sm);
}

.detail__map {
  height: 280px;
}

.detail__tiles {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.detail__tile {
  padding: var(--space-3) 14px;
  border-radius: var(--radius-md);
  background: var(--color-surface-muted);
}

.detail__tile dt {
  color: var(--color-muted);
  font-size: var(--fs-xs);
}

.detail__tile dd {
  margin: 0;
  font-size: var(--fs-title-md);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.detail__history {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.detail__history-title {
  margin: 0;
  font-size: var(--fs-md);
  font-weight: 700;
}

.detail__muted {
  margin: 0;
  color: var(--color-muted);
  font-size: var(--fs-control);
}

.detail__table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.detail__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-control);
  font-variant-numeric: tabular-nums;
}

.detail__table th {
  padding: 10px 14px;
  background: var(--color-surface-muted);
  font-weight: 700;
  text-align: left;
}

.detail__table td {
  padding: 10px 14px;
  border-top: 1px solid var(--color-border);
}

.detail__failure {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--space-2);
  color: var(--color-danger-ink);
}

.detail__failure p {
  margin: 0;
}

.detail__skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.detail__skeleton-line,
.detail__skeleton-map {
  height: 28px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-muted);
}

.detail__skeleton-line--short {
  width: 40%;
}

.detail__skeleton-map {
  height: 280px;
  border-radius: var(--radius-lg);
}

@media (max-width: 767.98px) {
  .detail {
    gap: 14px;
    padding: 0;
    border: none;
    background: transparent;
  }

  .detail__bar {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    margin: calc(-1 * var(--space-2)) calc(-1 * var(--space-3)) 0;
  }

  .detail__back,
  .detail__trash {
    display: flex;
    align-items: center;
    justify-content: center;
    width: var(--touch-min);
    height: var(--touch-min);
    border: none;
    background: transparent;
    color: var(--color-ink);
    cursor: pointer;
  }

  .detail__trash {
    color: var(--color-danger);
  }

  .detail__trash:disabled {
    opacity: 0.4;
  }

  .detail__bar-title {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    font-size: var(--fs-lg);
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .detail__head {
    display: contents;
  }

  .detail__heading,
  .detail__delete,
  .detail__tiles,
  .detail__table-wrap {
    display: none;
  }

  .detail__map {
    order: 1;
    height: 210px;
    border-radius: var(--radius-xl);
  }

  .detail__summary {
    display: flex;
    flex-direction: column;
    gap: 2px;
    order: 2;
  }

  .detail__hectares {
    margin: 0;
    font-size: var(--fs-metric);
    font-weight: 800;
    letter-spacing: -0.03em;
    line-height: 1.05;
  }

  .detail__line {
    margin: 0;
    color: var(--color-muted);
    font-size: var(--fs-control);
  }

  .detail__actions {
    display: grid;
    grid-template-areas: 'edit rename';
    grid-template-columns: 1fr 1fr;
    order: 3;
  }

  .detail__pending {
    order: 4;
  }

  .detail__edit {
    grid-area: edit;
  }

  .detail__rename {
    grid-area: rename;
  }

  .detail__report {
    display: none;
  }

  .detail__edit,
  .detail__rename {
    height: var(--control-h);
    font-weight: 700;
  }

  .detail__history {
    order: 5;
  }

  .detail__versions {
    display: block;
    margin: 0;
    padding: 0;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    background: var(--color-surface);
    list-style: none;
  }

  .detail__version {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-3) 14px;
    font-size: var(--fs-control);
  }

  .detail__version + .detail__version {
    border-top: 1px solid var(--color-border);
  }

  .detail__version-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .detail__version-name {
    font-weight: 600;
  }

  .detail__version--current .detail__version-name,
  .detail__version--current .detail__version-area {
    font-weight: 700;
  }

  .detail__version-meta {
    color: var(--color-muted);
    font-size: var(--fs-sm);
  }

  .detail__version-area {
    font-variant-numeric: tabular-nums;
  }

  .detail__button.detail__report-mobile {
    display: flex;
    order: 6;
    height: var(--control-h-lg);
    margin-top: var(--space-2);
    border-radius: var(--radius-lg);
    font-size: var(--fs-md);
  }
}
</style>
