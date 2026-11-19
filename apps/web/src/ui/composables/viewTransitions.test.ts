import { afterEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

import { installPlotTransitions, plotTransitionName } from './viewTransitions'

const page = { template: '<div />' }

function setup(reduced = false) {
  const startViewTransition = vi.fn((update: () => Promise<void>) => {
    void update()

    return { finished: Promise.resolve(), ready: Promise.resolve(), updateCallbackDone: Promise.resolve() }
  })
  Object.assign(document, { startViewTransition })
  vi.stubGlobal('matchMedia', (query: string) => ({ matches: reduced && query.includes('reduce') }))
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/plots', name: 'plots', component: page },
      { path: '/plots/:id', name: 'plot', component: page },
      { path: '/reports', name: 'reports', component: page },
    ],
  })
  installPlotTransitions(router)

  return { router, startViewTransition }
}

afterEach(() => {
  Reflect.deleteProperty(document, 'startViewTransition')
  vi.unstubAllGlobals()
})

describe('plot view transitions', () => {
  it('morphs between the list and a plot card', async () => {
    const { router, startViewTransition } = setup()
    await router.push('/plots')

    await router.push('/plots/plot-1')
    await router.push('/plots')

    expect(startViewTransition).toHaveBeenCalledTimes(2)
    expect(router.currentRoute.value.name).toBe('plots')
    expect(plotTransitionName('plot-1')).toBe('plot-plot-1')
  })

  it('skips the transition for other pages and when motion is reduced', async () => {
    const { router, startViewTransition } = setup(true)
    await router.push('/plots')
    await router.push('/plots/plot-1')

    expect(startViewTransition).not.toHaveBeenCalled()

    const plain = setup()
    await plain.router.push('/reports')
    await plain.router.push('/plots')
    expect(plain.startViewTransition).not.toHaveBeenCalled()
  })

  it('navigates without the API', async () => {
    const { router } = setup()
    Reflect.deleteProperty(document, 'startViewTransition')
    await router.push('/plots')
    await router.push('/plots/plot-2')

    expect(router.currentRoute.value.params['id']).toBe('plot-2')
  })
})
