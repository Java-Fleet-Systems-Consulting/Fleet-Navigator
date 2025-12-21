import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { supportedLocales } from '../i18n'

/**
 * Global Locale Management für Fleet Navigator
 * Wrapper um vue-i18n für einfache Nutzung in Komponenten
 * Unterstützt DE, EN, TR, FR, ES
 */

export function useLocale() {
  const { t, locale } = useI18n()

  const isGerman = computed(() => locale.value === 'de')
  const isEnglish = computed(() => locale.value === 'en')
  const isTurkish = computed(() => locale.value === 'tr')
  const isFrench = computed(() => locale.value === 'fr')
  const isSpanish = computed(() => locale.value === 'es')

  const setLocale = (newLocale) => {
    if (supportedLocales.includes(newLocale)) {
      locale.value = newLocale
      localStorage.setItem('fleet-navigator-locale', newLocale)
      document.documentElement.lang = newLocale
    }
  }

  return {
    locale: computed(() => locale.value),
    isGerman,
    isEnglish,
    isTurkish,
    isFrench,
    isSpanish,
    t,
    setLocale,
    availableLocales: computed(() => supportedLocales)
  }
}
