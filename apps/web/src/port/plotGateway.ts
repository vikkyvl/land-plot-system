import type { Coordinate } from '../domain/coordinate'
import type { NewPlot, Plot, PlotChanges, PlotPreview, PlotVersion } from '../domain/plot'

export interface PlotGateway {
  list(): Promise<Plot[]>
  get(id: string): Promise<Plot>
  create(plot: NewPlot): Promise<Plot>
  update(id: string, changes: PlotChanges): Promise<Plot>
  remove(id: string): Promise<void>
  versions(id: string): Promise<PlotVersion[]>
  preview(points: readonly Coordinate[]): Promise<PlotPreview>
}
