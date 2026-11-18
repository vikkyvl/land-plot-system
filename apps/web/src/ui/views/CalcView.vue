<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { useRouter } from 'vue-router'

import type { Coordinate } from '../../domain/coordinate'
import type { MapLayerId } from '../../domain/mapLayer'
import type { Place } from '../../domain/place'
import { hasScope, Scope } from '../../domain/profile'
import ChecksCard from '../components/ChecksCard.vue'
import ErrorPanel from '../components/ErrorPanel.vue'
import LayerSwitch from '../components/LayerSwitch.vue'
import OutagePanel from '../components/OutagePanel.vue'
import PlaceSearch from '../components/PlaceSearch.vue'
import PlotMap from '../components/PlotMap.vue'
import PointsCard from '../components/PointsCard.vue'
import StalledNotice from '../components/StalledNotice.vue'
import { errorText } from '../composables/errorText'
import {
  formatAres,
  formatHectares,
  formatMeters,
  formatPointCount,
  formatSquareMeters,
} from '../composables/format'
import { useMapLayers, useSettingsService } from '../composables/services'
import { useFollowedPlots } from '../composables/useFollowedPlots'
import { useSignInRedirect } from '../composables/useSignInRedirect'
import { t } from '../i18n'
import { useCalculatorStore } from '../stores/calculator'
import { useConfirmationsStore } from '../stores/confirmations'
import { useSessionStore } from '../stores/session'
import { useToastStore } from '../stores/toast'

const props = defineProps<{ plotId?: string }>()

const calculator = useCalculatorStore()
const router = useRouter()
const toasts = useToastStore()
const confirmations = useConfirmationsStore()
const followed = useFollowedPlots()
const session = useSessionStore()
const signIn = useSignInRedirect()
const layers = useMapLayers()
const preferences = useSettingsService()

const {
  points,
  name,
  preview,
  outage,
  errorDismissed,
  saving,
  saveError,
  savedPlot,
  editingId,
  loadError,
  enoughPoints,
  problems,
  measurement,
  updating,
  selfIntersection,
  flaggedVertices,
  canSave,
} = storeToRefs(calculator)
const { signedIn, profile } = storeToRefs(session)

const layerId = ref<MapLayerId>(preferences.layer())
const place = shallowRef<Place | null>(null)
const nameId = 'plot-name'

const layer = computed(() => layers[layerId.value])
const mainProblem = computed(() => problems.value[0] ?? null)
const showProblem = computed(() => mainProblem.value !== null && !errorDismissed.value && outage.value === null)
const canWrite = computed(() => profile.value === null || hasScope(profile.value, Scope.PlotsWrite))
const lastArea = computed(() => (preview.value?.valid ? preview.value.measurement : null))
const chip = computed(() =>
  updating.value && lastArea.value !== null
    ? t('calc.updatingChip', { value: formatHectares(lastArea.value.area.hectares) })
    : null,
)
const status = computed(() => {
  if (updating.value) {
    return t('calc.updating')
  }

  return measurement.value !== null ? t('calc.serverResult') : formatPointCount(points.value.length)
})

watch(layerId, (value) => preferences.chooseLayer(value))

function add(point: Coordinate): void {
  calculator.addPoint(point)
}

async function save(): Promise<void> {
  const outcome = await calculator.save(signedIn.value)
  if (outcome === 'sign-in') {
    await signIn(editingId.value === null ? '/' : undefined)
  } else if (outcome === 'updated' && editingId.value !== null) {
    confirmations.expect(editingId.value, {
      intent: 'update',
      name: name.value.trim(),
      expected: { name: name.value, points: points.value },
    })
    toasts.show(t('calc.boundarySent'))
    await router.push(`/plots/${editingId.value}`)
  }
}

const savedPhase = computed(() =>
  savedPlot.value === null ? null : (confirmations.entries[savedPlot.value.id]?.phase ?? 'pending'),
)

watch(
  savedPlot,
  (plot) => {
    followed.releaseAll()
    const phase = plot === null ? null : confirmations.entries[plot.id]?.phase
    if (plot !== null && editingId.value === null && (phase === undefined || phase === 'pending')) {
      confirmations.expect(plot.id, { intent: 'create', name: plot.name })
      followed.follow(plot.id)
    }
  },
  { immediate: true },
)

watch(
  () => props.plotId,
  (plotId) => (plotId === undefined ? calculator.start() : calculator.startEditing(plotId)),
  { immediate: true },
)
onBeforeUnmount(() => calculator.stop())
</script>

<template>
  <main class="calc">
    <section
      class="calc__map-column"
      :aria-label="$t('calc.mapRegion')"
    >
      <div class="calc__toolbar">
        <PlaceSearch
          class="calc__search"
          :placeholder="$t('calc.searchPlaceholder')"
          @select="place = $event"
        />
        <LayerSwitch
          v-model="layerId"
          class="calc__layers"
        />
      </div>
      <PlotMap
        class="calc__map"
        :points="points"
        :layer="layer"
        :self-intersection="showProblem ? selfIntersection : null"
        :flagged-vertices="flaggedVertices"
        :show-numbers="selfIntersection !== null"
        :place="place"
        @add="add"
        @move="calculator.movePoint"
        @insert="calculator.insertPoint"
      >
        <p
          v-if="chip"
          class="calc__chip"
          role="status"
        >
          {{ chip }}
        </p>
        <p
          v-else
          class="calc__hint"
        >
          {{ $t('calc.hint') }}
        </p>
      </PlotMap>
    </section>

    <aside
      class="calc__panel"
      :aria-label="$t('calc.panelRegion')"
    >
      <div
        v-if="editingId"
        class="calc__editing"
      >
        <span>{{ name ? $t('calc.editingNamed', { name }) : $t('calc.editing') }}</span>
        <RouterLink :to="`/plots/${editingId}`">
          {{ $t('calc.cancelEditing') }}
        </RouterLink>
      </div>
      <div
        v-if="loadError"
        class="calc__save-error"
        role="alert"
      >
        {{ errorText(loadError) }}
        <RouterLink to="/plots">
          {{ $t('app.toPlots') }}
        </RouterLink>
      </div>
      <OutagePanel
        v-if="outage"
        :outage="outage"
        @retry="calculator.retryPreview"
      />
      <ErrorPanel
        v-else-if="showProblem && mainProblem"
        :problem="mainProblem"
        @swap="calculator.swap"
        @dismiss="calculator.dismissError"
      />

      <section class="calc__card">
        <div class="calc__area">
          <div class="calc__area-head">
            <h1 class="calc__label">
              {{ $t('calc.area') }}
            </h1>
            <span
              class="calc__status"
              aria-live="polite"
            >{{ status }}</span>
          </div>
          <p
            class="calc__value"
            :class="{ 'calc__value--empty': measurement === null }"
          >
            {{ measurement ? formatHectares(measurement.area.hectares) : $t('units.noHectares') }}
          </p>
        </div>

        <dl
          v-if="measurement"
          class="calc__tiles"
        >
          <div class="calc__tile">
            <dt>{{ $t('units.squareMeters') }}</dt>
            <dd>{{ formatSquareMeters(measurement.area.squareMeters) }}</dd>
          </div>
          <div class="calc__tile">
            <dt>{{ $t('units.ares') }}</dt>
            <dd>{{ formatAres(measurement.area.ares) }}</dd>
          </div>
          <div class="calc__tile">
            <dt>{{ $t('units.perimeter') }}</dt>
            <dd>{{ formatMeters(measurement.perimeterMeters) }}</dd>
          </div>
        </dl>
        <p
          v-else-if="!enoughPoints"
          class="calc__muted"
        >
          <span class="calc__empty-web">{{ $t('calc.addPointsWeb') }}</span>
          <span class="calc__empty-mobile">{{ $t('calc.addPointsMobile') }}</span>
        </p>

        <div
          v-if="!problems.length && points.length"
          class="calc__field"
        >
          <label
            class="calc__field-label"
            :for="nameId"
          >{{ $t('calc.name') }}</label>
          <input
            :id="nameId"
            class="calc__input"
            type="text"
            maxlength="200"
            :placeholder="$t('calc.namePlaceholder')"
            :value="name"
            @input="calculator.rename(($event.target as HTMLInputElement).value)"
          >
        </div>

        <div class="calc__actions">
          <button
            v-if="canWrite"
            type="button"
            class="calc__button calc__button--primary calc__save"
            :disabled="!canSave"
            :aria-busy="saving"
            @click="save"
          >
            {{ saving ? $t('calc.saving') : $t('calc.save') }}
          </button>
          <button
            type="button"
            class="calc__button calc__undo"
            :disabled="points.length === 0"
            @click="calculator.undo"
          >
            {{ $t('calc.undo') }}
          </button>
          <button
            type="button"
            class="calc__button calc__clear"
            :disabled="points.length === 0"
            @click="calculator.clear"
          >
            {{ $t('calc.clear') }}
          </button>
        </div>

        <p
          v-if="problems.length"
          class="calc__muted"
        >
          {{ $t('calc.fixFirst') }}
        </p>
        <p
          v-if="!canWrite"
          class="calc__muted"
        >
          {{ $t('calc.noWriteScope') }}
        </p>
        <div
          v-if="saveError"
          class="calc__save-error"
          role="alert"
        >
          {{ errorText(saveError) }}
          <small v-if="saveError.requestId">{{ $t('app.requestId', { id: saveError.requestId }) }}</small>
        </div>
        <template v-if="savedPlot && !editingId">
          <p
            v-if="savedPhase === 'cancelled'"
            class="calc__save-error"
            role="alert"
          >
            {{ $t('calc.savedCancelled', { name: savedPlot.name }) }}
          </p>
          <StalledNotice
            v-else-if="savedPhase === 'stalled'"
            @retry="confirmations.retry(savedPlot.id)"
          />
          <p
            v-else
            class="calc__saved"
            role="status"
          >
            <template v-if="savedPhase === 'confirmed'">
              {{ $t('calc.savedConfirmed', { name: savedPlot.name }) }}
            </template>
            <template v-else>
              {{ $t('calc.savedPending', { name: savedPlot.name }) }}
            </template>
            <RouterLink :to="`/plots/${savedPlot.id}`">
              {{ $t('calc.openPlot') }}
            </RouterLink>
          </p>
        </template>
      </section>

      <ChecksCard
        v-if="problems.length"
        :problems="problems"
      />
      <PointsCard
        v-else-if="points.length"
        class="calc__points"
        :points="points"
        :flagged="flaggedVertices"
      />

      <p class="calc__note">
        {{ $t('calc.note') }}
      </p>
    </aside>
  </main>
</template>

<style scoped>
.calc {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-6);
  max-width: var(--container-max);
  margin: 0 auto;
  padding: var(--space-6);
}

.calc__map-column {
  display: flex;
  flex: 999 1 560px;
  flex-direction: column;
  gap: var(--space-3);
  min-width: 0;
}

.calc__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.calc__map {
  height: 640px;
  border: 1px solid var(--color-map-border);
  border-radius: var(--radius-2xl);
}

.calc__hint,
.calc__chip {
  position: absolute;
  z-index: 500;
  margin: 0;
  pointer-events: none;
}

.calc__hint {
  top: var(--space-4);
  left: var(--space-4);
  max-width: calc(100% - 96px);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: var(--color-surface);
  box-shadow: var(--shadow-float);
  color: var(--color-ink-2);
  font-size: var(--fs-control);
}

.calc__empty-mobile {
  display: none;
}

.calc__chip {
  bottom: 34px;
  left: var(--space-4);
  padding: 7px 11px;
  border-radius: var(--radius-xs);
  background: var(--color-chip-dark);
  color: var(--color-on-primary);
  font-size: var(--fs-sm);
  font-weight: 600;
}

.calc__panel {
  display: flex;
  flex: 1 1 340px;
  flex-direction: column;
  gap: var(--space-4);
}

.calc__card {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: var(--space-6);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-2xl);
  background: var(--color-surface);
}

.calc__area {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.calc__area-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-2);
}

.calc__label {
  margin: 0;
  color: var(--color-muted);
  font-size: var(--fs-sm);
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.calc__status {
  color: var(--color-muted);
  font-size: var(--fs-sm);
}

.calc__value {
  margin: 0;
  font-size: var(--fs-metric-2xl);
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1.05;
  font-variant-numeric: tabular-nums;
}

.calc__value--empty {
  color: var(--color-ink-disabled);
}

.calc__tiles {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--space-2);
  margin: 0;
}

.calc__tile {
  padding: 10px var(--space-3);
  border-radius: var(--radius-md);
  background: var(--color-surface-muted);
}

.calc__tile dt {
  color: var(--color-muted);
  font-size: var(--fs-xs);
}

.calc__tile dd {
  margin: 0;
  font-size: var(--fs-lg);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.calc__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.calc__field-label {
  font-size: var(--fs-control);
  font-weight: 600;
}

.calc__input {
  height: var(--control-h);
  padding: 0 14px;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-base);
}

.calc__input:focus-visible {
  border-color: var(--color-primary);
  outline: none;
}

.calc__actions {
  display: grid;
  grid-template-areas:
    'save save'
    'undo clear';
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.calc__button {
  min-height: var(--touch-min);
  padding: 0 var(--space-2);
  white-space: nowrap;
  border: 1.5px solid var(--color-border-strong);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-ink);
  font: inherit;
  font-size: var(--fs-control);
  font-weight: 600;
  cursor: pointer;
}

.calc__button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.calc__button--primary {
  height: var(--control-h-md);
  border: none;
  border-radius: var(--radius-lg);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: var(--fs-md);
  font-weight: 700;
}

.calc__button--primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
}

.calc__button--primary:disabled {
  background: var(--color-divider);
  color: var(--color-muted);
  opacity: 1;
}

.calc__save {
  grid-area: save;
}

.calc__undo {
  grid-area: undo;
}

.calc__clear {
  grid-area: clear;
}

.calc__editing {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-primary-soft);
  color: var(--color-primary-hover);
  font-size: var(--fs-control);
  font-weight: 600;
}

.calc__muted,
.calc__note {
  margin: 0;
  color: var(--color-muted);
  font-size: var(--fs-sm);
  line-height: 1.5;
}

.calc__save-error {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-danger-border);
  border-radius: var(--radius-md);
  background: var(--color-danger-bg);
  color: var(--color-danger-ink);
  font-size: var(--fs-control);
}

.calc__save-error small {
  font-size: var(--fs-2xs);
}

.calc__saved {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1) var(--space-2);
  margin: 0;
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-primary-soft);
  color: var(--color-primary-hover);
  font-size: var(--fs-control);
  font-weight: 600;
}

@media (max-width: 767.98px) {
  .calc {
    gap: 0;
    padding: 0;
  }

  .calc__map-column {
    position: relative;
    flex-basis: 100%;
    gap: 0;
  }

  .calc__toolbar {
    position: absolute;
    z-index: 600;
    top: 14px;
    right: 14px;
    left: 14px;
    gap: var(--space-2);
  }

  .calc__search {
    flex-basis: 100%;
  }

  .calc__search :deep(.place-search__input) {
    border: none;
    box-shadow: var(--shadow-on-imagery);
  }

  .calc__layers {
    box-shadow: var(--shadow-on-imagery);
  }

  .calc__layers :deep(.layer-switch__option) {
    height: var(--control-h-segment-sm);
    padding: 0 var(--space-3);
    font-size: var(--fs-sm);
  }

  .calc__map {
    --map-controls-top: 72px;
    --map-bottom-inset: 18px;

    height: min(60dvh, 520px);
    border: none;
    border-radius: 0;
  }

  .calc__hint,
  .calc__empty-web {
    display: none;
  }

  .calc__empty-mobile {
    display: inline;
  }

  .calc__chip {
    bottom: calc(var(--map-bottom-inset) + 30px);
  }

  .calc__panel {
    position: relative;
    z-index: 700;
    flex-basis: 100%;
    gap: 14px;
    margin-top: -18px;
    padding: 22px 20px calc(var(--space-5) + env(safe-area-inset-bottom));
    border-radius: var(--radius-sheet) var(--radius-sheet) 0 0;
    background: var(--color-surface);
    box-shadow: var(--shadow-sheet);
  }

  .calc__card {
    gap: var(--space-4);
    padding: 0;
    border: none;
    border-radius: 0;
  }

  .calc__value {
    font-size: var(--fs-metric-xl);
  }

  .calc__actions {
    grid-template-areas: 'undo save';
    grid-template-columns: 1fr 1.15fr;
  }

  .calc__undo,
  .calc__save {
    height: var(--control-h-md);
    border-radius: var(--radius-lg);
    font-size: var(--fs-base);
    font-weight: 700;
  }

  .calc__clear,
  .calc__points {
    display: none;
  }
}

.calc {
  padding-right: max(var(--space-6), env(safe-area-inset-right));
  padding-left: max(var(--space-6), env(safe-area-inset-left));
}

@media (max-width: 767.98px) {
  .calc {
    padding-right: 0;
    padding-left: 0;
  }

  .calc__panel {
    padding-right: max(20px, env(safe-area-inset-right));
    padding-left: max(20px, env(safe-area-inset-left));
  }
}
</style>
