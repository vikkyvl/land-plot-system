import { defineStore } from 'pinia'
import { ref } from 'vue'

import { isDomainError } from '../../domain/error'
import type { Plot, PlotChanges } from '../../domain/plot'
import { useConfirmationService } from '../composables/services'
import { t } from '../i18n'

import { useToastStore } from './toast'

export const pollInterval = 2000
export const maximumAttempts = 30

export type ConfirmationPhase = 'pending' | 'confirmed' | 'reverted' | 'cancelled' | 'gone' | 'stalled'

export interface ConfirmationEntry {
  readonly phase: ConfirmationPhase
  readonly plot: Plot | null
}

export interface ConfirmationContext {
  readonly intent: 'create' | 'update'
  readonly name: string
  readonly expected?: PlotChanges
}

interface Poller {
  subscribers: number
  attempts: number
  timer: ReturnType<typeof setTimeout> | null
  checking: boolean
  active: boolean
}

export const useConfirmationsStore = defineStore('confirmations', () => {
  const service = useConfirmationService()
  const toasts = useToastStore()
  const entries = ref<Record<string, ConfirmationEntry>>({})
  const pollers = new Map<string, Poller>()
  const contexts = new Map<string, ConfirmationContext>()
  let listening = false

  function hidden(): boolean {
    return typeof document !== 'undefined' && document.visibilityState === 'hidden'
  }

  function listen(): void {
    if (listening || typeof document === 'undefined') {
      return
    }
    listening = true
    document.addEventListener('visibilitychange', () => {
      for (const [id, poller] of pollers) {
        if (hidden()) {
          clearTimer(poller)
        } else if (poller.active && poller.subscribers > 0 && !poller.checking) {
          void check(id)
        }
      }
    })
  }

  function clearTimer(poller: Poller): void {
    if (poller.timer !== null) {
      clearTimeout(poller.timer)
      poller.timer = null
    }
  }

  function schedule(id: string, poller: Poller): void {
    clearTimer(poller)
    if (!poller.active || poller.subscribers === 0 || hidden()) {
      return
    }
    poller.timer = setTimeout(() => {
      poller.timer = null
      void check(id)
    }, pollInterval)
  }

  function finish(id: string, poller: Poller, entry: ConfirmationEntry): void {
    poller.active = false
    clearTimer(poller)
    entries.value = { ...entries.value, [id]: entry }
  }

  async function check(id: string): Promise<void> {
    const poller = pollers.get(id)
    if (poller === undefined || !poller.active || poller.checking || hidden()) {
      return
    }
    poller.checking = true
    poller.attempts += 1
    const context = contexts.get(id)
    try {
      const result = await service.check(id, context?.expected)
      switch (result.state) {
        case 'pending':
          entries.value = { ...entries.value, [id]: { phase: 'pending', plot: result.plot } }
          break
        case 'confirmed':
          finish(id, poller, { phase: 'confirmed', plot: result.plot })
          toasts.show(t('confirmations.saved', { name: result.plot.name }))
          contexts.delete(id)
          break
        case 'reverted':
          finish(id, poller, { phase: 'reverted', plot: result.plot })
          toasts.show(t('confirmations.reverted', { version: result.plot.version, name: result.plot.name }), 'error')
          contexts.delete(id)
          break
        case 'cancelled':
          finish(id, poller, { phase: 'cancelled', plot: null })
          toasts.show(cancelledText(context), 'error')
          contexts.delete(id)
          break
        case 'gone':
          finish(id, poller, { phase: 'gone', plot: null })
          contexts.delete(id)
          break
      }
    } catch (error) {
      if (isDomainError(error) && error.kind === 'unauthorized') {
        poller.active = false
      }
    } finally {
      poller.checking = false
    }

    if (poller.active && poller.attempts >= maximumAttempts) {
      finish(id, poller, { phase: 'stalled', plot: entries.value[id]?.plot ?? null })

      return
    }
    schedule(id, poller)
  }

  function start(id: string, poller: Poller): void {
    poller.active = true
    poller.attempts = 0
    entries.value = { ...entries.value, [id]: { phase: 'pending', plot: entries.value[id]?.plot ?? null } }
    schedule(id, poller)
  }

  function expect(id: string, context: ConfirmationContext): void {
    contexts.set(id, context)
    const poller = pollers.get(id)
    if (poller !== undefined && poller.subscribers > 0 && !poller.active) {
      start(id, poller)
    }
  }

  function follow(id: string): () => void {
    listen()
    const poller = pollers.get(id) ?? { subscribers: 0, attempts: 0, timer: null, checking: false, active: false }
    pollers.set(id, poller)
    poller.subscribers += 1
    if (!poller.active) {
      start(id, poller)
    } else if (poller.timer === null && !poller.checking) {
      schedule(id, poller)
    }

    let released = false

    return () => {
      if (released) {
        return
      }
      released = true
      poller.subscribers -= 1
      if (poller.subscribers === 0) {
        clearTimer(poller)
      }
    }
  }

  function retry(id: string): void {
    const poller = pollers.get(id)
    if (poller !== undefined && poller.subscribers > 0) {
      start(id, poller)
      void check(id)
    }
  }

  function isPolling(id: string): boolean {
    const poller = pollers.get(id)

    return poller !== undefined && poller.active && poller.subscribers > 0
  }

  return { entries, expect, follow, retry, isPolling }
})

function cancelledText(context: ConfirmationContext | undefined): string {
  const update = context?.intent === 'update'
  if (!context?.name) {
    return t(update ? 'confirmations.updateCancelledUnnamed' : 'confirmations.createCancelledUnnamed')
  }

  return t(update ? 'confirmations.updateCancelled' : 'confirmations.createCancelled', { name: context.name })
}
