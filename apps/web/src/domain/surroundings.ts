export type DataStatus = 'available' | 'unavailable'

export interface Terrain {
  readonly status: DataStatus
  readonly slopeDegrees: number
  readonly surfaceSquareMeters: number
  readonly surfaceHectares: number
  readonly surfaceIncreasePercent: number
  readonly minElevationMeters: number
  readonly maxElevationMeters: number
  readonly reliefMeters: number
  readonly coarse: boolean
}

export type Season = 'winter' | 'spring' | 'summer' | 'autumn'

export interface SeasonClimate {
  readonly season: Season
  readonly meanCelsius: number
  readonly meanMinCelsius: number
  readonly meanMaxCelsius: number
  readonly precipitationMillimeters: number
  readonly frostDays: number
  readonly heatDays: number
}

export interface MonthDay {
  readonly month: number
  readonly day: number
}

export interface Climate {
  readonly status: DataStatus
  readonly firstYear: number
  readonly lastYear: number
  readonly seasons: readonly SeasonClimate[]
  readonly lastSpringFrost: MonthDay | null
  readonly firstAutumnFrost: MonthDay | null
  readonly growingDays: number
}

const monthDayPattern = /^(\d{2})-(\d{2})$/

export function parseMonthDay(value: string | null | undefined): MonthDay | null {
  const match = monthDayPattern.exec(value ?? '')
  if (match === null) {
    return null
  }

  return { month: Number(match[1]), day: Number(match[2]) }
}
