import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'

import DeletePlotDialog from './DeletePlotDialog.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

function mountDialog(props: Partial<{ versionCount: number | null; busy: boolean }> = {}) {
  return mount(DeletePlotDialog, {
    attachTo: document.body,
    props: { name: 'Поле біля ставка', versionCount: 2, busy: false, failure: null, ...props },
  })
}

function dialog(): HTMLElement {
  return document.querySelector('[role="dialog"]') as HTMLElement
}

function button(label: string): HTMLButtonElement {
  return [...dialog().querySelectorAll('button')].find((item) => item.textContent?.trim() === label) as HTMLButtonElement
}

describe('DeletePlotDialog', () => {
  it('is a modal dialog with the agreed text about reports', () => {
    mountDialog()

    expect(dialog().getAttribute('aria-modal')).toBe('true')
    expect(dialog().querySelector('h2')?.textContent?.trim()).toBe('Видалити «Поле біля ставка»?')
    expect(dialog().textContent).toContain('історію вимірювань (2 версії)')
    expect(dialog().textContent).toContain('Звіти по ній залишаться з позначкою «Ділянку видалено»')
  })

  it('deletes only after the name is typed', async () => {
    const wrapper = mountDialog()
    const input = dialog().querySelector('input') as HTMLInputElement

    expect(button('Видалити назавжди').disabled).toBe(true)
    input.value = 'Поле'
    input.dispatchEvent(new Event('input'))
    await wrapper.vm.$nextTick()
    expect(button('Видалити назавжди').disabled).toBe(true)

    input.value = ' Поле біля ставка '
    input.dispatchEvent(new Event('input'))
    await wrapper.vm.$nextTick()
    button('Видалити назавжди').click()

    expect(wrapper.emitted('confirm')).toHaveLength(1)
  })

  it('closes on Escape and on «Скасувати», focusing the name field first', async () => {
    const wrapper = mountDialog()
    await wrapper.vm.$nextTick()

    expect(document.activeElement).toBe(dialog().querySelector('input'))
    dialog().dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    button('Скасувати').click()

    expect(wrapper.emitted('close')).toHaveLength(2)
  })
})
