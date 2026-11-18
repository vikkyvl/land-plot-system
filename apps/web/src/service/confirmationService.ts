import { isDomainError } from '../domain/error'
import { isPending, reflectsChanges, type Plot, type PlotChanges } from '../domain/plot'
import type { PlotGateway } from '../port/plotGateway'

export type ConfirmationCheck =
  | { readonly state: 'pending'; readonly plot: Plot }
  | { readonly state: 'confirmed'; readonly plot: Plot }
  | { readonly state: 'reverted'; readonly plot: Plot }
  | { readonly state: 'cancelled' }
  | { readonly state: 'gone' }

export class ConfirmationService {
  private readonly plots: PlotGateway

  constructor(plots: PlotGateway) {
    this.plots = plots
  }

  async check(id: string, expected?: PlotChanges): Promise<ConfirmationCheck> {
    let plot: Plot
    try {
      plot = await this.plots.get(id)
    } catch (error) {
      if (isDomainError(error) && error.kind === 'not_found') {
        return { state: 'cancelled' }
      }
      if (isDomainError(error) && error.kind === 'gone') {
        return { state: 'gone' }
      }
      throw error
    }

    if (isPending(plot)) {
      return { state: 'pending', plot }
    }
    if (expected !== undefined && !reflectsChanges(plot, expected)) {
      return { state: 'reverted', plot }
    }

    return { state: 'confirmed', plot }
  }
}
