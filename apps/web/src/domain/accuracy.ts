export type AccuracyBasis = 'capture' | 'assumed'

export interface Accuracy {
  readonly marginSquareMeters: number
  readonly marginHectares: number
  readonly percent: number
  readonly confidence: number
  readonly scaleShare: number
  readonly obstructionShare: number
  readonly worstSide: number | null
  readonly groundShiftMeters: number
  readonly basis: AccuracyBasis
}

export function scaleDominates(accuracy: Accuracy): boolean {
  return accuracy.scaleShare >= accuracy.obstructionShare
}
