import { computed, type ComputedRef } from 'vue'
import { useI18n } from 'vue-i18n'

import type { Locale } from '../../domain/locale'
import { applyLocale } from '../i18n'

import { useSettingsService } from './services'

export function useLocale(): { locale: ComputedRef<Locale>; choose: (locale: Locale) => void } {
  const settings = useSettingsService()
  const { locale } = useI18n()

  return {
    locale: computed(() => locale.value as Locale),
    choose: (next: Locale) => {
      settings.chooseLocale(next)
      applyLocale(next)
    },
  }
}
