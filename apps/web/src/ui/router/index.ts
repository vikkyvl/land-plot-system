import { createRouter, createWebHistory, type Router, type RouterHistory } from 'vue-router'

import { guardRoute, homeRouteName, loginRouteName, type SignedInCheck } from './guard'
import { installPlotTransitions } from '../composables/viewTransitions'

const CalcView = () => import('../views/CalcView.vue')

export function createAppRouter(session: SignedInCheck, history: RouterHistory = createWebHistory()): Router {
  const router = createRouter({
    history,
    routes: [
      { path: '/', name: homeRouteName, component: CalcView },
      {
        path: '/login',
        name: loginRouteName,
        component: () => import('../views/LoginView.vue'),
        meta: { guestOnly: true, bottomNav: false },
      },
      {
        path: '/auth/callback',
        name: 'auth-callback',
        component: () => import('../views/AuthCallbackView.vue'),
        meta: { bottomNav: false },
      },
      {
        path: '/plots',
        component: () => import('../views/PlotsView.vue'),
        meta: { requiresAuth: true, ownMobileHeader: true },
        children: [
          { path: '', name: 'plots', component: () => import('../views/PlotsEmptyView.vue') },
          { path: ':id', name: 'plot', component: () => import('../views/PlotDetailView.vue'), props: true },
        ],
      },
      {
        path: '/plots/:id/edit',
        name: 'plot-edit',
        component: CalcView,
        props: (route) => ({ plotId: route.params['id'] }),
        meta: { requiresAuth: true },
      },
      {
        path: '/reports',
        name: 'reports',
        component: () => import('../views/ReportsView.vue'),
        meta: { requiresAuth: true, ownMobileHeader: true },
      },
      {
        path: '/reports/:id',
        name: 'report',
        component: () => import('../views/ReportView.vue'),
        props: true,
        meta: { requiresAuth: true, ownMobileHeader: true },
      },
    ],
  })

  router.beforeEach((to) => guardRoute(to, session))
  installPlotTransitions(router)

  return router
}
