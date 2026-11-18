import type { Report, ReportQuery } from '../domain/report'
import type { ReportGateway } from '../port/reportGateway'

import { toReport, type ReportDto } from './dto'
import { pathOf, type HttpClient } from './http'

const reportsPath = '/reports'

export class ReportsClient implements ReportGateway {
  private readonly http: HttpClient

  constructor(http: HttpClient) {
    this.http = http
  }

  async list(query: ReportQuery = {}): Promise<Report[]> {
    const reports = await this.http.request<ReportDto[] | null>('GET', reportsPath, {
      query: { plot_id: query.plotId, limit: query.limit },
    })

    return (reports ?? []).map(toReport)
  }

  async get(id: string): Promise<Report> {
    return toReport(await this.http.request<ReportDto>('GET', reportsPath + pathOf(id)))
  }

  async create(plotId: string): Promise<Report> {
    return toReport(await this.http.request<ReportDto>('POST', reportsPath, { body: { plot_id: plotId } }))
  }
}
