import type { Coordinate } from '../domain/coordinate'
import { emptyDraft, type PlotDraft } from '../domain/draft'
import type { Plot, PlotPreview } from '../domain/plot'
import type { AuthSession } from '../port/authSession'
import type { DraftStore } from '../port/draftStore'
import type { PlotGateway } from '../port/plotGateway'

export const minimumPreviewPoints = 3

export class CalculatorService {
  private readonly plots: PlotGateway
  private readonly drafts: DraftStore
  private readonly session: Pick<AuthSession, 'isSignedIn'>

  constructor(plots: PlotGateway, drafts: DraftStore, session: Pick<AuthSession, 'isSignedIn'>) {
    this.plots = plots
    this.drafts = drafts
    this.session = session
  }

  canPreview(points: readonly Coordinate[]): boolean {
    return points.length >= minimumPreviewPoints
  }

  preview(points: readonly Coordinate[]): Promise<PlotPreview> {
    return this.plots.preview(points)
  }

  async loadDraft(): Promise<PlotDraft> {
    return (await this.drafts.load().catch(() => null)) ?? emptyDraft
  }

  keepDraft(draft: PlotDraft): Promise<void> {
    return this.drafts.save(draft).catch(() => undefined)
  }

  async save(draft: PlotDraft): Promise<Plot> {
    const plot = await this.plots.create({ name: draft.name.trim(), points: draft.points })
    await this.drafts.clear().catch(() => undefined)

    return plot
  }

  saveAfterSignIn(draft: PlotDraft): Promise<void> {
    return this.drafts.save({ ...draft, saveAfterSignIn: true })
  }

  async savePendingDraft(): Promise<Plot | null> {
    const draft = await this.drafts.load().catch(() => null)
    if (draft === null || !draft.saveAfterSignIn || !(await this.session.isSignedIn())) {
      return null
    }

    try {
      return await this.save(draft)
    } catch (error) {
      await this.keepDraft({ ...draft, saveAfterSignIn: false })
      throw error
    }
  }
}
