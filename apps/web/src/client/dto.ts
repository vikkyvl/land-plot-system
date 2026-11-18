import type { Accuracy, AccuracyBasis } from '../domain/accuracy'
import type { Coordinate } from '../domain/coordinate'
import type { MapLayerId } from '../domain/mapLayer'
import type { Measurement } from '../domain/measurement'
import type { Plot, PlotPreview, PlotStatus, PlotVersion } from '../domain/plot'
import type { Profile } from '../domain/profile'
import type { Report } from '../domain/report'
import { parseMonthDay, type Climate, type DataStatus, type Season, type Terrain } from '../domain/surroundings'
import type { Capture, Obstruction, ObstructionReason, PointerKind } from '../domain/survey'

import { toProblem, type ProblemDto } from './errors'

export interface CoordinateDto {
  readonly lat: number
  readonly lon: number
}

interface MeasurementDto {
  readonly area_m2: number
  readonly area_ares: number
  readonly area_hectares: number
  readonly perimeter_m: number
}

export interface CaptureDto {
  readonly zoom: number
  readonly layer: MapLayerId
  readonly pointer: PointerKind
}

export interface ObstructionDto {
  readonly side: number
  readonly reason: ObstructionReason
}

export interface AccuracyDto {
  readonly area_m2_margin: number
  readonly area_hectares_margin: number
  readonly percent: number
  readonly confidence: number
  readonly contributions: { readonly scale: number; readonly obstructions: number }
  readonly worst_side: number | null
  readonly ground_shift_m: number
  readonly basis?: AccuracyBasis
}

export interface PlotDto extends MeasurementDto {
  readonly id: string
  readonly owner_id: string
  readonly name: string
  readonly points: CoordinateDto[]
  readonly capture?: CaptureDto | null
  readonly obstructions?: ObstructionDto[] | null
  readonly accuracy?: AccuracyDto | null
  readonly version: number
  readonly status: PlotStatus
  readonly created_at: string
  readonly updated_at: string
}

export interface PlotVersionDto extends MeasurementDto {
  readonly version: number
  readonly name: string
  readonly points: CoordinateDto[]
  readonly obstructions?: ObstructionDto[] | null
  readonly accuracy?: AccuracyDto | null
  readonly created_at: string
}

export interface PlotPreviewDto {
  readonly area_m2: number | null
  readonly area_ares: number | null
  readonly area_hectares: number | null
  readonly perimeter_m: number | null
  readonly accuracy?: AccuracyDto | null
  readonly valid: boolean
  readonly errors: ProblemDto[] | null
}

export interface TerrainDto {
  readonly status: DataStatus
  readonly slope_degrees: number
  readonly surface_area_m2: number
  readonly surface_area_hectares: number
  readonly surface_increase_percent: number
  readonly min_elevation_m: number
  readonly max_elevation_m: number
  readonly relief_m: number
  readonly coarse: boolean
}

export interface SeasonDto {
  readonly season: Season
  readonly mean_c: number
  readonly mean_min_c: number
  readonly mean_max_c: number
  readonly precipitation_mm: number
  readonly frost_days: number
  readonly heat_days: number
}

export interface ClimateDto {
  readonly status: DataStatus
  readonly first_year: number
  readonly last_year: number
  readonly seasons: SeasonDto[] | null
  readonly last_spring_frost: string | null
  readonly first_autumn_frost: string | null
  readonly growing_days: number
}

export interface ReportDto extends MeasurementDto {
  readonly id: string
  readonly plot_id: string
  readonly plot_name: string
  readonly plot_version: number
  readonly created_at: string
  readonly plot_deleted: boolean
  readonly accuracy?: AccuracyDto | null
  readonly terrain?: TerrainDto | null
  readonly climate?: ClimateDto | null
}

export interface ProfileDto {
  readonly sub: string
  readonly email?: string
  readonly name?: string
  readonly scopes: string[] | null
  readonly created_at: string
  readonly updated_at: string
}

export function toCoordinate(dto: CoordinateDto): Coordinate {
  return { latitude: dto.lat, longitude: dto.lon }
}

export function toCoordinateDto(coordinate: Coordinate): CoordinateDto {
  return { lat: coordinate.latitude, lon: coordinate.longitude }
}

export function toAccuracy(dto: AccuracyDto | null | undefined): Accuracy | null {
  if (dto === null || dto === undefined) {
    return null
  }

  return {
    marginSquareMeters: dto.area_m2_margin,
    marginHectares: dto.area_hectares_margin,
    percent: dto.percent,
    confidence: dto.confidence,
    scaleShare: dto.contributions.scale,
    obstructionShare: dto.contributions.obstructions,
    worstSide: dto.worst_side,
    groundShiftMeters: dto.ground_shift_m,
    basis: dto.basis ?? 'capture',
  }
}

function toCapture(dto: CaptureDto | null | undefined): Capture | null {
  return dto === null || dto === undefined ? null : { zoom: dto.zoom, layer: dto.layer, pointer: dto.pointer }
}

function toObstructions(dtos: ObstructionDto[] | null | undefined): Obstruction[] {
  return (dtos ?? []).map(({ side, reason }) => ({ side, reason }))
}

export function toCaptureDto(capture: Capture): CaptureDto {
  return { zoom: capture.zoom, layer: capture.layer, pointer: capture.pointer }
}

export function toObstructionDtos(obstructions: readonly Obstruction[]): ObstructionDto[] {
  return obstructions.map(({ side, reason }) => ({ side, reason }))
}

function toMeasurement(dto: MeasurementDto): Measurement {
  return {
    area: { squareMeters: dto.area_m2, ares: dto.area_ares, hectares: dto.area_hectares },
    perimeterMeters: dto.perimeter_m,
  }
}

export function toPlot(dto: PlotDto): Plot {
  return {
    id: dto.id,
    ownerId: dto.owner_id,
    name: dto.name,
    points: dto.points.map(toCoordinate),
    measurement: toMeasurement(dto),
    capture: toCapture(dto.capture),
    obstructions: toObstructions(dto.obstructions),
    accuracy: toAccuracy(dto.accuracy),
    version: dto.version,
    status: dto.status,
    createdAt: new Date(dto.created_at),
    updatedAt: new Date(dto.updated_at),
  }
}

export function toPlotVersion(dto: PlotVersionDto): PlotVersion {
  return {
    version: dto.version,
    name: dto.name,
    points: dto.points.map(toCoordinate),
    measurement: toMeasurement(dto),
    obstructions: toObstructions(dto.obstructions),
    accuracy: toAccuracy(dto.accuracy),
    createdAt: new Date(dto.created_at),
  }
}

export function toPlotPreview(dto: PlotPreviewDto): PlotPreview {
  const { area_m2, area_ares, area_hectares, perimeter_m } = dto
  const measured = area_m2 !== null && area_ares !== null && area_hectares !== null && perimeter_m !== null

  return {
    valid: dto.valid,
    measurement: measured ? toMeasurement({ area_m2, area_ares, area_hectares, perimeter_m }) : null,
    accuracy: measured ? toAccuracy(dto.accuracy) : null,
    problems: (dto.errors ?? []).map(toProblem),
  }
}

export function toReport(dto: ReportDto): Report {
  return {
    id: dto.id,
    plotId: dto.plot_id,
    plotName: dto.plot_name,
    plotVersion: dto.plot_version,
    measurement: toMeasurement(dto),
    createdAt: new Date(dto.created_at),
    plotDeleted: dto.plot_deleted,
    accuracy: toAccuracy(dto.accuracy),
    terrain: toTerrain(dto.terrain),
    climate: toClimate(dto.climate),
  }
}

function toTerrain(dto: TerrainDto | null | undefined): Terrain | null {
  if (dto === null || dto === undefined) {
    return null
  }

  return {
    status: dto.status,
    slopeDegrees: dto.slope_degrees,
    surfaceSquareMeters: dto.surface_area_m2,
    surfaceHectares: dto.surface_area_hectares,
    surfaceIncreasePercent: dto.surface_increase_percent,
    minElevationMeters: dto.min_elevation_m,
    maxElevationMeters: dto.max_elevation_m,
    reliefMeters: dto.relief_m,
    coarse: dto.coarse,
  }
}

function toClimate(dto: ClimateDto | null | undefined): Climate | null {
  if (dto === null || dto === undefined) {
    return null
  }

  return {
    status: dto.status,
    firstYear: dto.first_year,
    lastYear: dto.last_year,
    seasons: (dto.seasons ?? []).map((season) => ({
      season: season.season,
      meanCelsius: season.mean_c,
      meanMinCelsius: season.mean_min_c,
      meanMaxCelsius: season.mean_max_c,
      precipitationMillimeters: season.precipitation_mm,
      frostDays: season.frost_days,
      heatDays: season.heat_days,
    })),
    lastSpringFrost: parseMonthDay(dto.last_spring_frost),
    firstAutumnFrost: parseMonthDay(dto.first_autumn_frost),
    growingDays: dto.growing_days,
  }
}

export function toProfile(dto: ProfileDto): Profile {
  return {
    subject: dto.sub,
    email: dto.email ?? null,
    name: dto.name ?? null,
    scopes: dto.scopes ?? [],
    createdAt: new Date(dto.created_at),
    updatedAt: new Date(dto.updated_at),
  }
}
