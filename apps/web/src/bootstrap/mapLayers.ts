import type { AppConfig } from '../config/env'
import type { MapLayers } from '../domain/mapLayer'

const mapAttribution = '© <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
const satelliteAttribution = 'Знімки © Esri, Maxar, Earthstar Geographics'
const mapMaxZoom = 19
const satelliteMaxZoom = 19

export function mapLayersFrom(config: AppConfig): MapLayers {
  return {
    map: { id: 'map', tileUrl: config.mapTilesUrl, attribution: mapAttribution, maxZoom: mapMaxZoom },
    satellite: {
      id: 'satellite',
      tileUrl: config.satelliteTilesUrl,
      attribution: satelliteAttribution,
      maxZoom: satelliteMaxZoom,
    },
  }
}
