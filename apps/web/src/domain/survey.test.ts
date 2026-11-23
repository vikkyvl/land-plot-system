import { describe, expect, it } from 'vitest'

import {
  captureOf,
  emptySurvey,
  surveyAfterAdd,
  surveyAfterInsert,
  surveyAfterMove,
  surveyAfterRemoveLast,
  surveyAfterSwap,
  surveyOfPlot,
  surveyWithObstruction,
  type Gesture,
  type Survey,
} from './survey'

const closeUp: Gesture = { zoom: 19, layer: 'satellite', pointer: 'touch' }
const farAway: Gesture = { zoom: 15, layer: 'map', pointer: 'mouse' }

function square(): Survey {
  return [closeUp, closeUp, closeUp, closeUp].reduce(surveyAfterAdd, emptySurvey)
}

describe('survey', () => {
  it('sends the smallest zoom of the current points with the last layer and pointer', () => {
    const survey = surveyAfterMove(surveyAfterAdd(square(), closeUp), 1, farAway)

    expect(captureOf(survey)).toEqual({ zoom: 15, layer: 'map', pointer: 'mouse' })
  })

  it('keeps the smallest zoom of a point when it is moved closer later', () => {
    const placedFar = surveyAfterAdd(square(), farAway)
    const movedClose = surveyAfterMove(placedFar, 4, closeUp)

    expect(captureOf(movedClose)?.zoom).toBe(15)
    expect(captureOf(surveyAfterRemoveLast(movedClose))?.zoom).toBe(19)
  })

  it('has no capture without points or for a plot saved without one', () => {
    expect(captureOf(emptySurvey)).toBeNull()
    expect(captureOf(surveyOfPlot(4, null, []))).toBeNull()
    expect(captureOf(surveyAfterMove(surveyOfPlot(4, null, []), 2, farAway))).toEqual(farAway)
  })

  it('marks and clears a side', () => {
    const marked = surveyWithObstruction(surveyWithObstruction(square(), 3, 'trees'), 1, 'unclear')

    expect(marked.obstructions).toEqual([
      { side: 1, reason: 'unclear' },
      { side: 3, reason: 'trees' },
    ])
    expect(surveyWithObstruction(marked, 1, null).obstructions).toEqual([{ side: 3, reason: 'trees' }])
    expect(surveyWithObstruction(marked, 1, 'buildings').obstructions).toContainEqual({ side: 1, reason: 'buildings' })
  })

  it('renumbers marks when a point is inserted into or after a side', () => {
    const marked = surveyWithObstruction(surveyWithObstruction(square(), 1, 'trees'), 3, 'unclear')

    expect(surveyAfterInsert(marked, 1, closeUp).obstructions).toEqual([
      { side: 1, reason: 'trees' },
      { side: 2, reason: 'trees' },
      { side: 4, reason: 'unclear' },
    ])
    expect(surveyAfterAdd(marked, closeUp).obstructions).toEqual([
      { side: 1, reason: 'trees' },
      { side: 3, reason: 'unclear' },
      { side: 4, reason: 'unclear' },
    ])
  })

  it('drops the marks of the removed sides when the last point is undone', () => {
    const marked = [0, 1, 2, 3].reduce(
      (survey, side) => surveyWithObstruction(survey, side, 'trees'),
      square(),
    )

    expect(surveyAfterRemoveLast(marked).obstructions).toEqual([
      { side: 0, reason: 'trees' },
      { side: 1, reason: 'trees' },
    ])
  })

  it('keeps a mark with its two points when points are swapped', () => {
    const marked = surveyWithObstruction(surveyWithObstruction(square(), 0, 'trees'), 2, 'unclear')
    const swapped = surveyAfterSwap(marked, 2, 3)

    expect(swapped.obstructions).toEqual([
      { side: 0, reason: 'trees' },
      { side: 2, reason: 'unclear' },
    ])
    expect(surveyAfterSwap(surveyWithObstruction(square(), 1, 'trees'), 0, 2).obstructions).toEqual([
      { side: 0, reason: 'trees' },
    ])
    expect(surveyAfterSwap(surveyWithObstruction(square(), 0, 'trees'), 1, 3).obstructions).toEqual([
      { side: 3, reason: 'trees' },
    ])
  })
})
