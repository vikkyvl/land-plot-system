import { createPinia } from 'pinia'
import { createApp } from 'vue'

import { createUserManager, OidcSession } from '../auth/oidcSession'
import { HttpClient } from '../client/http'
import { ProfileClient } from '../client/profileClient'
import { readConfig } from '../config/env'
import { SessionService } from '../service/sessionService'
import App from '../ui/App.vue'
import { sessionServiceKey } from '../ui/composables/services'
import { createAppRouter } from '../ui/router'
import '../ui/styles/index.css'

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
const sessionService = new SessionService(auth, new ProfileClient(http))

createApp(App)
  .provide(sessionServiceKey, sessionService)
  .use(createPinia())
  .use(createAppRouter(sessionService))
  .mount('#app')
