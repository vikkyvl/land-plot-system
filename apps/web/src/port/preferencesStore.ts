import type { MapLayerId } from '../domain/mapLayer'

export interface PreferencesStore {
  mapLayer(): MapLayerId | null
  saveMapLayer(layer: MapLayerId): void
}
