import type { Report, ReportQuery } from '../domain/report'

export interface ReportGateway {
  list(query?: ReportQuery): Promise<Report[]>
  get(id: string): Promise<Report>
  create(plotId: string): Promise<Report>
}
