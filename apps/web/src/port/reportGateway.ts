import type { Locale } from '../domain/locale'
import type { Report, ReportFormat, ReportQuery } from '../domain/report'

export interface ReportFile {
  readonly content: Blob
  readonly fileName: string | null
}

export interface ReportGateway {
  download(id: string, format: ReportFormat, locale: Locale): Promise<ReportFile>
  list(query?: ReportQuery): Promise<Report[]>
  get(id: string): Promise<Report>
  create(plotId: string): Promise<Report>
}
