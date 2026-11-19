import type { RouteLocationNormalized, Router } from 'vue-router'

export const maxStaggered = 8

const plotRoutes = new Set(['plots', 'plot'])
const readyTimeout = 800
const reducedMotion = '(prefers-reduced-motion: reduce)'

export function plotTransitionName(id: string): string {
  return `plot-${id}`
}

export function prefersReducedMotion(): boolean {
  return typeof window.matchMedia === 'function' && window.matchMedia(reducedMotion).matches
}

function canTransition(to: RouteLocationNormalized, from: RouteLocationNormalized): boolean {
  return (
    typeof document.startViewTransition === 'function' &&
    !prefersReducedMotion() &&
    plotRoutes.has(String(to.name)) &&
    plotRoutes.has(String(from.name)) &&
    to.fullPath !== from.fullPath
  )
}

function whenVisible(name: string): Promise<void> {
  return new Promise((resolve) => {
    const started = performance.now()
    const check = (): void => {
      const present = [...document.querySelectorAll<HTMLElement>('[style*="view-transition-name"]')].some(
        (element) => element.style.viewTransitionName === name,
      )
      if (present || performance.now() - started > readyTimeout) {
        resolve()

        return
      }
      requestAnimationFrame(check)
    }
    check()
  })
}

export function installPlotTransitions(router: Router): void {
  let finish: (() => void) | null = null

  router.beforeResolve((to, from) => {
    if (!canTransition(to, from)) {
      return true
    }

    return new Promise<boolean>((resolve) => {
      document.startViewTransition(
        () =>
          new Promise<void>((done) => {
            finish = done
            resolve(true)
          }),
      )
    })
  })

  router.afterEach(async (to) => {
    const done = finish
    finish = null
    if (done === null) {
      return
    }
    const id = to.params['id']
    if (typeof id === 'string') {
      await whenVisible(plotTransitionName(id))
    }
    done()
  })
}
