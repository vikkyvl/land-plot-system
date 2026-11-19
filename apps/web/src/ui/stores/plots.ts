import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'

import { emptyPlotSummary, type Plot, type PlotSummary } from '../../domain/plot'
import { usePlotService } from '../composables/services'

export const usePlotsStore = defineStore('plots', () => {
  const service = usePlotService()
  const plots = shallowRef<readonly Plot[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const query = ref('')
  const summary = shallowRef<PlotSummary>(emptyPlotSummary)

  const visible = computed(() => {
    const needle = query.value.trim().toLocaleLowerCase('uk-UA')

    return needle === '' ? plots.value : plots.value.filter((plot) => plot.name.toLocaleLowerCase('uk-UA').includes(needle))
  })
  const totalHectares = computed(() => summary.value.totalHectares)
  const count = computed(() => summary.value.count)

  async function refreshSummary(): Promise<void> {
    try {
      summary.value = await service.summary()
    } catch {
      return
    }
  }

  async function load(): Promise<void> {
    loading.value = true
    try {
      const [list, totals] = await Promise.all([service.list(), service.summary()])
      plots.value = list
      summary.value = totals
      loaded.value = true
    } finally {
      loading.value = false
    }
  }

  function replace(plot: Plot): void {
    plots.value = plots.value.map((current) => (current.id === plot.id ? plot : current))
  }

  function forget(id: string): void {
    plots.value = plots.value.filter((plot) => plot.id !== id)
    void refreshSummary()
  }

  return { plots, loading, loaded, query, visible, totalHectares, count, load, replace, forget, refreshSummary }
})
