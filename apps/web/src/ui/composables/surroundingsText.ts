import type { MonthDay } from '../../domain/surroundings'
import { intlLocale, t } from '../i18n'

const referenceYear = 2001
const oneDecimal = 1

function number(value: number, digits: number): string {
  return value.toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

export function formatDegrees(value: number): string {
  return `${number(value, oneDecimal)}°`
}

export function formatCelsius(value: number): string {
  return t('surroundings.celsius', { value: number(value, oneDecimal) })
}

export function formatMillimeters(value: number): string {
  return t('surroundings.millimeters', { value: number(value, 0) })
}

export function formatWhole(value: number): string {
  return number(value, 0)
}

export function formatPercent(value: number): string {
  return (value / 100).toLocaleString(intlLocale(), { style: 'percent', maximumFractionDigits: oneDecimal })
}

export function formatMonthDay(monthDay: MonthDay | null): string {
  if (monthDay === null) {
    return t('surroundings.noFrost')
  }

  return new Date(referenceYear, monthDay.month - 1, monthDay.day).toLocaleDateString(intlLocale(), {
    day: 'numeric',
    month: 'long',
  })
}
