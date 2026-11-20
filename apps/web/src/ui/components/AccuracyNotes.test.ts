import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { Accuracy } from '../../domain/accuracy'
import { formatMargin } from '../composables/accuracyText'
import { applyLocale } from '../i18n'

import AccuracyNotes from './AccuracyNotes.vue'

const fromScale: Accuracy = {
  marginSquareMeters: 508,
  marginHectares: 0.0508,
  percent: 2.1,
  confidence: 0.95,
  scaleShare: 1,
  obstructionShare: 0,
  worstSide: null,
  groundShiftMeters: 2,
  basis: 'capture',
}

const fromTrees: Accuracy = { ...fromScale, scaleShare: 0.22, obstructionShare: 0.78, worstSide: 1, groundShiftMeters: 4 }

function notes(accuracy: Accuracy, zoomHint = true) {
  return mount(AccuracyNotes, {
    props: { accuracy, obstructions: [{ side: 1, reason: 'trees' }], pointCount: 4, zoomHint },
  }).text()
}

describe('AccuracyNotes', () => {
  it('shows the margin, the main source, the zoom hint and the ground shift', () => {
    expect(formatMargin(fromScale)).toBe('± 0,05 га')
    expect(notes(fromScale)).toContain('Найбільше впливає: масштаб карти')
    expect(notes(fromScale)).toContain('Наблизьте карту, щоб зменшити похибку')
    expect(notes(fromScale)).toContain('Можливий зсув на місцевості ± 2 м')
    expect(notes(fromScale, false)).not.toContain('Наблизьте карту')
  })

  it('names the marked side when it adds the most', () => {
    const text = notes(fromTrees)

    expect(text).toContain('Найбільше впливає: дерева на стороні 2–3')
    expect(text).not.toContain('Наблизьте карту')
  })

  it('marks an assumed margin as approximate and asks to change the boundary', () => {
    const assumed: Accuracy = { ...fromScale, marginHectares: 0.54, basis: 'assumed' }

    expect(formatMargin(assumed)).toBe('≈ ± 0,54 га')
    expect(notes(assumed)).toContain('(орієнтовно: ділянку збережено без даних про масштаб)')
    expect(notes(assumed)).toContain('Змініть межу, щоб уточнити похибку')
    expect(notes(assumed)).not.toContain('Наблизьте карту')
  })

  it('shows a small margin in square metres and speaks English', () => {
    applyLocale('en')

    expect(formatMargin({ ...fromScale, marginHectares: 0.0045, marginSquareMeters: 45 })).toBe('± 45 m²')
    expect(notes(fromTrees)).toContain('Affected most by: trees on side 2–3')
    expect(notes(fromTrees)).toContain('Possible shift on the ground ± 4 m')
  })
})
