import type { Plot, PlotVersion } from '../domain/plot'
import type { Report } from '../domain/report'
import type { PlotGateway } from '../port/plotGateway'
import type { ReportGateway } from '../port/reportGateway'

export class PlotService {
  private readonly plots: PlotGateway
  private readonly reports: ReportGateway

  constructor(plots: PlotGateway, reports: ReportGateway) {
    this.plots = plots
    this.reports = reports
  }

  list(): Promise<Plot[]> {
    return this.plots.list()
  }

  get(id: string): Promise<Plot> {
    return this.plots.get(id)
  }

  versions(id: string): Promise<PlotVersion[]> {
    return this.plots.versions(id)
  }

  rename(id: string, name: string): Promise<Plot> {
    return this.plots.update(id, { name: name.trim() })
  }

  remove(id: string): Promise<void> {
    return this.plots.remove(id)
  }

  createReport(plotId: string): Promise<Report> {
    return this.reports.create(plotId)
  }
}
