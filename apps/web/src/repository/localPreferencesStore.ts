import type { MapLayerId } from '../domain/mapLayer'
import type { PreferencesStore } from '../port/preferencesStore'

const mapLayerKey = 'arqetra.mapLayer'
const mapLayers: readonly MapLayerId[] = ['map', 'satellite']

export class LocalPreferencesStore implements PreferencesStore {
  private readonly storage: Storage

  constructor(storage: Storage) {
    this.storage = storage
  }

  mapLayer(): MapLayerId | null {
    const value = this.read(mapLayerKey)

    return mapLayers.find((layer) => layer === value) ?? null
  }

  saveMapLayer(layer: MapLayerId): void {
    try {
      this.storage.setItem(mapLayerKey, layer)
    } catch {
      return
    }
  }

  private read(key: string): string | null {
    try {
      return this.storage.getItem(key)
    } catch {
      return null
    }
  }
}
