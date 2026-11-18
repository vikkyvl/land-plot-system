import { mount } from '@vue/test-utils'
import L from 'leaflet'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { MapLayer } from '../../domain/mapLayer'

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

function mountMap() {
  const markers: L.Marker[] = []
  const create = L.marker
  vi.spyOn(L, 'marker').mockImplementation((...args: Parameters<typeof L.marker>) => {
    const marker = create(...args)
    markers.push(marker)

    return marker
  })
  const wrapper = mount(PlotMap, {
    attachTo: document.body,
    props: { points, layer, selfIntersection: null, flaggedVertices: [], showNumbers: false, place: null },
  })
  const vertices = markers.filter((marker) => marker.options.draggable)
  const midpoints = markers.filter((marker) => !marker.options.draggable)

  return { wrapper, vertices, midpoints }
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

  it('draws a draggable vertex per point and a midpoint per side that inserts a vertex', () => {
    const { wrapper, vertices, midpoints } = mountMap()

    expect(vertices).toHaveLength(3)
    expect(midpoints).toHaveLength(3)
    midpoints[0]?.fire('click')
    const [afterIndex, point] = wrapper.emitted('insert')?.[0] as [number, { latitude: number; longitude: number }]
    expect(afterIndex).toBe(0)
    expect(point.latitude).toBeCloseTo(50.4505, 9)
    expect(point.longitude).toBeCloseTo(30.5215, 9)
    wrapper.unmount()
  })
})
