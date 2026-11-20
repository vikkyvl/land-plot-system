import { localeFromLanguage, type Locale } from '../domain/locale'
import type { MapLayerId } from '../domain/mapLayer'
import type { SettingsStore } from '../port/settingsStore'

export const defaultMapLayer: MapLayerId = 'map'

export class SettingsService {
  private readonly store: SettingsStore
  private readonly browserLanguage: string | undefined

  constructor(store: SettingsStore, browserLanguage: string | undefined) {
    this.store = store
    this.browserLanguage = browserLanguage
  }

  layer(): MapLayerId {
    return this.store.mapLayer() ?? defaultMapLayer
  }

  chooseLayer(layer: MapLayerId): void {
    this.store.saveMapLayer(layer)
  }

  locale(): Locale {
    return this.store.locale() ?? localeFromLanguage(this.browserLanguage)
  }

  chooseLocale(locale: Locale): void {
    this.store.saveLocale(locale)
  }
}
