import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp } from 'vue'

import type { Plot } from '../../domain/plot'
import type { ConfirmationCheck, ConfirmationService } from '../../service/confirmationService'
import { confirmationServiceKey } from '../composables/services'

import { maximumAttempts, pollInterval, useConfirmationsStore } from './confirmations'
import { useToastStore } from './toast'

const pendingPlot = { id: 'plot-1', name: 'Поле', status: 'PENDING', version: 1 } as unknown as Plot
const confirmedPlot = { ...pendingPlot, status: 'CONFIRMED', version: 2 } as Plot

function setup(answers: ConfirmationCheck[], fallback: ConfirmationCheck = { state: 'pending', plot: pendingPlot }) {
  const check = vi.fn(async () => answers.shift() ?? fallback)
  const app = createApp({})
  app.provide(confirmationServiceKey, { check } as unknown as ConfirmationService)
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)
  const store = app.runWithContext(() => useConfirmationsStore())
  const toasts = app.runWithContext(() => useToastStore())

  return { store, toasts, check }
}

function setVisibility(state: 'visible' | 'hidden'): void {
  Object.defineProperty(document, 'visibilityState', { value: state, configurable: true })
  document.dispatchEvent(new Event('visibilitychange'))
}

async function tick(times = 1): Promise<void> {
  await vi.advanceTimersByTimeAsync(pollInterval * times)
}

beforeEach(() => {
  vi.useFakeTimers()
  setVisibility('visible')
})

afterEach(() => {
  vi.useRealTimers()
})

describe('confirmations store', () => {
  it('asks every 2 s while the plot is pending and stops once it is confirmed', async () => {
    const { store, toasts, check } = setup([
      { state: 'pending', plot: pendingPlot },
      { state: 'confirmed', plot: confirmedPlot },
    ])

    store.follow('plot-1')
    await vi.advanceTimersByTimeAsync(pollInterval - 1)
    expect(check).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)
    expect(check).toHaveBeenCalledTimes(1)
    await tick()

    expect(store.entries['plot-1']).toEqual({ phase: 'confirmed', plot: confirmedPlot })
    expect(toasts.toasts.map((toast) => toast.text)).toEqual(['Ділянку «Поле» збережено'])
    await tick(5)
    expect(check).toHaveBeenCalledTimes(2)
  })

  it('explains a save cancelled by the saga', async () => {
    const { store, toasts } = setup([{ state: 'cancelled' }])

    store.expect('plot-1', { intent: 'create', name: 'Поле' })
    store.follow('plot-1')
    await tick()

    expect(store.entries['plot-1']?.phase).toBe('cancelled')
    expect(toasts.toasts[0]?.text).toContain('Збереження «Поле» скасовано')
    expect(toasts.toasts[0]?.text).toContain('проєкцію ділянки')
  })

  it('reports a change rolled back with the restored version', async () => {
    const { store, toasts, check } = setup([{ state: 'reverted', plot: confirmedPlot }])

    store.expect('plot-1', { intent: 'update', name: 'Поле', expected: { name: 'FAIL Поле' } })
    store.follow('plot-1')
    await tick()

    expect(check).toHaveBeenCalledWith('plot-1', { name: 'FAIL Поле' })
    expect(store.entries['plot-1']).toEqual({ phase: 'reverted', plot: confirmedPlot })
    expect(toasts.toasts[0]?.text).toContain('Зміну відкочено')
    expect(toasts.toasts[0]?.text).toContain('версію 2')
  })

  it('gives up after 60 s and checks again on request', async () => {
    const { store, check } = setup([])

    store.follow('plot-1')
    await tick(maximumAttempts)
    expect(check).toHaveBeenCalledTimes(maximumAttempts)
    expect(store.entries['plot-1']?.phase).toBe('stalled')

    await tick(10)
    expect(check).toHaveBeenCalledTimes(maximumAttempts)

    store.retry('plot-1')
    await vi.advanceTimersByTimeAsync(0)
    expect(check).toHaveBeenCalledTimes(maximumAttempts + 1)
    expect(store.entries['plot-1']?.phase).toBe('pending')
  })

  it('pauses while the tab is hidden and checks at once when it is back', async () => {
    const { store, check } = setup([])

    store.follow('plot-1')
    await tick()
    expect(check).toHaveBeenCalledTimes(1)

    setVisibility('hidden')
    await tick(5)
    expect(check).toHaveBeenCalledTimes(1)

    setVisibility('visible')
    await vi.advanceTimersByTimeAsync(0)
    expect(check).toHaveBeenCalledTimes(2)
    await tick()
    expect(check).toHaveBeenCalledTimes(3)
  })

  it('keeps one poll per plot and stops when every page has left', async () => {
    const { store, check } = setup([])

    const leaveList = store.follow('plot-1')
    const leaveDetail = store.follow('plot-1')
    await tick(3)
    expect(check).toHaveBeenCalledTimes(3)

    leaveList()
    leaveList()
    await tick()
    expect(check).toHaveBeenCalledTimes(4)

    leaveDetail()
    await tick(5)
    expect(check).toHaveBeenCalledTimes(4)
    expect(store.isPolling('plot-1')).toBe(false)
  })
})
