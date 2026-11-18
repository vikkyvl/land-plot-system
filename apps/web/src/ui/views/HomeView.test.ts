import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import HomeView from './HomeView.vue'

describe('HomeView', () => {
  it('shows the application name', () => {
    expect(mount(HomeView).get('h1').text()).toBe('Arqetra')
  })
})
