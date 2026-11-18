export interface Area {
  readonly squareMeters: number
  readonly ares: number
  readonly hectares: number
}

export interface Measurement {
  readonly area: Area
  readonly perimeterMeters: number
}
