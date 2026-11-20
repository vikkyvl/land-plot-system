import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import PlotStatusBadge from './PlotStatusBadge.vue'

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('PlotStatusBadge', () => {
  it('shimmers while pending and shows a check for a while once confirmed', async () => {
    const wrapper = mount(PlotStatusBadge, { props: { status: 'PENDING' } })
    expect(wrapper.get('.status-badge').classes()).toContain('motion-shimmer')

    await wrapper.setProps({ status: 'CONFIRMED' })
    expect(wrapper.get('.status-badge--confirmed').text()).toBe('Підтверджено')
    expect(wrapper.find('.status-badge__check').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(2400)
    expect(wrapper.find('.status-badge').exists()).toBe(false)
  })

  it('shows nothing for a plot that was already confirmed', () => {
    expect(mount(PlotStatusBadge, { props: { status: 'CONFIRMED' } }).find('.status-badge').exists()).toBe(false)
  })
})
