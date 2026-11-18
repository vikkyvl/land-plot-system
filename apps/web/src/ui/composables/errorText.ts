import type { DomainError } from '../../domain/error'
import { ErrorCode } from '../../domain/errorCode'
import type { Problem, SelfIntersectionDetails } from '../../domain/problem'
import { i18n, t } from '../i18n'

export interface ProblemText {
  readonly title: string
  readonly text: string
}

const problemTitles: Readonly<Record<string, string>> = {
  [ErrorCode.InsufficientPoints]: 'problems.insufficientTitle',
  [ErrorCode.ZeroArea]: 'problems.zeroAreaTitle',
  [ErrorCode.TooManyPoints]: 'problems.tooManyTitle',
  [ErrorCode.SelfIntersection]: 'problems.selfIntersectionTitle',
  [ErrorCode.DuplicatePoint]: 'problems.duplicateTitle',
  [ErrorCode.InvalidCoordinate]: 'problems.coordinateTitle',
}

export function errorText(error: Pick<DomainError, 'code'>): string {
  const key = `errors.${error.code}`

  return i18n.global.te(key) ? t(key) : t('errors.fallback')
}

function oneBased(index: number): number {
  return index + 1
}

export function sidesText(details: SelfIntersectionDetails): string {
  const [first, second] = details.segments

  return t('problems.sides', {
    a: oneBased(first[0]),
    b: oneBased(first[1]),
    c: oneBased(second[0]),
    d: oneBased(second[1]),
  })
}

export function swapText(details: SelfIntersectionDetails): string {
  return t('problems.swap', { first: oneBased(details.swap[0]), second: oneBased(details.swap[1]) })
}

export function problemText(problem: Problem): ProblemText {
  const details = problem.details
  switch (details?.kind) {
    case 'self_intersection':
      return {
        title: t('problems.selfIntersectionTitle'),
        text: t('problems.selfIntersectionText', { sides: sidesText(details) }),
      }
    case 'duplicate_point':
      return {
        title: t('problems.duplicateTitle'),
        text: t('problems.duplicateText', {
          first: oneBased(details.indices[0]),
          second: oneBased(details.indices[1]),
        }),
      }
    case 'invalid_coordinate':
      return {
        title: t('problems.coordinateTitle'),
        text: t('problems.coordinateText', { index: oneBased(details.index) }),
      }
    default:
      return { title: t(problemTitles[problem.code] ?? 'problems.genericTitle'), text: errorText(problem) }
  }
}
