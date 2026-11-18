import { inject, type InjectionKey } from 'vue'

import type { MapLayers } from '../../domain/mapLayer'
import type { CalculatorService } from '../../service/calculatorService'
import type { ConfirmationService } from '../../service/confirmationService'
import type { PlaceSearchService } from '../../service/placeSearchService'
import type { PlotService } from '../../service/plotService'
import type { ReportService } from '../../service/reportService'
import type { SessionService } from '../../service/sessionService'
import type { SettingsService } from '../../service/settingsService'

export const sessionServiceKey: InjectionKey<SessionService> = Symbol('SessionService')
export const calculatorServiceKey: InjectionKey<CalculatorService> = Symbol('CalculatorService')
export const placeSearchServiceKey: InjectionKey<PlaceSearchService> = Symbol('PlaceSearchService')
export const settingsServiceKey: InjectionKey<SettingsService> = Symbol('SettingsService')
export const plotServiceKey: InjectionKey<PlotService> = Symbol('PlotService')
export const reportServiceKey: InjectionKey<ReportService> = Symbol('ReportService')
export const confirmationServiceKey: InjectionKey<ConfirmationService> = Symbol('ConfirmationService')
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
export const useSettingsService = (): SettingsService => required(settingsServiceKey)
export const usePlotService = (): PlotService => required(plotServiceKey)
export const useReportService = (): ReportService => required(reportServiceKey)
export const useConfirmationService = (): ConfirmationService => required(confirmationServiceKey)
export const useMapLayers = (): MapLayers => required(mapLayersKey)
