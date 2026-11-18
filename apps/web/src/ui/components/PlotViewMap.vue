<script setup lang="ts">
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

import type { Coordinate } from '../../domain/coordinate'
import type { MapLayer } from '../../domain/mapLayer'

const props = defineProps<{ points: readonly Coordinate[]; layer: MapLayer }>()

const fitPadding: L.PointTuple = [28, 28]
const container = ref<HTMLDivElement | null>(null)
let map: L.Map | null = null
const shape = L.layerGroup()

function css(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function draw(): void {
  if (map === null) {
    return
  }
  shape.clearLayers()
  const latLngs = props.points.map((point): L.LatLngTuple => [point.latitude, point.longitude])
  if (latLngs.length < 3) {
    return
  }
  const stroke = css('--color-plot-stroke')
  L.polygon(latLngs, { color: stroke, weight: 3, fillColor: stroke, fillOpacity: 0.2, interactive: false }).addTo(
    shape,
  )
  map.fitBounds(L.latLngBounds(latLngs), { padding: fitPadding })
}

onMounted(() => {
  if (container.value === null) {
    return
  }
  map = L.map(container.value, {
    zoomControl: false,
    attributionControl: false,
    dragging: false,
    scrollWheelZoom: false,
    doubleClickZoom: false,
    touchZoom: false,
    boxZoom: false,
    keyboard: false,
    fadeAnimation: false,
  })
  L.control.attribution({ position: 'bottomleft', prefix: false }).addTo(map)
  L.tileLayer(props.layer.tileUrl, { attribution: props.layer.attribution, maxZoom: props.layer.maxZoom }).addTo(map)
  shape.addTo(map)
  draw()
})

onBeforeUnmount(() => {
  map?.remove()
  map = null
})

watch(() => props.points, draw)
</script>

<template>
  <div
    ref="container"
    class="plot-view-map"
    role="img"
    :aria-label="$t('map.viewLabel')"
  />
</template>

<style scoped>
.plot-view-map {
  overflow: hidden;
  border-radius: var(--radius-lg);
  background: var(--color-map-bg);
  font-family: var(--font-sans);
}

.plot-view-map :deep(.leaflet-control-attribution) {
  margin: 0 0 6px var(--space-3);
  padding: 0;
  background: transparent;
  color: var(--color-ink-3);
  font-size: var(--fs-2xs);
}
</style>
