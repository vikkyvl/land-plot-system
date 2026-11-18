import { inject, type InjectionKey } from 'vue'

import type { MapLayers } from '../../domain/mapLayer'
import type { CalculatorService } from '../../service/calculatorService'
import type { MapPreferencesService } from '../../service/mapPreferencesService'
import type { PlaceSearchService } from '../../service/placeSearchService'
import type { PlotService } from '../../service/plotService'
import type { ReportService } from '../../service/reportService'
import type { SessionService } from '../../service/sessionService'

export const sessionServiceKey: InjectionKey<SessionService> = Symbol('SessionService')
export const calculatorServiceKey: InjectionKey<CalculatorService> = Symbol('CalculatorService')
export const placeSearchServiceKey: InjectionKey<PlaceSearchService> = Symbol('PlaceSearchService')
export const mapPreferencesServiceKey: InjectionKey<MapPreferencesService> = Symbol('MapPreferencesService')
export const plotServiceKey: InjectionKey<PlotService> = Symbol('PlotService')
export const reportServiceKey: InjectionKey<ReportService> = Symbol('ReportService')
export const mapLayersKey: InjectionKey<MapLayers> = Symbol('MapLayers')

function required<T>(key: InjectionKey<T>): T {
  const value = inject(key)
  if (value === undefined) {
    throw new Error(`${String(key.description)} is not provided`)
  }

  return value
}

export const useSessionService = (): SessionService => required(sessionServiceKey)
export const useCalculatorService = (): CalculatorService => required(calculatorServiceKey)
export const usePlaceSearchService = (): PlaceSearchService => required(placeSearchServiceKey)
export const useMapPreferencesService = (): MapPreferencesService => required(mapPreferencesServiceKey)
export const usePlotService = (): PlotService => required(plotServiceKey)
export const useReportService = (): ReportService => required(reportServiceKey)
export const useMapLayers = (): MapLayers => required(mapLayersKey)
