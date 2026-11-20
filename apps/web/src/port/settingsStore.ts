import type { Locale } from '../domain/locale'
import type { MapLayerId } from '../domain/mapLayer'

export interface SettingsStore {
  mapLayer(): MapLayerId | null
  saveMapLayer(layer: MapLayerId): void
  locale(): Locale | null
  saveLocale(locale: Locale): void
}
