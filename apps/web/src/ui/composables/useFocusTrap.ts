import { nextTick, onBeforeUnmount, onMounted, type Ref } from 'vue'

const focusable = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'

export function useFocusTrap(container: Ref<HTMLElement | null>, close: () => void): (event: KeyboardEvent) => void {
  let previous: HTMLElement | null = null

  function elements(): HTMLElement[] {
    return [...(container.value?.querySelectorAll<HTMLElement>(focusable) ?? [])].filter(
      (element) => !element.hasAttribute('disabled'),
    )
  }

  function onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault()
      close()

      return
    }
    if (event.key !== 'Tab') {
      return
    }
    const list = elements()
    const first = list[0]
    const last = list.at(-1)
    if (first === undefined || last === undefined) {
      return
    }
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  onMounted(async () => {
    previous = document.activeElement as HTMLElement | null
    await nextTick()
    const input = container.value?.querySelector<HTMLElement>('input')
    ;(input ?? elements()[0])?.focus()
  })

  onBeforeUnmount(() => previous?.focus())

  return onKeydown
}
