import type { MapLayerId } from './mapLayer'

export type PointerKind = 'mouse' | 'touch' | 'pen'

export type ObstructionReason = 'trees' | 'buildings' | 'unclear'

export const obstructionReasons: readonly ObstructionReason[] = ['trees', 'buildings', 'unclear']

export interface Capture {
  readonly zoom: number
  readonly layer: MapLayerId
  readonly pointer: PointerKind
}

export interface Obstruction {
  readonly side: number
  readonly reason: ObstructionReason
}

export type Gesture = Capture

export type MapGesture = Omit<Gesture, 'layer'>

export interface Survey {
  readonly zooms: readonly (number | null)[]
  readonly layer: MapLayerId | null
  readonly pointer: PointerKind | null
  readonly obstructions: readonly Obstruction[]
}

const closedShapePoints = 3

export const emptySurvey: Survey = { zooms: [], layer: null, pointer: null, obstructions: [] }

export function surveyOfPlot(
  pointCount: number,
  capture: Capture | null,
  obstructions: readonly Obstruction[],
): Survey {
  return {
    zooms: Array.from({ length: pointCount }, () => capture?.zoom ?? null),
    layer: capture?.layer ?? null,
    pointer: capture?.pointer ?? null,
    obstructions: [...obstructions],
  }
}

export function captureOf(survey: Survey): Capture | null {
  const known = survey.zooms.filter((zoom): zoom is number => zoom !== null)
  if (known.length === 0 || survey.layer === null || survey.pointer === null) {
    return null
  }

  return { zoom: Math.min(...known), layer: survey.layer, pointer: survey.pointer }
}

function withGesture(survey: Survey, gesture: Gesture): Pick<Survey, 'layer' | 'pointer'> {
  return { ...survey, layer: gesture.layer, pointer: gesture.pointer }
}

export function surveyAfterAdd(survey: Survey, gesture: Gesture): Survey {
  const count = survey.zooms.length
  if (count >= closedShapePoints) {
    return surveyAfterInsert(survey, count - 1, gesture)
  }

  return { ...survey, ...withGesture(survey, gesture), zooms: [...survey.zooms, gesture.zoom] }
}

export function surveyAfterMove(survey: Survey, index: number, gesture: Gesture): Survey {
  const zooms = survey.zooms.map((zoom, position) =>
    position === index ? Math.min(zoom ?? gesture.zoom, gesture.zoom) : zoom,
  )

  return { ...survey, ...withGesture(survey, gesture), zooms }
}

export function surveyAfterInsert(survey: Survey, afterIndex: number, gesture: Gesture): Survey {
  const zooms = [...survey.zooms.slice(0, afterIndex + 1), gesture.zoom, ...survey.zooms.slice(afterIndex + 1)]
  const obstructions = survey.obstructions.flatMap((obstruction) => {
    if (obstruction.side < afterIndex) {
      return [obstruction]
    }
    if (obstruction.side > afterIndex) {
      return [{ ...obstruction, side: obstruction.side + 1 }]
    }

    return [obstruction, { ...obstruction, side: afterIndex + 1 }]
  })

  return { ...survey, ...withGesture(survey, gesture), zooms, obstructions }
}

export function surveyAfterRemoveLast(survey: Survey): Survey {
  const lastSide = survey.zooms.length - 2

  return {
    ...survey,
    zooms: survey.zooms.slice(0, -1),
    obstructions: survey.obstructions.filter((obstruction) => obstruction.side < lastSide),
  }
}

export function surveyAfterSwap(survey: Survey, first: number, second: number): Survey {
  const count = survey.zooms.length
  const newIndex = (index: number): number => (index === first ? second : index === second ? first : index)
  const zooms = survey.zooms.map((_, position) => survey.zooms[newIndex(position)] ?? null)
  const obstructions = survey.obstructions.flatMap((obstruction) => {
    const from = newIndex(obstruction.side)
    const to = newIndex((obstruction.side + 1) % count)
    if (to === (from + 1) % count) {
      return [{ ...obstruction, side: from }]
    }
    if (from === (to + 1) % count) {
      return [{ ...obstruction, side: to }]
    }

    return []
  })

  return { ...survey, zooms, obstructions: sortBySide(obstructions) }
}

export function surveyWithObstruction(survey: Survey, side: number, reason: ObstructionReason | null): Survey {
  const others = survey.obstructions.filter((obstruction) => obstruction.side !== side)
  const obstructions = reason === null ? others : [...others, { side, reason }]

  return { ...survey, obstructions: sortBySide(obstructions) }
}

export function obstructionOn(survey: Pick<Survey, 'obstructions'>, side: number): ObstructionReason | null {
  return survey.obstructions.find((obstruction) => obstruction.side === side)?.reason ?? null
}

function sortBySide(obstructions: readonly Obstruction[]): Obstruction[] {
  return [...obstructions].sort((first, second) => first.side - second.side)
}
