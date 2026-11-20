import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'

import OfflineBanner from './OfflineBanner.vue'

function setOnline(online: boolean): void {
  Object.defineProperty(navigator, 'onLine', { value: online, configurable: true })
  window.dispatchEvent(new Event(online ? 'online' : 'offline'))
}

afterEach(() => setOnline(true))

describe('OfflineBanner', () => {
  it('appears when the connection is lost and hides when it is back', async () => {
    const wrapper = mount(OfflineBanner)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)

    setOnline(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.get('[role="status"]').text()).toContain("Немає з'єднання.")

    setOnline(true)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
  })
})
