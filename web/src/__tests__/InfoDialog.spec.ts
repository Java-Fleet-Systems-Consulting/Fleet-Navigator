import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import InfoDialog from '../components/InfoDialog.vue'

// Mock API
vi.mock('../services/api', () => ({
  default: {
    getSystemVersion: vi.fn(),
    checkForUpdate: vi.fn()
  }
}))

import api from '../services/api'

// Erstelle i18n Instanz für Tests
const i18n = createI18n({
  legacy: false,
  locale: 'de',
  messages: {
    de: {
      infoDialog: {
        title: 'Über Fleet Navigator',
        subtitle: 'KI-gestütztes Experten-System',
        version: 'Version',
        buildTime: 'Build-Datum',
        checkUpdate: 'Nach Updates suchen',
        checking: 'Suche nach Updates...',
        checkFailed: 'Update-Prüfung fehlgeschlagen',
        newVersion: 'Neue Version',
        viewRelease: 'Release anzeigen',
        madeWith: 'Made with Go & Vue.js'
      }
    }
  }
})

describe('InfoDialog.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('rendert nicht wenn show=false', () => {
    const wrapper = mount(InfoDialog, {
      props: { show: false },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    expect(wrapper.find('.fixed').exists()).toBe(false)
  })

  it('rendert Dialog wenn show=true', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: '2025-01-01'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.find('.fixed').exists()).toBe(true)
    expect(wrapper.text()).toContain('Fleet Navigator')
    expect(wrapper.text()).toContain('Über Fleet Navigator')
  })

  it('zeigt Version und Build-Datum an', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '2.0.0',
      buildTime: '2025-12-25T10:30:00Z'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('2.0.0')
    expect(wrapper.text()).toContain('Version')
    expect(wrapper.text()).toContain('Build-Datum')
  })

  it('zeigt development build wenn buildTime="development"', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '0.8.0',
      buildTime: 'development'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('Development Build')
  })

  it('ruft checkForUpdate beim Button-Klick auf', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: 'development'
    })
    ;(api.checkForUpdate as any).mockResolvedValueOnce({
      updateAvailable: false,
      currentVersion: '1.0.0',
      message: 'Sie verwenden die aktuelle Version'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    // Finde den Update-Button (der mit dem Text "Nach Updates suchen")
    const buttons = wrapper.findAll('button')
    const updateButton = buttons.find(b => b.text().includes('Updates suchen'))

    expect(updateButton).toBeDefined()
    if (updateButton) {
      await updateButton.trigger('click')
      await flushPromises()

      expect(api.checkForUpdate).toHaveBeenCalled()
    }
  })

  it('zeigt Update-Status nach Prüfung', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: 'development'
    })

    const updateResponse = {
      updateAvailable: true,
      currentVersion: '1.0.0',
      latestVersion: '2.0.0',
      message: 'Neue Version 2.0.0 verfügbar!',
      releaseURL: 'https://github.com/example/releases/v2.0.0'
    }
    ;(api.checkForUpdate as any).mockResolvedValueOnce(updateResponse)

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    // Klicke Update-Button
    const buttons = wrapper.findAll('button')
    const updateButton = buttons.find(b => b.text().includes('Updates suchen'))

    expect(updateButton).toBeDefined()
    if (updateButton) {
      await updateButton.trigger('click')
      await flushPromises()

      // Der Text sollte jetzt die neue Version enthalten
      const text = wrapper.text()
      expect(text).toContain('Neue Version')
    }
  })

  it('emitiert close Event beim Klick auf X-Button', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: 'development'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    // Finde den Close-Button (erster Button im Header)
    const closeButton = wrapper.find('button')
    await closeButton.trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('emitiert close Event beim Klick auf Backdrop', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: 'development'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    // Finde den Backdrop und klicke
    const backdrop = wrapper.find('.bg-black\\/60')
    if (backdrop.exists()) {
      await backdrop.trigger('click')
      expect(wrapper.emitted('close')).toBeTruthy()
    }
  })

  it('zeigt Footer mit Copyright', async () => {
    ;(api.getSystemVersion as any).mockResolvedValueOnce({
      version: '1.0.0',
      buildTime: 'development'
    })

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('JavaFleet Systems Consulting')
    expect(wrapper.text()).toContain('Made with Go & Vue.js')
  })

  it('behandelt API-Fehler graceful', async () => {
    ;(api.getSystemVersion as any).mockRejectedValueOnce(new Error('Network error'))

    const wrapper = mount(InfoDialog, {
      props: { show: true },
      global: {
        plugins: [i18n],
        stubs: {
          Teleport: true
        }
      }
    })

    await flushPromises()

    // Dialog sollte trotzdem rendern
    expect(wrapper.find('.fixed').exists()).toBe(true)
    // Version sollte leer oder '...' sein
    expect(wrapper.text()).toContain('...')
  })
})
