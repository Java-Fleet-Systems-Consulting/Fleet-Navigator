import { createI18n } from 'vue-i18n'
import de from './locales/de.json'
import en from './locales/en.json'
import tr from './locales/tr.json'
import fr from './locales/fr.json'
import es from './locales/es.json'

// Unterstützte Sprachen
export const supportedLocales = ['de', 'en', 'tr', 'fr', 'es']

// Sprache aus localStorage laden oder Browser-Sprache verwenden
function getDefaultLocale() {
  const saved = localStorage.getItem('fleet-navigator-locale')
  if (saved && supportedLocales.includes(saved)) {
    return saved
  }

  // Browser-Sprache prüfen
  const browserLang = navigator.language?.substring(0, 2)
  if (supportedLocales.includes(browserLang)) {
    return browserLang
  }

  return 'de' // Default: Deutsch
}

const i18n = createI18n({
  legacy: false, // Composition API
  locale: getDefaultLocale(),
  fallbackLocale: 'en',
  messages: {
    de,
    en,
    tr,
    fr,
    es
  }
})

// Helper: Sprache wechseln und speichern
export function setLocale(locale) {
  if (supportedLocales.includes(locale)) {
    i18n.global.locale.value = locale
    localStorage.setItem('fleet-navigator-locale', locale)
    document.documentElement.lang = locale
  }
}

// Helper: Aktuelle Sprache abrufen
export function getLocale() {
  return i18n.global.locale.value
}

export default i18n
