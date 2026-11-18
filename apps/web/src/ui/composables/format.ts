import type { Coordinate } from '../../domain/coordinate'

const locale = 'uk-UA'
const coordinateDigits = 5

function decimal(value: number, digits: number): string {
  return value.toLocaleString(locale, { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

export function formatHectares(value: number): string {
  return `${decimal(value, 2)} га`
}

export function formatAres(value: number): string {
  return decimal(value, 1)
}

export function formatSquareMeters(value: number): string {
  return decimal(Math.round(value), 0)
}

export function formatMeters(value: number): string {
  return `${decimal(Math.round(value), 0)} м`
}

export function formatCoordinate(point: Coordinate): string {
  return `${point.latitude.toFixed(coordinateDigits)}, ${point.longitude.toFixed(coordinateDigits)}`
}

export function formatPointCount(count: number): string {
  return plural(count, 'точка', 'точки', 'точок')
}

export function formatDate(date: Date): string {
  return date.toLocaleDateString(locale, { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function plural(count: number, one: string, few: string, many: string): string {
  const lastTwo = count % 100
  const last = count % 10
  if (last === 1 && lastTwo !== 11) {
    return `${count} ${one}`
  }
  if (last >= 2 && last <= 4 && (lastTwo < 12 || lastTwo > 14)) {
    return `${count} ${few}`
  }

  return `${count} ${many}`
}

export function formatPlotCount(count: number): string {
  return plural(count, 'ділянка', 'ділянки', 'ділянок')
}

export function formatVersionCount(count: number): string {
  return plural(count, 'версія', 'версії', 'версій')
}

export function initials(name: string): string {
  return name
    .split(/[\s@.]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('')
}
