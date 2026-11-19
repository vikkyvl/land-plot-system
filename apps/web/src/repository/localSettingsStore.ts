import { isLocale, type Locale } from '../domain/locale'
import type { MapLayerId } from '../domain/mapLayer'
import type { SettingsStore } from '../port/settingsStore'

const mapLayerKey = 'arqetra.mapLayer'
const localeKey = 'arqetra.locale'
const mapLayers: readonly MapLayerId[] = ['map', 'satellite']

export class LocalSettingsStore implements SettingsStore {
  private readonly storage: Storage

  constructor(storage: Storage) {
    this.storage = storage
  }

  mapLayer(): MapLayerId | null {
    const value = this.read(mapLayerKey)

    return mapLayers.find((layer) => layer === value) ?? null
  }

  saveMapLayer(layer: MapLayerId): void {
    this.write(mapLayerKey, layer)
  }

  locale(): Locale | null {
    const value = this.read(localeKey)

    return isLocale(value) ? value : null
  }

  saveLocale(locale: Locale): void {
    this.write(localeKey, locale)
  }

  private read(key: string): string | null {
    try {
      return this.storage.getItem(key)
    } catch {
      return null
    }
  }

  private write(key: string, value: string): void {
    try {
      this.storage.setItem(key, value)
    } catch {
      return
    }
  }
}
