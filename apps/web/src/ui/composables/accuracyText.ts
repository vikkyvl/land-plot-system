import { scaleDominates, type Accuracy } from '../../domain/accuracy'
import { obstructionOn, type Obstruction } from '../../domain/survey'
import { intlLocale, t } from '../i18n'

import { formatHectares, formatSquareMeters } from './format'

const smallestHectareMargin = 0.005

export function formatMargin(accuracy: Accuracy): string {
  const value =
    accuracy.marginHectares >= smallestHectareMargin
      ? formatHectares(accuracy.marginHectares)
      : t('units.squareMetersValue', { value: formatSquareMeters(accuracy.marginSquareMeters) })

  return accuracy.basis === 'assumed' ? t('accuracy.approximate', { value }) : t('accuracy.margin', { value })
}

export function mainSource(accuracy: Accuracy, obstructions: readonly Obstruction[], pointCount: number): string {
  const side = accuracy.worstSide
  const reason = side === null ? null : obstructionOn({ obstructions }, side)

  if (scaleDominates(accuracy) || side === null || reason === null) {
    return t('accuracy.mainSource', { source: t('accuracy.scaleSource') })
  }

  return t('accuracy.mainSource', {
    source: t('accuracy.sideSource', {
      reason: t(`obstructions.${reason}`),
      from: side + 1,
      to: ((side + 1) % pointCount) + 1,
    }),
  })
}

export function groundShift(accuracy: Accuracy): string {
  return t('accuracy.groundShift', { value: Math.round(accuracy.groundShiftMeters).toLocaleString(intlLocale()) })
}
