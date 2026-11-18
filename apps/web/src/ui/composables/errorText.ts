import { ErrorCode } from '../../domain/errorCode'
import type { DomainError } from '../../domain/error'
import type { Problem, SelfIntersectionDetails } from '../../domain/problem'

export interface ProblemText {
  readonly title: string
  readonly text: string
}

const fallbackMessage = 'Щось пішло не так. Спробуйте ще раз.'

const messages: Readonly<Record<string, string>> = {
  [ErrorCode.InvalidRequest]: 'Запит не вдалося обробити. Оновіть сторінку і спробуйте ще раз.',
  [ErrorCode.InvalidName]: 'Вкажіть назву ділянки.',
  [ErrorCode.InsufficientPoints]: 'Для ділянки потрібно щонайменше 3 різні точки.',
  [ErrorCode.InvalidCoordinate]: 'Одна з точок має неможливі координати.',
  [ErrorCode.DuplicatePoint]: 'Дві точки межі збігаються.',
  [ErrorCode.SelfIntersection]: 'Сторони контуру перетинаються.',
  [ErrorCode.ZeroArea]: 'Точки лежать на одній лінії, тому площа дорівнює нулю.',
  [ErrorCode.TooManyPoints]: 'Забагато точок для однієї ділянки. Максимум 1000.',
  [ErrorCode.NoChanges]: 'Змін немає.',
  [ErrorCode.InvalidPlotId]: 'Ділянку не знайдено.',
  [ErrorCode.InvalidReportId]: 'Звіт не знайдено.',
  [ErrorCode.InvalidQuery]: 'Неправильний запит.',
  [ErrorCode.Unauthorized]: 'Сесія завершилася. Увійдіть ще раз.',
  [ErrorCode.Forbidden]: 'Ця ділянка належить іншому користувачу.',
  [ErrorCode.InsufficientScope]: 'Недостатньо прав для цієї дії.',
  [ErrorCode.OriginNotAllowed]: 'Цей сайт не має доступу до сервера.',
  [ErrorCode.PlotNotFound]: 'Ділянку не знайдено.',
  [ErrorCode.ReportNotFound]: 'Звіт не знайдено.',
  [ErrorCode.RouteNotFound]: 'Сервер не знає такої дії.',
  [ErrorCode.VersionConflict]: 'Ділянку щойно змінили. Оновіть сторінку.',
  [ErrorCode.PlotNotConfirmed]: 'Ділянка ще обробляється.',
  [ErrorCode.PlotDeleted]: 'Ділянку видалено.',
  [ErrorCode.RateLimited]: 'Забагато запитів. Спробуйте трохи пізніше.',
  [ErrorCode.ServiceUnavailable]: 'Сервіс тимчасово недоступний. Спробуйте пізніше.',
  [ErrorCode.PlotServiceUnavailable]: 'Сервіс ділянок тимчасово недоступний. Спробуйте пізніше.',
  [ErrorCode.BadGateway]: 'Сервіс тимчасово недоступний. Спробуйте пізніше.',
  [ErrorCode.InternalError]: fallbackMessage,
  [ErrorCode.NetworkError]: "Немає з'єднання з сервером. Перевірте інтернет.",
}

export function errorText(error: Pick<DomainError, 'code'>): string {
  return messages[error.code] ?? fallbackMessage
}

function oneBased(index: number): number {
  return index + 1
}

export function sidesText(details: SelfIntersectionDetails): string {
  const [first, second] = details.segments

  return `${oneBased(first[0])}–${oneBased(first[1])} і ${oneBased(second[0])}–${oneBased(second[1])}`
}

export function swapText(details: SelfIntersectionDetails): string {
  return `Поміняти точки ${oneBased(details.swap[0])} і ${oneBased(details.swap[1])}`
}

export function problemText(problem: Problem): ProblemText {
  const details = problem.details
  switch (details?.kind) {
    case 'self_intersection':
      return {
        title: 'Сторони контуру перетинаються',
        text: `Сторони ${sidesText(details)} перетинаються. Площу такої фігури порахувати неможливо. Найчастіше точки поставлено не по порядку обходу.`,
      }
    case 'duplicate_point':
      return {
        title: 'Точки повторюються',
        text: `Точки ${oneBased(details.indices[0])} і ${oneBased(details.indices[1])} збігаються. Перемістіть або видаліть одну з них.`,
      }
    case 'invalid_coordinate':
      return {
        title: 'Координати поза межами',
        text: `Точка ${oneBased(details.index)} має неможливі координати. Широта від −90 до 90, довгота від −180 до 180.`,
      }
    default:
      return { title: problemTitles[problem.code] ?? 'Контур не можна порахувати', text: errorText(problem) }
  }
}

const problemTitles: Readonly<Record<string, string>> = {
  [ErrorCode.InsufficientPoints]: 'Замало точок',
  [ErrorCode.ZeroArea]: 'Ділянка без площі',
  [ErrorCode.TooManyPoints]: 'Забагато точок',
  [ErrorCode.SelfIntersection]: 'Сторони контуру перетинаються',
  [ErrorCode.DuplicatePoint]: 'Точки повторюються',
  [ErrorCode.InvalidCoordinate]: 'Координати поза межами',
}
