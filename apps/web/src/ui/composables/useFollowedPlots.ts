import { onBeforeUnmount } from 'vue'

import { useConfirmationsStore } from '../stores/confirmations'

export function useFollowedPlots(): { follow: (id: string) => void; releaseAll: () => void } {
  const confirmations = useConfirmationsStore()
  const releases = new Map<string, () => void>()

  function follow(id: string): void {
    if (!releases.has(id)) {
      releases.set(id, confirmations.follow(id))
    }
  }

  function releaseAll(): void {
    for (const release of releases.values()) {
      release()
    }
    releases.clear()
  }

  onBeforeUnmount(releaseAll)

  return { follow, releaseAll }
}
