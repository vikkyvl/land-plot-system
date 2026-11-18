import { createPinia } from 'pinia'
import { createApp } from 'vue'

import App from '../ui/App.vue'
import { createAppRouter } from '../ui/router'
import '../ui/styles/index.css'

createApp(App).use(createPinia()).use(createAppRouter()).mount('#app')
