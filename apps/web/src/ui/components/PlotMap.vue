<script setup lang="ts">
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'

import type { Coordinate } from '../../domain/coordinate'
import type { MapLayer } from '../../domain/mapLayer'
import type { Place } from '../../domain/place'
import type { SelfIntersectionDetails } from '../../domain/problem'
import { obstructionOn, type MapGesture, type Obstruction, type PointerKind } from '../../domain/survey'
import { t } from '../i18n'

const props = defineProps<{
  points: readonly Coordinate[]
  layer: MapLayer
  selfIntersection: SelfIntersectionDetails | null
  flaggedVertices: readonly number[]
  showNumbers: boolean
  place: Place | null
  obstructions?: readonly Obstruction[]
}>()

const emit = defineEmits<{
  add: [point: Coordinate, gesture: MapGesture]
  move: [index: number, point: Coordinate, gesture: MapGesture]
  side: [side: number, middle: Coordinate, gesture: MapGesture, anchor: { x: number; y: number }]
}>()

const kyiv: L.LatLngTuple = [50.4501, 30.5234]
const startZoom = 15
const placeZoom = 16
const fitPadding: L.PointTuple = [48, 48]
const vertexSize = 22
const midpointSize = 14
const sideHitWeight = 24
const obstructedDash = '8 8'

const container = ref<HTMLDivElement | null>(null)
const locating = ref(false)
const map = shallowRef<L.Map | null>(null)
let tiles: L.TileLayer | null = null
const shapes = L.layerGroup()
const handles = L.layerGroup()
let lastPointer: PointerKind = 'mouse'

function rememberPointer(event: PointerEvent): void {
  lastPointer = event.pointerType === 'touch' || event.pointerType === 'pen' ? event.pointerType : 'mouse'
}

function gesture(): MapGesture {
  return { zoom: map.value?.getZoom() ?? startZoom, pointer: lastPointer }
}

function css(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}

function toCoordinate(latLng: L.LatLng): Coordinate {
  return { latitude: latLng.lat, longitude: latLng.lng }
}

function toLatLng(point: Coordinate): L.LatLngTuple {
  return [point.latitude, point.longitude]
}

function showLayer(layer: MapLayer): void {
  if (map.value === null) {
    return
  }
  tiles?.remove()
  tiles = L.tileLayer(layer.tileUrl, { attribution: layer.attribution, maxZoom: layer.maxZoom }).addTo(map.value)
  map.value.setMaxZoom(layer.maxZoom)
}

function crossing(a: Coordinate, b: Coordinate, c: Coordinate, d: Coordinate): L.LatLngTuple | null {
  const denominator =
    (a.longitude - b.longitude) * (c.latitude - d.latitude) - (a.latitude - b.latitude) * (c.longitude - d.longitude)
  if (denominator === 0) {
    return null
  }
  const t =
    ((a.longitude - c.longitude) * (c.latitude - d.latitude) - (a.latitude - c.latitude) * (c.longitude - d.longitude)) /
    denominator

  return [a.latitude + t * (b.latitude - a.latitude), a.longitude + t * (b.longitude - a.longitude)]
}

function anchorOf(event: L.LeafletMouseEvent): { x: number; y: number } {
  const original = event.originalEvent as MouseEvent | undefined

  return { x: original?.clientX ?? 0, y: original?.clientY ?? 0 }
}

function middleOf(from: L.LatLngTuple, to: L.LatLngTuple): L.LatLngTuple {
  return [(from[0] + to[0]) / 2, (from[1] + to[1]) / 2]
}

function openSide(index: number, middle: L.LatLngTuple, event: L.LeafletMouseEvent): void {
  emit('side', index, { latitude: middle[0], longitude: middle[1] }, gesture(), anchorOf(event))
}

function drawSides(latLngs: L.LatLngTuple[], stroke: string): void {
  latLngs.forEach((from, index) => {
    const to = latLngs[(index + 1) % latLngs.length]
    if (to === undefined) {
      return
    }
    const reason = obstructionOn({ obstructions: props.obstructions ?? [] }, index)
    L.polyline([from, to], {
      color: stroke,
      weight: 3,
      lineCap: 'round',
      dashArray: reason === null ? undefined : obstructedDash,
      interactive: false,
    }).addTo(shapes)
    L.polyline([from, to], { weight: sideHitWeight, opacity: 0, bubblingMouseEvents: false })
      .on('click', (event: L.LeafletMouseEvent) => openSide(index, middleOf(from, to), event))
      .addTo(shapes)
    if (reason !== null) {
      L.marker(middleOf(from, to), {
        icon: L.divIcon({
          className: 'plot-side-label',
          html: `<span class="plot-side-label__text">${t(`obstructions.${reason}`)}</span>`,
          iconSize: [0, 0],
        }),
        interactive: false,
        keyboard: false,
      }).addTo(shapes)
    }
  })
}

function drawShapes(latLngs: L.LatLngTuple[]): void {
  shapes.clearLayers()
  const stroke = css('--color-plot-stroke')
  const intersection = css('--color-intersection')
  const broken = props.selfIntersection !== null

  if (latLngs.length >= 3) {
    L.polygon(latLngs, {
      stroke: false,
      fillColor: broken ? intersection : stroke,
      fillOpacity: broken ? 0.12 : 0.2,
      interactive: false,
    }).addTo(shapes)
    drawSides(latLngs, stroke)
  } else if (latLngs.length === 2) {
    L.polyline(latLngs, { color: stroke, weight: 3, interactive: false }).addTo(shapes)
  }

  if (props.selfIntersection === null) {
    return
  }

  const segments = props.selfIntersection.segments.map(([from, to]) => [props.points[from], props.points[to]])
  for (const [from, to] of segments) {
    if (from && to) {
      L.polyline([toLatLng(from), toLatLng(to)], {
        color: intersection,
        weight: 5,
        lineCap: 'round',
        interactive: false,
      }).addTo(shapes)
    }
  }

  const [[a, b], [c, d]] = segments as [[Coordinate?, Coordinate?], [Coordinate?, Coordinate?]]
  const point = a && b && c && d ? crossing(a, b, c, d) : null
  if (point !== null) {
    L.circleMarker(point, {
      radius: 16,
      color: intersection,
      weight: 2.5,
      fillColor: intersection,
      fillOpacity: 0.18,
      interactive: false,
    }).addTo(shapes)
  }
}

function vertexIcon(index: number): L.DivIcon {
  const classes = ['plot-vertex']
  if (index === props.points.length - 1) {
    classes.push('plot-vertex--last')
  }
  if (props.flaggedVertices.includes(index)) {
    classes.push('plot-vertex--flagged')
  }
  const label = props.showNumbers ? `<span class="plot-vertex__label">${index + 1}</span>` : ''

  return L.divIcon({
    className: classes.join(' '),
    html: label,
    iconSize: [vertexSize, vertexSize],
    iconAnchor: [vertexSize / 2, vertexSize / 2],
  })
}

function draw(): void {
  if (map.value === null) {
    return
  }
  const latLngs = props.points.map(toLatLng)
  drawShapes(latLngs)
  handles.clearLayers()

  if (latLngs.length >= 2) {
    latLngs.forEach((from, index) => {
      const to = latLngs[(index + 1) % latLngs.length]
      const crossingSide = props.selfIntersection?.segments.some(([from]) => from === index) ?? false
      if (to === undefined || (latLngs.length === 2 && index === 1) || crossingSide) {
        return
      }
      const middle = middleOf(from, to)
      L.marker(middle, {
        icon: L.divIcon({
          className: 'plot-midpoint',
          iconSize: [midpointSize, midpointSize],
          iconAnchor: [midpointSize / 2, midpointSize / 2],
        }),
        keyboard: false,
        title: t('sides.side', { from: index + 1, to: ((index + 1) % latLngs.length) + 1 }),
      })
        .on('click', (event: L.LeafletMouseEvent) => openSide(index, middle, event))
        .addTo(handles)
    })
  }

  latLngs.forEach((latLng, index) => {
    const vertex = L.marker(latLng, {
      icon: vertexIcon(index),
      draggable: true,
      autoPan: true,
      title: t('map.point', { n: index + 1 }),
      alt: t('map.point', { n: index + 1 }),
    })
    vertex.on('drag', () => {
      const moved = [...latLngs]
      const position = vertex.getLatLng()
      moved[index] = [position.lat, position.lng]
      drawShapes(moved)
    })
    vertex.on('dragend', () => emit('move', index, toCoordinate(vertex.getLatLng()), gesture()))
    vertex.addTo(handles)
  })
}

function fitPoints(): void {
  if (map.value !== null && props.points.length >= 2) {
    map.value.fitBounds(L.latLngBounds(props.points.map(toLatLng)), { padding: fitPadding, maxZoom: placeZoom + 1 })
  } else if (map.value !== null && props.points[0] !== undefined) {
    map.value.setView(toLatLng(props.points[0]), placeZoom)
  }
}

function zoomIn(): void {
  map.value?.zoomIn()
}

function zoomOut(): void {
  map.value?.zoomOut()
}

function locate(): void {
  if (!('geolocation' in navigator)) {
    return
  }
  locating.value = true
  navigator.geolocation.getCurrentPosition(
    (position) => {
      locating.value = false
      map.value?.setView([position.coords.latitude, position.coords.longitude], placeZoom)
    },
    () => {
      locating.value = false
    },
    { enableHighAccuracy: true, timeout: 10_000 },
  )
}

onMounted(() => {
  if (container.value === null) {
    return
  }
  map.value = L.map(container.value, { zoomControl: false, attributionControl: false, fadeAnimation: false }).setView(
    kyiv,
    startZoom,
  )
  L.control.attribution({ position: 'bottomleft', prefix: false }).addTo(map.value)
  showLayer(props.layer)
  shapes.addTo(map.value)
  handles.addTo(map.value)
  container.value.addEventListener('pointerdown', rememberPointer, { capture: true })
  map.value.on('click', (event: L.LeafletMouseEvent) => emit('add', toCoordinate(event.latlng), gesture()))
  draw()
  fitPoints()
})

onBeforeUnmount(() => {
  container.value?.removeEventListener('pointerdown', rememberPointer, { capture: true })
  map.value?.remove()
  map.value = null
})

watch(
  () => [props.points, props.selfIntersection, props.flaggedVertices, props.showNumbers, props.obstructions],
  () => draw(),
)
watch(
  () => props.points,
  (points, previous) => {
    if (previous.length === 0 && points.length > 0) {
      fitPoints()
    }
  },
)
watch(
  () => props.layer,
  (layer) => showLayer(layer),
)
watch(
  () => props.place,
  (place) => {
    if (map.value === null || place === null) {
      return
    }
    if (place.bounds !== null) {
      map.value.fitBounds([
        [place.bounds.south, place.bounds.west],
        [place.bounds.north, place.bounds.east],
      ])
    } else {
      map.value.setView(toLatLng(place.coordinate), placeZoom)
    }
  },
)

defineExpose({ fitPoints })
</script>

<template>
  <div class="plot-map">
    <div
      ref="container"
      class="plot-map__canvas"
      role="application"
      :aria-label="$t('map.label')"
    />
    <div class="plot-map__controls">
      <button
        type="button"
        class="plot-map__control"
        :aria-label="$t('map.zoomIn')"
        @click="zoomIn"
      >
        +
      </button>
      <button
        type="button"
        class="plot-map__control"
        :aria-label="$t('map.zoomOut')"
        @click="zoomOut"
      >
        −
      </button>
      <button
        type="button"
        class="plot-map__control"
        :aria-label="$t('map.locate')"
        :aria-busy="locating"
        @click="locate"
      >
        <svg
          width="22"
          height="22"
          viewBox="0 0 24 24"
          aria-hidden="true"
        >
          <circle
            cx="12"
            cy="12"
            r="6.5"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          />
          <circle
            cx="12"
            cy="12"
            r="2.6"
            fill="currentColor"
          />
          <path
            d="M12 2v3.5M12 18.5V22M2 12h3.5M18.5 12H22"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          />
        </svg>
      </button>
    </div>
    <slot />
  </div>
</template>

<style scoped>
.plot-map {
  position: relative;
  overflow: hidden;
  background: var(--color-map-bg);
}

.plot-map__canvas {
  position: absolute;
  inset: 0;
  background: var(--color-map-bg);
  font-family: var(--font-sans);
}

.plot-map__controls {
  position: absolute;
  z-index: 500;
  top: var(--map-controls-top, var(--space-4));
  right: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.plot-map__control {
  display: flex;
  align-items: center;
  justify-content: center;
  width: var(--touch-min);
  height: var(--touch-min);
  border: none;
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  box-shadow: var(--shadow-control);
  color: var(--color-ink);
  font: inherit;
  font-size: 22px;
  cursor: pointer;
}

.plot-map :deep(.leaflet-bottom) {
  bottom: var(--map-bottom-inset, 0px);
}

.plot-map :deep(.leaflet-control-attribution) {
  margin: 0 0 var(--space-2) var(--space-3);
  padding: 0;
  background: transparent;
  color: var(--color-ink-3);
  font-size: var(--fs-xs);
}

.plot-map :deep(.leaflet-control-attribution a) {
  color: inherit;
}

.plot-map :deep(.plot-vertex) {
  border: 3px solid var(--color-plot-stroke);
  border-radius: 50%;
  background: var(--color-surface);
  cursor: grab;
}

.plot-map :deep(.plot-vertex--last) {
  border-color: var(--color-surface);
  background: var(--color-plot-stroke);
}

.plot-map :deep(.plot-vertex--flagged) {
  border-color: var(--color-intersection);
}

.plot-map :deep(.leaflet-marker-dragging.plot-vertex),
.plot-map :deep(.plot-vertex:active) {
  border-color: var(--color-ink);
  background: var(--color-vertex-active);
  cursor: grabbing;
}

.plot-map :deep(.plot-vertex__label) {
  position: absolute;
  top: -26px;
  left: 50%;
  transform: translateX(-50%);
  color: var(--color-ink);
  font-size: var(--fs-md);
  font-weight: 700;
  text-shadow:
    0 0 3px var(--color-surface),
    0 0 3px var(--color-surface);
}

.plot-map :deep(.plot-midpoint) {
  border: 2px solid var(--color-plot-stroke);
  border-radius: 50%;
  background: var(--color-surface);
  opacity: 0.75;
  cursor: copy;
}

.plot-map :deep(.plot-side-label__text) {
  position: absolute;
  padding: 2px var(--space-2);
  border-radius: var(--radius-xs);
  background: var(--color-surface);
  box-shadow: var(--shadow-control);
  color: var(--color-ink);
  font-size: var(--fs-xs);
  font-weight: 700;
  white-space: nowrap;
  transform: translate(-50%, -150%);
}

.plot-map :deep(.plot-vertex)::before,
.plot-map :deep(.plot-midpoint)::before {
  position: absolute;
  inset: calc((var(--touch-min) - 100%) / -2);
  content: '';
}
</style>
