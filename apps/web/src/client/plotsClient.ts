import type { Coordinate } from '../domain/coordinate'
import type { NewPlot, Plot, PlotChanges, PlotPreview, PlotVersion, SurveyFacts } from '../domain/plot'
import type { PlotGateway } from '../port/plotGateway'

import {
  toCaptureDto,
  toCoordinateDto,
  toObstructionDtos,
  toPlot,
  toPlotPreview,
  toPlotVersion,
  type CaptureDto,
  type ObstructionDto,
  type PlotDto,
  type PlotPreviewDto,
  type PlotVersionDto,
} from './dto'
import { pathOf, type HttpClient } from './http'

const plotsPath = '/plots'

export class PlotsClient implements PlotGateway {
  private readonly http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  async list(): Promise<Plot[]> {
    const plots = await this.http.request<PlotDto[] | null>('GET', plotsPath)

    return (plots ?? []).map(toPlot)
  }

  async get(id: string): Promise<Plot> {
    return toPlot(await this.http.request<PlotDto>('GET', plotsPath + pathOf(id)))
  }

  async create(plot: NewPlot): Promise<Plot> {
    const body = { name: plot.name, points: plot.points.map(toCoordinateDto), ...surveyBody(plot) }

    return toPlot(await this.http.request<PlotDto>('POST', plotsPath, { body }))
  }

  async update(id: string, changes: PlotChanges): Promise<Plot> {
    const body = {
      name: changes.name,
      points: changes.points?.map(toCoordinateDto),
      capture: changes.capture ? toCaptureDto(changes.capture) : undefined,
      obstructions: changes.obstructions ? toObstructionDtos(changes.obstructions) : undefined,
    }

    return toPlot(await this.http.request<PlotDto>('PATCH', plotsPath + pathOf(id), { body }))
  }

  async remove(id: string): Promise<void> {
    await this.http.request<void>('DELETE', plotsPath + pathOf(id))
  }

  async versions(id: string): Promise<PlotVersion[]> {
    const versions = await this.http.request<PlotVersionDto[] | null>('GET', plotsPath + pathOf(id, 'versions'))

    return (versions ?? []).map(toPlotVersion)
  }

  async preview(points: readonly Coordinate[], facts: SurveyFacts): Promise<PlotPreview> {
    const body = { points: points.map(toCoordinateDto), ...surveyBody(facts) }
    const preview = await this.http.request<PlotPreviewDto>('POST', plotsPath + pathOf('preview'), {
      body,
      authenticated: false,
    })

    return toPlotPreview(preview)
  }
}

interface SurveyBody {
  readonly capture?: CaptureDto
  readonly obstructions: ObstructionDto[]
}

function surveyBody(facts: SurveyFacts): SurveyBody {
  return {
    capture: facts.capture === null ? undefined : toCaptureDto(facts.capture),
    obstructions: toObstructionDtos(facts.obstructions),
  }
}
