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
import { useCalculatorService } from '../composables/services'

export const previewDelay = 300
export const defaultRetryAfterSeconds = 5
const secondMilliseconds = 1000

export interface PreviewOutage {
  readonly rateLimited: boolean
  readonly secondsLeft: number | null
}

export type SaveOutcome = 'saved' | 'sign-in' | 'failed'

export const useCalculatorStore = defineStore('calculator', () => {
  const service = useCalculatorService()

  const points = shallowRef<readonly Coordinate[]>([])
  const name = ref('')
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

  let previewTimer: ReturnType<typeof setTimeout> | null = null
  let countdownTimer: ReturnType<typeof setInterval> | null = null
  let requestNumber = 0

  const enoughPoints = computed(() => service.canPreview(points.value))
  const problems = computed<readonly Problem[]>(() => (enoughPoints.value ? (preview.value?.problems ?? []) : []))
  const measurement = computed(() => (enoughPoints.value && preview.value?.valid ? preview.value.measurement : null))
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
    return { name: name.value, points: points.value, saveAfterSignIn: false }
  }

  async function start(): Promise<Plot | null> {
    const stored = await service.loadDraft()
    points.value = stored.points
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

  function change(next: readonly Coordinate[]): void {
    points.value = next
    version.value += 1
    errorDismissed.value = false
    savedPlot.value = null
    saveError.value = null
    void service.keepDraft(draft())
    requestPreview(previewDelay)
  }

  function rename(value: string): void {
    name.value = value
    saveError.value = null
    void service.keepDraft(draft())
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
      const result = await service.preview(points.value)
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

  function finishSave(plot: Plot): void {
    savedPlot.value = plot
    points.value = []
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
    preview,
    previewing,
    outage,
    errorDismissed,
    saving,
    saveError,
    savedPlot,
    ready,
    enoughPoints,
    problems,
    measurement,
    updating,
    selfIntersection,
    flaggedVertices,
    canSave,
    start,
    stop,
    rename,
    retryPreview,
    save,
    addPoint: (point: Coordinate) => change(appendPoint(points.value, point)),
    movePoint: (index: number, point: Coordinate) => change(replacePoint(points.value, index, point)),
    insertPoint: (afterIndex: number, point: Coordinate) => change(insertPointAfter(points.value, afterIndex, point)),
    undo: () => change(removeLastPoint(points.value)),
    clear: () => change([]),
    swap: (first: number, second: number) => change(exchangePoints(points.value, first, second)),
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
