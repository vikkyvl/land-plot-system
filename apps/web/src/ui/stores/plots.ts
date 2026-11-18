import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'

import type { Plot } from '../../domain/plot'
import { usePlotService } from '../composables/services'

export const usePlotsStore = defineStore('plots', () => {
  const service = usePlotService()
  const plots = shallowRef<readonly Plot[]>([])
  const loading = ref(false)
  const loaded = ref(false)
  const query = ref('')

  const visible = computed(() => {
    const needle = query.value.trim().toLocaleLowerCase('uk-UA')

    return needle === '' ? plots.value : plots.value.filter((plot) => plot.name.toLocaleLowerCase('uk-UA').includes(needle))
  })
  const totalHectares = computed(() => plots.value.reduce((sum, plot) => sum + plot.measurement.area.hectares, 0))

  async function load(): Promise<void> {
    loading.value = true
    try {
      plots.value = await service.list()
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
  }

  return { plots, loading, loaded, query, visible, totalHectares, load, replace, forget }
})
