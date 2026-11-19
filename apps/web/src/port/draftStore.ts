import type { PlotDraft } from '../domain/draft'

export interface DraftStore {
  load(): Promise<PlotDraft | null>
  save(draft: PlotDraft): Promise<void>
  clear(): Promise<void>
}
