import { createStore, del, get, set, type UseStore } from 'idb-keyval'

import type { Coordinate } from '../domain/coordinate'
import type { PlotDraft } from '../domain/draft'
import type { DraftStore } from '../port/draftStore'

const databaseName = 'arqetra'
const storeName = 'drafts'
const draftKey = 'calculator'

export class IndexedDbDraftStore implements DraftStore {
  private readonly store: UseStore

  constructor(store: UseStore = createStore(databaseName, storeName)) {
    this.store = store
  }

  async load(): Promise<PlotDraft | null> {
    return toDraft(await get<unknown>(draftKey, this.store))
  }

  save(draft: PlotDraft): Promise<void> {
    const plain: PlotDraft = {
      name: draft.name,
      points: draft.points.map(({ latitude, longitude }) => ({ latitude, longitude })),
      saveAfterSignIn: draft.saveAfterSignIn,
    }

    return set(draftKey, plain, this.store)
  }

  clear(): Promise<void> {
    return del(draftKey, this.store)
  }
}

function toDraft(value: unknown): PlotDraft | null {
  if (typeof value !== 'object' || value === null) {
    return null
  }

  const { name, points, saveAfterSignIn } = value as Record<string, unknown>
  if (typeof name !== 'string' || !Array.isArray(points) || !points.every(isCoordinate)) {
    return null
  }

  return { name, points, saveAfterSignIn: saveAfterSignIn === true }
}

function isCoordinate(value: unknown): value is Coordinate {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const { latitude, longitude } = value as Record<string, unknown>

  return Number.isFinite(latitude) && Number.isFinite(longitude)
}
