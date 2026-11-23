<script setup lang="ts">
import type { Terrain } from '../../domain/surroundings'
import { formatHectares } from '../composables/format'
import { formatDegrees, formatPercent, formatWhole } from '../composables/surroundingsText'

defineProps<{ terrain: Terrain }>()
</script>

<template>
  <div class="report-terrain">
    <p
      v-if="terrain.coarse"
      class="report-terrain__coarse"
    >
      <b>{{ $t('surroundings.coarse') }}</b>
      {{ $t('surroundings.coarseDetails') }}
    </p>
    <dl class="report-terrain__tiles">
      <div class="report-terrain__tile">
        <dt>{{ $t('surroundings.slope') }}</dt>
        <dd>{{ formatDegrees(terrain.slopeDegrees) }}</dd>
      </div>
      <div class="report-terrain__tile">
        <dt>{{ $t('surroundings.surfaceArea') }}</dt>
        <dd>
          {{ formatHectares(terrain.surfaceHectares) }}
          <small>{{ $t('surroundings.increase', { value: formatPercent(terrain.surfaceIncreasePercent) }) }}</small>
        </dd>
      </div>
      <div class="report-terrain__tile">
        <dt>{{ $t('surroundings.elevations') }}</dt>
        <dd>
          {{
            $t('surroundings.elevationRange', {
              from: formatWhole(terrain.minElevationMeters),
              to: formatWhole(terrain.maxElevationMeters),
            })
          }}
        </dd>
      </div>
      <div class="report-terrain__tile">
        <dt>{{ $t('surroundings.relief') }}</dt>
        <dd>{{ $t('units.meters', { value: formatWhole(terrain.reliefMeters) }) }}</dd>
      </div>
    </dl>
    <p class="report-terrain__note">
      {{ $t('surroundings.cadastral') }}
    </p>
  </div>
</template>

<style scoped>
.report-terrain {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.report-terrain__coarse {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-warning-border);
  border-radius: var(--radius-xs);
  background: var(--color-warning-bg);
  color: var(--color-warning-ink);
  font-size: var(--fs-control);
}

.report-terrain__tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: var(--space-2);
  margin: 0;
}

.report-terrain__tile {
  padding: 10px var(--space-3);
  border-radius: var(--radius-xs);
  background: var(--color-surface-muted);
}

.report-terrain__tile dt {
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
}

.report-terrain__tile dd {
  margin: var(--space-1) 0 0;
  font-size: var(--fs-tile);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.report-terrain__tile small {
  display: block;
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
  font-weight: 600;
}

.report-terrain__note {
  margin: 0;
  color: var(--color-ink-3);
  font-size: var(--fs-sm);
}

@media (max-width: 767.98px) {
  .report-terrain__tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
