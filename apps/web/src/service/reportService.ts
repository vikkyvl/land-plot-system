import { isDomainError } from '../domain/error'
import type { Locale } from '../domain/locale'
import type { Report, ReportFormat, ReportQuery } from '../domain/report'
import type { ReportDetails } from '../domain/reportDetails'
import type { PlotGateway } from '../port/plotGateway'
import type { ReportFile, ReportGateway } from '../port/reportGateway'

export class ReportService {
  private readonly reports: ReportGateway
  private readonly plots: PlotGateway

  constructor(reports: ReportGateway, plots: PlotGateway) {
    this.reports = reports
    this.plots = plots
  }

  list(query?: ReportQuery): Promise<Report[]> {
    return this.reports.list(query)
  }

  create(plotId: string): Promise<Report> {
    return this.reports.create(plotId)
  }

  download(id: string, format: ReportFormat, locale: Locale): Promise<ReportFile> {
    return this.reports.download(id, format, locale)
  }

  async details(id: string): Promise<ReportDetails> {
    const report = await this.reports.get(id)
    const stored = report.boundary.length > 0 ? report.boundary : null
    if (report.plotDeleted) {
      return { report, plot: null, points: stored, obstructions: [] }
    }

    try {
      const [plot, versions] = await Promise.all([this.plots.get(report.plotId), this.plots.versions(report.plotId)])
      const version = versions.find((candidate) => candidate.version === report.plotVersion)

      return { report, plot, points: stored ?? version?.points ?? null, obstructions: version?.obstructions ?? [] }
    } catch (error) {
      if (isDomainError(error) && error.kind === 'gone') {
        return { report: { ...report, plotDeleted: true }, plot: null, points: stored, obstructions: [] }
      }
      throw error
    }
  }
}
