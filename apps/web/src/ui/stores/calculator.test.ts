import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createApp } from 'vue'

import { DomainError } from '../../domain/error'
import type { PlotDraft } from '../../domain/draft'
import type { Plot, PlotPreview } from '../../domain/plot'
import { emptySurvey } from '../../domain/survey'
import type { CalculatorService } from '../../service/calculatorService'
import { calculatorServiceKey } from '../composables/services'

import { previewDelay, useCalculatorStore } from './calculator'

const a = { latitude: 50.45, longitude: 30.52 }
const b = { latitude: 50.451, longitude: 30.523 }
const c = { latitude: 50.449, longitude: 30.524 }
const d = { latitude: 50.448, longitude: 30.521 }
const gesture = { zoom: 18, layer: 'map', pointer: 'mouse' } as const

const valid: PlotPreview = {
  valid: true,
  measurement: { area: { squareMeters: 24810, ares: 248.1, hectares: 2.481 }, perimeterMeters: 612 },
  accuracy: null,
  problems: [],
}

const crossed: PlotPreview = {
  valid: false,
  measurement: null,
  accuracy: null,
  problems: [
    {
      code: 'self_intersection',
      message: 'polygon sides intersect',
      details: { kind: 'self_intersection', segments: [[1, 2], [3, 0]], swap: [2, 3], point: null },
    },
  ],
}

function fakeService(overrides: { preview?: () => Promise<PlotPreview>; loadDraft?: () => Promise<PlotDraft>; savePendingDraft?: () => Promise<Plot | null> } = {}) {
  let draft: PlotDraft = { name: '', points: [], survey: emptySurvey, saveAfterSignIn: false }

  return {
    canPreview: (points: readonly unknown[]) => points.length >= 3,
    preview: vi.fn(overrides.preview ?? (async (): Promise<PlotPreview> => valid)),
    loadDraft: vi.fn(overrides.loadDraft ?? (async () => draft)),
    keepDraft: vi.fn(async (next: PlotDraft) => {
      draft = next
    }),
    save: vi.fn(async (next: PlotDraft) => ({ id: 'plot-1', name: next.name, status: 'PENDING' }) as Plot),
    saveAfterSignIn: vi.fn(async (next: PlotDraft) => {
      draft = { ...next, survey: emptySurvey, saveAfterSignIn: true }
    }),
    savePendingDraft: vi.fn(overrides.savePendingDraft ?? (async (): Promise<Plot | null> => null)),
    loadPlot: vi.fn(
      async () =>
        ({
          id: 'plot-1',
          name: 'Поле',
          points: [a, b, c],
          capture: { zoom: 17, layer: 'satellite', pointer: 'touch' },
          obstructions: [{ side: 2, reason: 'trees' }],
          status: 'CONFIRMED',
        }) as unknown as Plot,
    ),
    update: vi.fn(async (_id: string, next: PlotDraft) => ({ id: 'plot-1', name: next.name, status: 'PENDING' }) as Plot),
  }
}

function useStore(service = fakeService()) {
  const app = createApp({})
  app.provide(calculatorServiceKey, service as unknown as CalculatorService)
  const pinia = createPinia()
  app.use(pinia)
  setActivePinia(pinia)

  return { store: app.runWithContext(() => useCalculatorStore()), service }
}

async function settle(milliseconds = previewDelay): Promise<void> {
  await vi.advanceTimersByTimeAsync(milliseconds)
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('calculator store', () => {
  it('asks the server for a preview 300 ms after the third point, not earlier', async () => {
    const { store, service } = useStore()

    store.addPoint(a, gesture)
    store.addPoint(b, gesture)
    await settle()
    expect(service.preview).not.toHaveBeenCalled()

    store.addPoint(c, gesture)
    await settle(previewDelay - 1)
    expect(service.preview).not.toHaveBeenCalled()
    expect(store.updating).toBe(true)

    await settle(1)
    expect(service.preview).toHaveBeenCalledTimes(1)
    expect(service.preview).toHaveBeenCalledWith([a, b, c], expect.anything())
    expect(store.measurement?.area.hectares).toBe(2.481)
    expect(store.canSave).toBe(true)
  })

  it('asks for a new preview with the marks when a side is marked', async () => {
    const { store, service } = useStore()
    store.addPoint(a, gesture)
    store.addPoint(b, gesture)
    store.addPoint(c, gesture)
    await settle()

    store.markSide(1, 'trees')
    await settle()

    expect(service.preview).toHaveBeenCalledTimes(2)
    expect(service.preview).toHaveBeenLastCalledWith([a, b, c], {
      zooms: [18, 18, 18],
      layer: 'map',
      pointer: 'mouse',
      obstructions: [{ side: 1, reason: 'trees' }],
    })
  })

  it('sends one preview for quick edits in a row', async () => {
    const { store, service } = useStore()
    store.addPoint(a, gesture)
    store.addPoint(b, gesture)
    store.addPoint(c, gesture)
    await settle(100)
    store.insertPoint(2, d, gesture)
    await settle(100)
    store.movePoint(0, b, gesture)
    await settle()

    expect(service.preview).toHaveBeenCalledTimes(1)
    expect(service.preview).toHaveBeenCalledWith([b, b, c, d], expect.anything())
  })

  it('shows the last value as updating while a new preview runs and ignores stale answers', async () => {
    let answer: (preview: PlotPreview) => void = () => undefined
    const service = fakeService()
    const { store } = useStore(service)
    store.addPoint(a, gesture)
    store.addPoint(b, gesture)
    store.addPoint(c, gesture)
    await settle()

    service.preview.mockImplementationOnce(() => new Promise((resolve) => (answer = resolve)))
    store.addPoint(d, gesture)
    await settle()
    expect(store.updating).toBe(true)
    expect(store.preview).toBe(valid)

    store.undo()
    await settle()
    answer(crossed)
    await settle(0)

    expect(store.preview).toBe(valid)
    expect(store.updating).toBe(false)
  })

  it('highlights the self-intersection and swaps the points from details', async () => {
    const service = fakeService({ preview: async () => crossed })
    const { store } = useStore(service)
    for (const point of [a, b, c, d]) {
      store.addPoint(point, gesture)
    }
    await settle()

    expect(store.selfIntersection?.swap).toEqual([2, 3])
    expect(store.measurement).toBeNull()
    expect(store.canSave).toBe(false)

    store.swap(2, 3)
    expect(store.points).toEqual([a, b, d, c])
  })

  it('counts down Retry-After after 429 and asks again', async () => {
    const limited = new DomainError({ kind: 'rate_limited', code: 'rate_limited', message: '', retryAfterSeconds: 3 })
    const service = fakeService()
    service.preview.mockRejectedValueOnce(limited)
    const { store } = useStore(service)
    store.addPoint(a, gesture)
    store.addPoint(b, gesture)
    store.addPoint(c, gesture)
    await settle()

    expect(store.outage).toEqual({ rateLimited: true, secondsLeft: 3 })
    expect(store.canSave).toBe(false)
    await settle(1000)
    expect(store.outage?.secondsLeft).toBe(2)
    await settle(2000)
    await settle(100)

    expect(service.preview).toHaveBeenCalledTimes(2)
    expect(store.outage).toBeNull()
    expect(store.canSave).toBe(true)
  })

  it('keeps the guest draft for saving after the sign-in', async () => {
    const { store, service } = useStore()
    store.addPoint(a, gesture)
    store.rename('Поле')

    expect(await store.save(false)).toBe('sign-in')
    expect(service.saveAfterSignIn).toHaveBeenCalledWith({
      name: 'Поле',
      points: [a],
      survey: { zooms: [18], layer: 'map', pointer: 'mouse', obstructions: [] },
      saveAfterSignIn: false,
    })
  })

  it('saves for a signed-in user and starts a new plot', async () => {
    const { store } = useStore()
    store.addPoint(a, gesture)
    store.rename('Поле')

    expect(await store.save(true)).toBe('saved')
    expect(store.savedPlot?.name).toBe('Поле')
    expect(store.points).toEqual([])
  })

  it('goes to the sign-in when the session has expired and keeps other errors', async () => {
    const expired = new DomainError({ kind: 'unauthorized', code: 'unauthorized', message: '' })
    const invalid = new DomainError({ kind: 'validation', code: 'invalid_name', message: '' })
    const service = fakeService()
    service.save.mockRejectedValueOnce(expired).mockRejectedValueOnce(invalid)
    const { store } = useStore(service)

    expect(await store.save(true)).toBe('sign-in')
    expect(service.saveAfterSignIn).toHaveBeenCalledTimes(1)
    expect(await store.save(true)).toBe('failed')
    expect(store.saveError?.code).toBe('invalid_name')
  })

  it('restores the draft and finishes the save started before the sign-in', async () => {
    const service = fakeService({
      loadDraft: async () => ({ name: 'Поле', points: [a, b, c], survey: emptySurvey, saveAfterSignIn: true }),
      savePendingDraft: async () => ({ id: 'plot-1', name: 'Поле', status: 'PENDING' }) as Plot,
    })
    const { store } = useStore(service)

    await store.start()

    expect(store.savedPlot?.id).toBe('plot-1')
    expect(store.points).toEqual([])
  })

  it('edits a saved plot with PATCH and leaves the guest draft alone', async () => {
    const { store, service } = useStore()

    await store.startEditing('plot-1')
    await settle(0)
    expect(store.editingId).toBe('plot-1')
    expect(store.points).toEqual([a, b, c])
    expect(service.preview).toHaveBeenCalledWith([a, b, c], expect.anything())

    store.addPoint(d, gesture)
    await settle()
    expect(service.keepDraft).not.toHaveBeenCalled()

    expect(await store.save(true)).toBe('updated')
    expect(service.update).toHaveBeenCalledWith('plot-1', {
      name: 'Поле',
      points: [a, b, c, d],
      survey: {
        zooms: [17, 17, 17, 18],
        layer: 'map',
        pointer: 'mouse',
        obstructions: [
          { side: 2, reason: 'trees' },
          { side: 3, reason: 'trees' },
        ],
      },
      saveAfterSignIn: false,
    })
    expect(service.save).not.toHaveBeenCalled()
  })
})
