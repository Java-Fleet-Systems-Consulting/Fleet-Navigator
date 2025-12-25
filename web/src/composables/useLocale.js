import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { supportedLocales } from '../i18n'

/**
 * Global Locale Management für Fleet Navigator
 * Wrapper um vue-i18n für einfache Nutzung in Komponenten
 * Unterstützt DE, EN, TR, FR, ES
 *
 * WICHTIG: Bei Sprachwechsel wird das Backend informiert!
 * - Experten-Prompts werden in der neuen Sprache geladen
 * - Verfügbare TTS-Stimmen werden zurückgegeben
 */

// Reaktiver State für Sprachwechsel-Info
const lastLanguageChangeResult = ref(null)

export function useLocale() {
  const { t, locale } = useI18n()

  const isGerman = computed(() => locale.value === 'de')
  const isEnglish = computed(() => locale.value === 'en')
  const isTurkish = computed(() => locale.value === 'tr')
  const isFrench = computed(() => locale.value === 'fr')
  const isSpanish = computed(() => locale.value === 'es')

  /**
   * Setzt die Sprache und informiert das Backend
   * @param {string} newLocale - 'de', 'en', 'tr', 'fr', 'es'
   * @returns {Promise<object>} Backend-Antwort mit verfügbaren Stimmen
   */
  const setLocale = async (newLocale) => {
    if (!supportedLocales.includes(newLocale)) {
      console.warn(`[useLocale] Unsupported locale: ${newLocale}`)
      return null
    }

    // UI sofort aktualisieren
    locale.value = newLocale
    localStorage.setItem('fleet-navigator-locale', newLocale)
    document.documentElement.lang = newLocale

    // Backend informieren (für Experten-Prompts & Stimmen)
    try {
      const response = await fetch('/api/settings/language', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ locale: newLocale })
      })

      if (response.ok) {
        const result = await response.json()
        lastLanguageChangeResult.value = result
        console.log(`[useLocale] Sprache gewechselt zu ${newLocale}:`, result)

        // Event dispatchen für Komponenten die darauf reagieren wollen
        window.dispatchEvent(new CustomEvent('locale-changed', {
          detail: { locale: newLocale, ...result }
        }))

        return result
      } else {
        console.warn(`[useLocale] Backend konnte Sprache nicht speichern: ${response.status}`)
      }
    } catch (err) {
      console.warn('[useLocale] Backend nicht erreichbar:', err.message)
    }

    return null
  }

  /**
   * Lädt die aktuelle Sprache vom Backend (beim App-Start)
   */
  const loadLocaleFromBackend = async () => {
    try {
      const response = await fetch('/api/settings/language')
      if (response.ok) {
        const result = await response.json()
        if (result.locale && supportedLocales.includes(result.locale)) {
          locale.value = result.locale
          localStorage.setItem('fleet-navigator-locale', result.locale)
          document.documentElement.lang = result.locale
          lastLanguageChangeResult.value = result
          return result
        }
      }
    } catch (err) {
      console.warn('[useLocale] Konnte Sprache nicht vom Backend laden:', err.message)
    }
    return null
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
    loadLocaleFromBackend,
    lastLanguageChangeResult: computed(() => lastLanguageChangeResult.value),
    availableLocales: computed(() => supportedLocales)
  }
}
