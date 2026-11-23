export type MapLayerId = 'map' | 'satellite'

export interface MapLayer {
  readonly id: MapLayerId
  readonly tileUrl: string
  readonly attribution: string
  readonly maxZoom: number
}

export type MapLayers = Readonly<Record<MapLayerId, MapLayer>>
