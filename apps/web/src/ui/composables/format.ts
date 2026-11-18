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
  const lastTwo = count % 100
  const last = count % 10
  if (last === 1 && lastTwo !== 11) {
    return `${count} точка`
  }
  if (last >= 2 && last <= 4 && (lastTwo < 12 || lastTwo > 14)) {
    return `${count} точки`
  }

  return `${count} точок`
}
