import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Toast {
  readonly id: number
  readonly text: string
  readonly tone: 'info' | 'error'
}

export const toastLifetime = 4000

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])
  let nextId = 1

  function show(text: string, tone: Toast['tone'] = 'info'): void {
    const toast = { id: nextId, text, tone }
    nextId += 1
    toasts.value = [...toasts.value, toast]
    setTimeout(() => dismiss(toast.id), toastLifetime)
  }

  function dismiss(id: number): void {
    toasts.value = toasts.value.filter((toast) => toast.id !== id)
  }

  return { toasts, show, dismiss }
})
