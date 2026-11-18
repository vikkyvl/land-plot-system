import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'

import AuthCallbackView from '../views/AuthCallbackView.vue'
import CalcView from '../views/CalcView.vue'
import LoginView from '../views/LoginView.vue'
import PlotDetailView from '../views/PlotDetailView.vue'
import PlotsEmptyView from '../views/PlotsEmptyView.vue'
import PlotsView from '../views/PlotsView.vue'

import { guardRoute, homeRouteName, loginRouteName, type SignedInCheck } from './guard'

export function createAppRouter(session: SignedInCheck, history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: homeRouteName, component: CalcView },
      { path: '/login', name: loginRouteName, component: LoginView, meta: { guestOnly: true, bottomNav: false } },
      { path: '/auth/callback', name: 'auth-callback', component: AuthCallbackView, meta: { bottomNav: false } },
      {
        path: '/plots',
        component: PlotsView,
        meta: { requiresAuth: true, ownMobileHeader: true },
        children: [
          { path: '', name: 'plots', component: PlotsEmptyView },
          { path: ':id', name: 'plot', component: PlotDetailView, props: true },
        ],
      },
      {
        path: '/plots/:id/edit',
        name: 'plot-edit',
        component: CalcView,
        props: (route) => ({ plotId: route.params['id'] }),
        meta: { requiresAuth: true },
      },
    ],
  })

  router.beforeEach((to) => guardRoute(to, session))

  return router
}
