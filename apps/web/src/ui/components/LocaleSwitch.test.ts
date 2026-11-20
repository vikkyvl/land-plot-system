import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

import type { SettingsService } from '../../service/settingsService'
import { formatDate, formatHectares, formatPointCount } from '../composables/format'
import { settingsServiceKey } from '../composables/services'
import { ukrainianPlural } from '../i18n'

import LocaleSwitch from './LocaleSwitch.vue'

const Probe = defineComponent({
  components: { LocaleSwitch },
  setup: () => ({ formatHectares, formatPointCount, formatDate }),
  template: `<div>
    <LocaleSwitch />
    <p class="title">{{ $t('calc.area') }}</p>
    <p class="area">{{ formatHectares(2.48124) }}</p>
    <p class="count">{{ formatPointCount(5) }}</p>
    <p class="date">{{ formatDate(new Date(2026, 9, 27)) }}</p>
  </div>`,
})

function mountProbe() {
  const settings = { chooseLocale: vi.fn() }

  return {
    settings,
    wrapper: mount(Probe, {
      global: { provide: { [settingsServiceKey as symbol]: settings as unknown as SettingsService } },
    }),
  }
}

describe('LocaleSwitch', () => {
  it('switches texts, numbers, dates and the page language and remembers the choice', async () => {
    const { wrapper, settings } = mountProbe()
    expect(wrapper.get('.title').text()).toBe('Площа ділянки')
    expect(wrapper.get('.area').text()).toBe('2,48 га')
    expect(wrapper.get('.count').text()).toBe('5 точок')
    expect(wrapper.get('.date').text()).toBe('27.10.2026')
    expect(document.documentElement.lang).toBe('uk')
    expect(wrapper.get('[aria-pressed="true"]').text()).toBe('УК')

    await wrapper.get('button[lang="en"]').trigger('click')

    expect(settings.chooseLocale).toHaveBeenCalledWith('en')
    expect(document.documentElement.lang).toBe('en')
    expect(wrapper.get('[aria-pressed="true"]').text()).toBe('EN')
    expect(wrapper.get('.title').text()).toBe('Plot area')
    expect(wrapper.get('.area').text()).toBe('2.48 ha')
    expect(wrapper.get('.count').text()).toBe('5 points')
    expect(wrapper.get('.date').text()).toBe('27/10/2026')
  })

  it.each([
    [1, 0],
    [3, 1],
    [5, 2],
    [11, 2],
    [21, 0],
    [22, 1],
    [114, 2],
  ])('uses Ukrainian plural form for %i', (count, form) => {
    expect(ukrainianPlural(count, 3)).toBe(form)
  })
})
