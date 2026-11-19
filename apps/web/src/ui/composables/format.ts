import type { Coordinate } from '../../domain/coordinate'
import { intlLocale, t } from '../i18n'

const coordinateDigits = 5

function decimal(value: number, digits: number): string {
  return value.toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

export function formatHectareNumber(value: number): string {
  return decimal(value, 2)
}

export function formatHectares(value: number): string {
  return t('units.hectares', { value: formatHectareNumber(value) })
}

export function formatAres(value: number): string {
  return decimal(value, 1)
}

export function formatSquareMeters(value: number): string {
  return decimal(Math.round(value), 0)
}

export function formatMeters(value: number): string {
  return t('units.meters', { value: decimal(Math.round(value), 0) })
}

export function formatCoordinate(point: Coordinate): string {
  return `${point.latitude.toFixed(coordinateDigits)}, ${point.longitude.toFixed(coordinateDigits)}`
}

export function formatPointCount(count: number): string {
  return t('counts.points', count)
}

export function formatDate(date: Date): string {
  return date.toLocaleDateString(intlLocale(), { day: '2-digit', month: '2-digit', year: 'numeric' })
}

export function formatPlotCount(count: number): string {
  return t('counts.plots', count)
}

export function formatVersionCount(count: number): string {
  return t('counts.versions', count)
}

export function initials(name: string): string {
  return name
    .split(/[\s@.]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('')
}
