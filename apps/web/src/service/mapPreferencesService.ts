import type { MapLayerId } from '../domain/mapLayer'
import type { PreferencesStore } from '../port/preferencesStore'

export const defaultMapLayer: MapLayerId = 'map'

export class MapPreferencesService {
  private readonly preferences: PreferencesStore

  constructor(preferences: PreferencesStore) {
    this.preferences = preferences
  }

  layer(): MapLayerId {
    return this.preferences.mapLayer() ?? defaultMapLayer
  }

  chooseLayer(layer: MapLayerId): void {
    this.preferences.saveMapLayer(layer)
  }
}
