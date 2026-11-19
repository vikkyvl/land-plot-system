import { defineStore } from 'pinia'
import { computed, ref, shallowRef } from 'vue'

import type { Coordinate } from '../../domain/coordinate'
import {
  addPoint as appendPoint,
  insertPointAfter,
  movePoint as replacePoint,
  removeLastPoint,
  swapPoints as exchangePoints,
  type PlotDraft,
} from '../../domain/draft'
import { DomainError, isDomainError } from '../../domain/error'
import type { Plot, PlotPreview } from '../../domain/plot'
import type { Problem, SelfIntersectionDetails } from '../../domain/problem'
import {
  emptySurvey,
  surveyAfterAdd,
  surveyAfterInsert,
  surveyAfterMove,
  surveyAfterRemoveLast,
  surveyAfterSwap,
  surveyOfPlot,
  surveyWithObstruction,
  type Gesture,
  type ObstructionReason,
  type Survey,
} from '../../domain/survey'
import { useCalculatorService } from '../composables/services'

export const previewDelay = 300
export const defaultRetryAfterSeconds = 5
const secondMilliseconds = 1000

export interface PreviewOutage {
  readonly rateLimited: boolean
  readonly secondsLeft: number | null
}

export type SaveOutcome = 'saved' | 'updated' | 'sign-in' | 'failed'

export const useCalculatorStore = defineStore('calculator', () => {
  const service = useCalculatorService()

  const points = shallowRef<readonly Coordinate[]>([])
  const name = ref('')
  const survey = shallowRef<Survey>(emptySurvey)
  const preview = shallowRef<PlotPreview | null>(null)
  const previewedVersion = ref(-1)
  const version = ref(0)
  const previewing = ref(false)
  const outage = shallowRef<PreviewOutage | null>(null)
  const errorDismissed = ref(false)
  const saving = ref(false)
  const saveError = shallowRef<DomainError | null>(null)
  const savedPlot = shallowRef<Plot | null>(null)
  const ready = ref(false)
  const editingId = ref<string | null>(null)
  const loadError = shallowRef<DomainError | null>(null)

  let previewTimer: ReturnType<typeof setTimeout> | null = null
  let countdownTimer: ReturnType<typeof setInterval> | null = null
  let requestNumber = 0

  const enoughPoints = computed(() => service.canPreview(points.value))
  const problems = computed<readonly Problem[]>(() => (enoughPoints.value ? (preview.value?.problems ?? []) : []))
  const measurement = computed(() => (enoughPoints.value && preview.value?.valid ? preview.value.measurement : null))
  const accuracy = computed(() => (measurement.value === null ? null : (preview.value?.accuracy ?? null)))
  const upToDate = computed(() => previewedVersion.value === version.value && !previewing.value)
  const updating = computed(() => enoughPoints.value && !upToDate.value && outage.value === null)
  const selfIntersection = computed<SelfIntersectionDetails | null>(() => {
    for (const problem of problems.value) {
      if (problem.details?.kind === 'self_intersection') {
        return problem.details
      }
    }

    return null
  })
  const flaggedVertices = computed(() =>
    problems.value.flatMap((problem) => {
      switch (problem.details?.kind) {
        case 'duplicate_point':
          return [...problem.details.indices]
        case 'invalid_coordinate':
          return [problem.details.index]
        default:
          return []
      }
    }),
  )
  const canSave = computed(
    () => measurement.value !== null && upToDate.value && outage.value === null && !saving.value,
  )

  function draft(): PlotDraft {
    return { name: name.value, points: points.value, survey: survey.value, saveAfterSignIn: false }
  }

  async function startEditing(plotId: string): Promise<void> {
    editingId.value = plotId
    loadError.value = null
    savedPlot.value = null
    saveError.value = null
    points.value = []
    name.value = ''
    survey.value = emptySurvey
    ready.value = false
    try {
      const plot = await service.loadPlot(plotId)
      points.value = plot.points
      survey.value = surveyOfPlot(plot.points.length, plot.capture, plot.obstructions)
      name.value = plot.name
      version.value += 1
      ready.value = true
      requestPreview(0)
    } catch (error) {
      loadError.value = asDomainError(error)
    }
  }

  async function start(): Promise<Plot | null> {
    editingId.value = null
    loadError.value = null
    const stored = await service.loadDraft()
    points.value = stored.points
    survey.value = stored.survey
    name.value = stored.name
    ready.value = true
    requestPreview(0)

    try {
      const plot = await service.savePendingDraft()
      if (plot !== null) {
        finishSave(plot)
      }

      return plot
    } catch (error) {
      saveError.value = asDomainError(error)
      const restored = await service.loadDraft()
      name.value = restored.name

      return null
    }
  }

  function change(next: readonly Coordinate[], nextSurvey: Survey): void {
    points.value = next
    survey.value = nextSurvey
    version.value += 1
    errorDismissed.value = false
    savedPlot.value = null
    saveError.value = null
    keepDraft()
    requestPreview(previewDelay)
  }

  function rename(value: string): void {
    name.value = value
    saveError.value = null
    keepDraft()
  }

  function keepDraft(): void {
    if (editingId.value === null) {
      void service.keepDraft(draft())
    }
  }

  function requestPreview(delay: number): void {
    if (previewTimer !== null) {
      clearTimeout(previewTimer)
      previewTimer = null
    }
    stopCountdown()
    outage.value = null

    if (!enoughPoints.value) {
      requestNumber += 1
      preview.value = null
      previewing.value = false
      previewedVersion.value = version.value

      return
    }

    previewTimer = setTimeout(() => {
      previewTimer = null
      void runPreview()
    }, delay)
  }

  async function runPreview(): Promise<void> {
    requestNumber += 1
    const request = requestNumber
    const requestedVersion = version.value
    previewing.value = true

    try {
      const result = await service.preview(points.value, survey.value)
      if (request !== requestNumber) {
        return
      }
      preview.value = result
      previewedVersion.value = requestedVersion
    } catch (error) {
      if (request !== requestNumber) {
        return
      }
      showOutage(asDomainError(error))
    } finally {
      if (request === requestNumber) {
        previewing.value = false
      }
    }
  }

  function showOutage(error: DomainError): void {
    const rateLimited = error.kind === 'rate_limited'
    const secondsLeft = rateLimited ? (error.retryAfterSeconds ?? defaultRetryAfterSeconds) : null
    outage.value = { rateLimited, secondsLeft }

    if (secondsLeft === null) {
      return
    }

    countdownTimer = setInterval(() => {
      const left = (outage.value?.secondsLeft ?? 1) - 1
      if (left <= 0) {
        requestPreview(0)

        return
      }
      outage.value = { rateLimited, secondsLeft: left }
    }, secondMilliseconds)
  }

  function stopCountdown(): void {
    if (countdownTimer !== null) {
      clearInterval(countdownTimer)
      countdownTimer = null
    }
  }

  function retryPreview(): void {
    requestPreview(0)
  }

  async function save(signedIn: boolean): Promise<SaveOutcome> {
    if (editingId.value !== null) {
      return update(editingId.value)
    }
    if (!signedIn) {
      await service.saveAfterSignIn(draft())

      return 'sign-in'
    }

    saving.value = true
    saveError.value = null
    try {
      finishSave(await service.save(draft()))

      return 'saved'
    } catch (error) {
      const failure = asDomainError(error)
      if (failure.kind === 'unauthorized') {
        await service.saveAfterSignIn(draft())

        return 'sign-in'
      }
      saveError.value = failure

      return 'failed'
    } finally {
      saving.value = false
    }
  }

  async function update(plotId: string): Promise<SaveOutcome> {
    saving.value = true
    saveError.value = null
    try {
      savedPlot.value = await service.update(plotId, draft())

      return 'updated'
    } catch (error) {
      const failure = asDomainError(error)
      if (failure.kind === 'unauthorized') {
        return 'sign-in'
      }
      saveError.value = failure

      return 'failed'
    } finally {
      saving.value = false
    }
  }

  function finishSave(plot: Plot): void {
    savedPlot.value = plot
    points.value = []
    survey.value = emptySurvey
    name.value = ''
    version.value += 1
    requestPreview(0)
  }

  function stop(): void {
    if (previewTimer !== null) {
      clearTimeout(previewTimer)
      previewTimer = null
    }
    stopCountdown()
  }

  return {
    points,
    name,
    survey,
    preview,
    previewing,
    outage,
    errorDismissed,
    saving,
    saveError,
    savedPlot,
    ready,
    editingId,
    loadError,
    enoughPoints,
    problems,
    measurement,
    accuracy,
    updating,
    selfIntersection,
    flaggedVertices,
    canSave,
    start,
    startEditing,
    stop,
    rename,
    retryPreview,
    save,
    addPoint: (point: Coordinate, gesture: Gesture) =>
      change(appendPoint(points.value, point), surveyAfterAdd(survey.value, gesture)),
    movePoint: (index: number, point: Coordinate, gesture: Gesture) =>
      change(replacePoint(points.value, index, point), surveyAfterMove(survey.value, index, gesture)),
    insertPoint: (afterIndex: number, point: Coordinate, gesture: Gesture) =>
      change(insertPointAfter(points.value, afterIndex, point), surveyAfterInsert(survey.value, afterIndex, gesture)),
    undo: () => change(removeLastPoint(points.value), surveyAfterRemoveLast(survey.value)),
    clear: () => change([], emptySurvey),
    swap: (first: number, second: number) =>
      change(exchangePoints(points.value, first, second), surveyAfterSwap(survey.value, first, second)),
    markSide: (side: number, reason: ObstructionReason | null) =>
      change(points.value, surveyWithObstruction(survey.value, side, reason)),
    dismissError: () => {
      errorDismissed.value = true
    },
  }
})

function asDomainError(error: unknown): DomainError {
  return isDomainError(error)
    ? error
    : new DomainError({ kind: 'internal', code: 'internal_error', message: String(error) })
}
