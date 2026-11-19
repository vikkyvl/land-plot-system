import { mount } from '@vue/test-utils'
import L from 'leaflet'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { Coordinate } from '../../domain/coordinate'
import type { MapLayer } from '../../domain/mapLayer'
import type { Obstruction } from '../../domain/survey'

import PlotMap from './PlotMap.vue'

const layer: MapLayer = { id: 'map', tileUrl: 'http://tiles.test/{z}/{x}/{y}.png', attribution: 'tiles', maxZoom: 19 }
const points = [
  { latitude: 50.45, longitude: 30.52 },
  { latitude: 50.451, longitude: 30.523 },
  { latitude: 50.449, longitude: 30.524 },
]

afterEach(() => {
  vi.restoreAllMocks()
})

function mountMap(obstructions: readonly Obstruction[] = []) {
  const markers: L.Marker[] = []
  const lines: L.Polyline[] = []
  const createMarker = L.marker
  const createLine = L.polyline
  vi.spyOn(L, 'marker').mockImplementation((...args: Parameters<typeof L.marker>) => {
    const marker = createMarker(...args)
    markers.push(marker)

    return marker
  })
  vi.spyOn(L, 'polyline').mockImplementation((...args: Parameters<typeof L.polyline>) => {
    const line = createLine(...args)
    lines.push(line)

    return line
  })
  const wrapper = mount(PlotMap, {
    attachTo: document.body,
    props: { points, layer, selfIntersection: null, flaggedVertices: [], showNumbers: false, place: null, obstructions },
  })
  const vertices = markers.filter((marker) => marker.options.draggable)
  const midpoints = markers.filter((marker) => !marker.options.draggable && marker.options.interactive !== false)
  const labels = markers.filter((marker) => marker.options.interactive === false)
  const sides = lines.filter((line) => line.options.opacity === 0)
  const dashed = lines.filter((line) => typeof line.options.dashArray === 'string')

  return { wrapper, vertices, midpoints, labels, sides, dashed }
}

describe('PlotMap', () => {
  it('reports a moved vertex only when the drag ends', () => {
    const { wrapper, vertices } = mountMap()
    const vertex = vertices[1] as L.Marker

    vertex.setLatLng([50.4515, 30.5235])
    vertex.fire('drag')
    vertex.fire('drag')
    expect(wrapper.emitted('move')).toBeUndefined()

    vertex.fire('dragend')
    expect(wrapper.emitted('move')).toEqual([
      [1, { latitude: 50.4515, longitude: 30.5235 }, { zoom: expect.any(Number), pointer: 'mouse' }],
    ])
    wrapper.unmount()
  })

  it('opens the side menu from a midpoint or from the side itself', () => {
    const { wrapper, vertices, midpoints, sides } = mountMap()

    expect(vertices).toHaveLength(3)
    expect(midpoints).toHaveLength(3)
    expect(sides).toHaveLength(3)
    midpoints[0]?.fire('click', { originalEvent: { clientX: 120, clientY: 80 } })
    sides[2]?.fire('click', { originalEvent: { clientX: 10, clientY: 20 } })

    const [first, second] = wrapper.emitted('side') as [number, Coordinate, unknown, { x: number; y: number }][]
    expect(first?.[0]).toBe(0)
    expect(first?.[1].latitude).toBeCloseTo(50.4505, 9)
    expect(first?.[1].longitude).toBeCloseTo(30.5215, 9)
    expect(first?.[2]).toEqual({ zoom: expect.any(Number), pointer: 'mouse' })
    expect(first?.[3]).toEqual({ x: 120, y: 80 })
    expect(second?.[0]).toBe(2)
    expect(wrapper.emitted('add')).toBeUndefined()
    wrapper.unmount()
  })

  it('draws a marked side dashed with the reason next to it', () => {
    const { wrapper, labels, dashed } = mountMap([{ side: 1, reason: 'trees' }])

    expect(dashed).toHaveLength(1)
    expect(dashed[0]?.getLatLngs()).toEqual([L.latLng(50.451, 30.523), L.latLng(50.449, 30.524)])
    expect(labels).toHaveLength(1)
    expect(String((labels[0]?.options.icon as L.DivIcon).options.html)).toContain('дерева')
    wrapper.unmount()
  })

  it('puts swapped and new points in place at once and draws the crossing statically', async () => {
    const { wrapper } = mountMap()
    const [first, second, third] = points as [Coordinate, Coordinate, Coordinate]

    await wrapper.setProps({ points: [first, third, second, { latitude: 50.4495, longitude: 30.5205 }] })
    await wrapper.setProps({
      selfIntersection: { kind: 'self_intersection', segments: [[1, 2], [3, 0]], swap: [2, 3], point: null },
    })

    const vertices = [...document.querySelectorAll<HTMLElement>('.plot-vertex')]
    expect(vertices).toHaveLength(4)
    expect(vertices.every((vertex) => vertex.style.translate === '' && !vertex.className.includes('--new'))).toBe(true)
    expect(document.querySelectorAll('[class*="crossing"]')).toHaveLength(0)
    wrapper.unmount()
  })
})
