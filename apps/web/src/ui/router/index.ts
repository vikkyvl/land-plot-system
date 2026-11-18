import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'

import AuthCallbackView from '../views/AuthCallbackView.vue'
import HomeView from '../views/HomeView.vue'
import LoginView from '../views/LoginView.vue'

import { guardRoute, homeRouteName, loginRouteName, type SignedInCheck } from './guard'

export function createAppRouter(session: SignedInCheck, history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: homeRouteName, component: HomeView },
      { path: '/login', name: loginRouteName, component: LoginView, meta: { guestOnly: true } },
      { path: '/auth/callback', name: 'auth-callback', component: AuthCallbackView },
    ],
  })

  router.beforeEach((to) => guardRoute(to, session))

  return router
}
