import { createPinia } from 'pinia'
import { createApp } from 'vue'

import { createUserManager, OidcSession } from '../auth/oidcSession'
import { HttpClient } from '../client/http'
import { NominatimClient } from '../client/nominatimClient'
import { PlotsClient } from '../client/plotsClient'
import { ProfileClient } from '../client/profileClient'
import { ReportsClient } from '../client/reportsClient'
import { readConfig } from '../config/env'
import { IndexedDbDraftStore } from '../repository/indexedDbDraftStore'
import { LocalSettingsStore } from '../repository/localSettingsStore'
import { CalculatorService } from '../service/calculatorService'
import { ConfirmationService } from '../service/confirmationService'
import { PlaceSearchService } from '../service/placeSearchService'
import { PlotService } from '../service/plotService'
import { ReportService } from '../service/reportService'
import { SessionService } from '../service/sessionService'
import { SettingsService } from '../service/settingsService'
import App from '../ui/App.vue'
import {
  calculatorServiceKey,
  confirmationServiceKey,
  mapLayersKey,
  placeSearchServiceKey,
  plotServiceKey,
  reportServiceKey,
  sessionServiceKey,
  settingsServiceKey,
} from '../ui/composables/services'
import { applyLocale, i18n } from '../ui/i18n'
import { createAppRouter } from '../ui/router'
import '../ui/styles/index.css'

import { mapLayersFrom } from './mapLayers'

const config = readConfig(import.meta.env)
const auth = new OidcSession(
  createUserManager({
    authority: config.oidcAuthority,
    clientId: config.oidcClientId,
    origin: window.location.origin,
    storage: window.sessionStorage,
  }),
)
const http = new HttpClient({ baseUrl: config.gatewayUrl, session: auth })
const plots = new PlotsClient(http)
const sessionService = new SessionService(auth, new ProfileClient(http))
const calculatorService = new CalculatorService(plots, new IndexedDbDraftStore(), auth)
const placeSearchService = new PlaceSearchService(new NominatimClient({ baseUrl: config.geocodingUrl }))
const reports = new ReportsClient(http)
const plotService = new PlotService(plots, reports)
const reportService = new ReportService(reports, plots)
const settingsService = new SettingsService(new LocalSettingsStore(window.localStorage), navigator.language)
applyLocale(settingsService.locale())

createApp(App)
  .provide(sessionServiceKey, sessionService)
  .provide(calculatorServiceKey, calculatorService)
  .provide(confirmationServiceKey, new ConfirmationService(plots))
  .provide(placeSearchServiceKey, placeSearchService)
  .provide(plotServiceKey, plotService)
  .provide(reportServiceKey, reportService)
  .provide(settingsServiceKey, settingsService)
  .provide(mapLayersKey, mapLayersFrom(config))
  .use(i18n)
  .use(createPinia())
  .use(createAppRouter(sessionService))
  .mount('#app')
