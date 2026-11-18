import type { Coordinate } from '../domain/coordinate'
import type { Measurement } from '../domain/measurement'
import type { Plot, PlotPreview, PlotStatus, PlotVersion } from '../domain/plot'
import type { Profile } from '../domain/profile'
import type { Report } from '../domain/report'

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

export interface PlotDto extends MeasurementDto {
  readonly id: string
  readonly owner_id: string
  readonly name: string
  readonly points: CoordinateDto[]
  readonly version: number
  readonly status: PlotStatus
  readonly created_at: string
  readonly updated_at: string
}

export interface PlotVersionDto extends MeasurementDto {
  readonly version: number
  readonly name: string
  readonly points: CoordinateDto[]
  readonly created_at: string
}

export interface PlotPreviewDto {
  readonly area_m2: number | null
  readonly area_ares: number | null
  readonly area_hectares: number | null
  readonly perimeter_m: number | null
  readonly valid: boolean
  readonly errors: ProblemDto[] | null
}

export interface ReportDto extends MeasurementDto {
  readonly id: string
  readonly plot_id: string
  readonly plot_name: string
  readonly plot_version: number
  readonly created_at: string
  readonly plot_deleted: boolean
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
    createdAt: new Date(dto.created_at),
  }
}

export function toPlotPreview(dto: PlotPreviewDto): PlotPreview {
  const { area_m2, area_ares, area_hectares, perimeter_m } = dto
  const measured = area_m2 !== null && area_ares !== null && area_hectares !== null && perimeter_m !== null

  return {
    valid: dto.valid,
    measurement: measured ? toMeasurement({ area_m2, area_ares, area_hectares, perimeter_m }) : null,
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
