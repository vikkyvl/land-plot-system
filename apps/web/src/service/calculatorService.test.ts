import { describe, expect, it, vi } from 'vitest'

import { DomainError } from '../domain/error'
import type { PlotDraft } from '../domain/draft'
import type { Plot } from '../domain/plot'
import { emptySurvey } from '../domain/survey'
import type { DraftStore } from '../port/draftStore'
import type { PlotGateway } from '../port/plotGateway'

import { CalculatorService } from './calculatorService'

const points = [
  { latitude: 50.45, longitude: 30.52 },
  { latitude: 50.451, longitude: 30.523 },
  { latitude: 50.449, longitude: 30.524 },
]

const plot = { id: 'plot-1', name: 'Поле', status: 'PENDING' } as Plot

class MemoryDrafts implements DraftStore {
  draft: PlotDraft | null

  constructor(draft: PlotDraft | null = null) {
    this.draft = draft
  }

  async load() {
    return this.draft
  }

  async save(draft: PlotDraft) {
    this.draft = draft
  }

  async clear() {
    this.draft = null
  }
}

function setup(options: { draft?: PlotDraft | null; signedIn?: boolean; create?: () => Promise<Plot> } = {}) {
  const plots = {
    create: vi.fn(options.create ?? (async () => plot)),
    get: vi.fn(async () => plot),
    update: vi.fn(async () => plot),
    preview: vi.fn(async () => ({ valid: true, measurement: null, problems: [] })),
  } as unknown as PlotGateway & Record<'create' | 'get' | 'update', ReturnType<typeof vi.fn>>
  const drafts = new MemoryDrafts(options.draft ?? null)
  const service = new CalculatorService(plots, drafts, { isSignedIn: async () => options.signedIn ?? true })

  return { plots, drafts, service }
}

describe('CalculatorService', () => {
  it('previews only from three points', () => {
    const { service } = setup()

    expect(service.canPreview(points.slice(0, 2))).toBe(false)
    expect(service.canPreview(points)).toBe(true)
  })

  it('starts from an empty draft and keeps changes', async () => {
    const { service, drafts } = setup()

    expect(await service.loadDraft()).toEqual({ name: '', points: [], survey: emptySurvey, saveAfterSignIn: false })
    await service.keepDraft({ name: 'Поле', points, survey: emptySurvey, saveAfterSignIn: false })
    expect(drafts.draft?.points).toEqual(points)
  })

  it('saves a plot with a trimmed name and forgets the draft', async () => {
    const { service, plots, drafts } = setup({ draft: { name: ' Поле ', points, survey: emptySurvey, saveAfterSignIn: false } })

    expect(await service.save({ name: ' Поле ', points, survey: emptySurvey, saveAfterSignIn: false })).toBe(plot)
    expect(plots.create).toHaveBeenCalledWith({ name: 'Поле', points, capture: null, obstructions: [] })
    expect(drafts.draft).toBeNull()
  })

  it('saves the guest draft once the user is signed in', async () => {
    const { service, drafts } = setup({ signedIn: false })

    await service.saveAfterSignIn({ name: 'Поле', points, survey: emptySurvey, saveAfterSignIn: false })

    expect(drafts.draft?.saveAfterSignIn).toBe(true)
    expect(await service.savePendingDraft()).toBeNull()

    const signedIn = setup({ draft: drafts.draft, signedIn: true })
    expect(await signedIn.service.savePendingDraft()).toBe(plot)
    expect(signedIn.drafts.draft).toBeNull()
  })

  it('does nothing without a pending draft', async () => {
    const { service, plots } = setup({ draft: { name: 'Поле', points, survey: emptySurvey, saveAfterSignIn: false } })

    expect(await service.savePendingDraft()).toBeNull()
    expect(plots.create).not.toHaveBeenCalled()
  })

  it('keeps the draft without the pending mark when the automatic save fails', async () => {
    const rejected = new DomainError({ kind: 'validation', code: 'invalid_name', message: 'name is required' })
    const { service, drafts } = setup({
      draft: { name: '', points, survey: emptySurvey, saveAfterSignIn: true },
      create: async () => {
        throw rejected
      },
    })

    await expect(service.savePendingDraft()).rejects.toBe(rejected)
    expect(drafts.draft).toEqual({ name: '', points, survey: emptySurvey, saveAfterSignIn: false })
  })

  it('loads a saved plot and updates its boundary without touching the draft', async () => {
    const { service, plots, drafts } = setup({ draft: { name: 'Чернетка', points, survey: emptySurvey, saveAfterSignIn: false } })

    expect(await service.loadPlot('plot-1')).toBe(plot)
    await service.update('plot-1', { name: ' Поле ', points, survey: emptySurvey, saveAfterSignIn: false })

    expect(plots.update).toHaveBeenCalledWith('plot-1', { name: 'Поле', points, capture: null, obstructions: [] })
    expect(drafts.draft?.name).toBe('Чернетка')
  })
})
