<script setup lang="ts">
import type { Climate } from '../../domain/surroundings'
import { formatCelsius, formatMillimeters, formatMonthDay, formatWhole } from '../composables/surroundingsText'

defineProps<{ climate: Climate }>()
</script>

<template>
  <div class="report-climate">
    <p class="report-climate__period">
      {{ $t('surroundings.climatePeriod', { from: climate.firstYear, to: climate.lastYear }) }}
    </p>
    <div class="report-climate__table-wrap">
      <table class="report-climate__table">
        <thead>
          <tr>
            <th scope="col">
              {{ $t('surroundings.season') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.mean') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.minimum') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.maximum') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.precipitation') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.frostDays') }}
            </th>
            <th scope="col">
              {{ $t('surroundings.heatDays') }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="season in climate.seasons"
            :key="season.season"
          >
            <th scope="row">
              {{ $t(`seasons.${season.season}`) }}
            </th>
            <td>{{ formatCelsius(season.meanCelsius) }}</td>
            <td>{{ formatCelsius(season.meanMinCelsius) }}</td>
            <td>{{ formatCelsius(season.meanMaxCelsius) }}</td>
            <td>{{ formatMillimeters(season.precipitationMillimeters) }}</td>
            <td>{{ formatWhole(season.frostDays) }}</td>
            <td>{{ formatWhole(season.heatDays) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <ul class="report-climate__cards">
      <li
        v-for="season in climate.seasons"
        :key="season.season"
        class="report-climate__card"
      >
        <span class="report-climate__card-season">{{ $t(`seasons.${season.season}`) }}</span>
        <span class="report-climate__card-mean">{{ formatCelsius(season.meanCelsius) }}</span>
        <span class="report-climate__card-line">{{
          $t('surroundings.seasonTemperatures', {
            min: formatCelsius(season.meanMinCelsius),
            max: formatCelsius(season.meanMaxCelsius),
          })
        }}</span>
        <span class="report-climate__card-line">{{ formatMillimeters(season.precipitationMillimeters) }}</span>
        <span class="report-climate__card-line">{{
          $t('surroundings.seasonDays', { frost: formatWhole(season.frostDays), heat: formatWhole(season.heatDays) })
        }}</span>
      </li>
    </ul>
    <ul class="report-climate__facts">
      <li>{{ $t('surroundings.lastSpringFrost', { date: formatMonthDay(climate.lastSpringFrost) }) }}</li>
      <li>{{ $t('surroundings.firstAutumnFrost', { date: formatMonthDay(climate.firstAutumnFrost) }) }}</li>
      <li>{{ $t('counts.growingDays', Math.round(climate.growingDays)) }}</li>
    </ul>
  </div>
</template>

<style scoped>
.report-climate {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.report-climate__period {
  margin: 0;
  color: var(--color-ink-3);
  font-size: var(--fs-sm);
}

.report-climate__table-wrap {
  overflow-x: auto;
}

.report-climate__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-control);
  font-variant-numeric: tabular-nums;
}

.report-climate__table th,
.report-climate__table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  text-align: right;
  white-space: nowrap;
}

.report-climate__table th[scope='row'],
.report-climate__table th:first-child {
  text-align: left;
}

.report-climate__table thead th {
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
  font-weight: 600;
}

.report-climate__cards {
  display: none;
}

.report-climate__facts {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding-left: var(--space-5);
  color: var(--color-ink-2);
  font-size: var(--fs-control);
}

@media (max-width: 767.98px) {
  .report-climate__table-wrap {
    display: none;
  }

  .report-climate__cards {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: var(--space-2);
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .report-climate__card {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 10px var(--space-3);
    border-radius: var(--radius-xs);
    background: var(--color-surface-muted);
    font-variant-numeric: tabular-nums;
  }

  .report-climate__card-season {
    color: var(--color-ink-3);
    font-size: var(--fs-xs);
    font-weight: 700;
  }

  .report-climate__card-mean {
    font-size: var(--fs-tile);
    font-weight: 800;
  }

  .report-climate__card-line {
    color: var(--color-ink-2);
    font-size: var(--fs-xs);
  }
}
</style>
