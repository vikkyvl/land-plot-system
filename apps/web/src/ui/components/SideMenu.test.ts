import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'

import PointsCard from './PointsCard.vue'
import SideMenu from './SideMenu.vue'

afterEach(() => {
  document.body.innerHTML = ''
})

function mountMenu() {
  return mount(SideMenu, {
    attachTo: document.body,
    props: { side: 3, pointCount: 4, reason: 'trees', anchor: { x: 100, y: 100 } },
  })
}

function buttons(): HTMLButtonElement[] {
  return [...document.body.querySelectorAll<HTMLButtonElement>('.side-menu button')]
}

describe('SideMenu', () => {
  it('names the side, starts with adding a point and focuses it', async () => {
    const wrapper = mountMenu()
    await wrapper.vm.$nextTick()

    expect(document.body.querySelector('.side-menu__title')?.textContent?.trim()).toBe('Сторона 4–1')
    expect(buttons().map((button) => button.textContent?.trim())).toEqual([
      '+ Додати точку тут',
      'Нічого',
      'Дерева',
      'Будівлі й тіні',
      'Погано видно межу',
      'Скасувати',
    ])
    expect(document.activeElement).toBe(buttons()[0])
    expect(buttons()[2]?.getAttribute('aria-pressed')).toBe('true')
    wrapper.unmount()
  })

  it('emits the insertion, the chosen reason and closing', async () => {
    const wrapper = mountMenu()

    buttons()[0]?.click()
    buttons()[1]?.click()
    buttons()[4]?.click()
    document.body.querySelector('.side-menu')?.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))

    expect(wrapper.emitted('insert')).toHaveLength(1)
    expect(wrapper.emitted('mark')).toEqual([[null], ['unclear']])
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })
})

describe('PointsCard sides', () => {
  const points = [
    { latitude: 50.45, longitude: 30.52 },
    { latitude: 50.451, longitude: 30.523 },
    { latitude: 50.449, longitude: 30.524 },
  ]

  it('lists every side with its mark for keyboard and screen reader users', async () => {
    const wrapper = mount(PointsCard, {
      props: { points, flagged: [], obstructions: [{ side: 2, reason: 'buildings' }] },
    })
    const selects = wrapper.findAll('select')

    expect(wrapper.findAll('.points-card__side label').map((label) => label.text())).toEqual([
      'Сторона 1–2',
      'Сторона 2–3',
      'Сторона 3–1',
    ])
    expect(selects[2]?.attributes('aria-label')).toBe('Межу сторони 3–1 закривають')
    expect((selects[2]?.element as HTMLSelectElement).value).toBe('buildings')

    await selects[0]?.setValue('trees')
    await selects[2]?.setValue('')
    expect(wrapper.emitted('mark')).toEqual([
      [0, 'trees'],
      [2, null],
    ])
  })
})
