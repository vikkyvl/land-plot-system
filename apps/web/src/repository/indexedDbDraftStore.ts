import { createStore, del, get, set, type UseStore } from 'idb-keyval'

import type { Coordinate } from '../domain/coordinate'
import type { PlotDraft } from '../domain/draft'
import { emptySurvey, obstructionReasons, type Obstruction, type Survey } from '../domain/survey'
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
      survey: {
        zooms: [...draft.survey.zooms],
        layer: draft.survey.layer,
        pointer: draft.survey.pointer,
        obstructions: draft.survey.obstructions.map(({ side, reason }) => ({ side, reason })),
      },
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

  const { name, points, survey, saveAfterSignIn } = value as Record<string, unknown>
  if (typeof name !== 'string' || !Array.isArray(points) || !points.every(isCoordinate)) {
    return null
  }

  return { name, points, survey: toSurvey(survey, points.length), saveAfterSignIn: saveAfterSignIn === true }
}

function toSurvey(value: unknown, pointCount: number): Survey {
  if (typeof value !== 'object' || value === null) {
    return { ...emptySurvey, zooms: Array.from({ length: pointCount }, () => null) }
  }

  const { zooms, layer, pointer, obstructions } = value as Record<string, unknown>
  const validZooms =
    Array.isArray(zooms) &&
    zooms.length === pointCount &&
    zooms.every((zoom) => zoom === null || Number.isFinite(zoom))

  return {
    zooms: validZooms ? (zooms as (number | null)[]) : Array.from({ length: pointCount }, () => null),
    layer: layer === 'map' || layer === 'satellite' ? layer : null,
    pointer: pointer === 'mouse' || pointer === 'touch' || pointer === 'pen' ? pointer : null,
    obstructions: Array.isArray(obstructions) ? obstructions.filter(isObstruction) : [],
  }
}

function isObstruction(value: unknown): value is Obstruction {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const { side, reason } = value as Record<string, unknown>

  return Number.isInteger(side) && obstructionReasons.includes(reason as Obstruction['reason'])
}

function isCoordinate(value: unknown): value is Coordinate {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  const { latitude, longitude } = value as Record<string, unknown>

  return Number.isFinite(latitude) && Number.isFinite(longitude)
}
