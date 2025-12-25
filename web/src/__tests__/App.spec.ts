import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import App from '../App.vue'

// Mock fetch
global.fetch = vi.fn()

// Mock Komponenten
vi.mock('../components/SetupWizard.vue', () => ({
  default: {
    template: '<div class="setup-wizard-mock">Setup Wizard</div>',
    emits: ['complete']
  }
}))

vi.mock('../components/GlobalConfirmDialog.vue', () => ({
  default: { template: '<div></div>' }
}))

vi.mock('../components/UpdateNotification.vue', () => ({
  default: { template: '<div></div>' }
}))

describe('App.vue - Setup Wizard Logic', () => {
  let router: any

  beforeEach(() => {
    vi.clearAllMocks()

    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/help', component: { template: '<div>Help</div>' } }
      ]
    })
  })

  it('zeigt Setup-Wizard wenn isFirstRun=true und NICHT auf /help', async () => {
    // Mock: isFirstRun = true
    ;(global.fetch as any).mockResolvedValueOnce({
      json: () => Promise.resolve({ isFirstRun: true })
    })

    router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: { plugins: [router] }
    })

    await flushPromises()

    expect(wrapper.find('.setup-wizard-mock').exists()).toBe(true)
  })

  it('versteckt Setup-Wizard auf /help auch wenn isFirstRun=true', async () => {
    // Mock: isFirstRun = true
    ;(global.fetch as any).mockResolvedValueOnce({
      json: () => Promise.resolve({ isFirstRun: true })
    })

    router.push('/help')
    await router.isReady()

    const wrapper = mount(App, {
      global: { plugins: [router] }
    })

    await flushPromises()

    // Setup-Wizard sollte NICHT angezeigt werden auf /help
    expect(wrapper.find('.setup-wizard-mock').exists()).toBe(false)
  })

  it('zeigt keinen Setup-Wizard wenn isFirstRun=false', async () => {
    // Mock: isFirstRun = false
    ;(global.fetch as any).mockResolvedValueOnce({
      json: () => Promise.resolve({ isFirstRun: false })
    })

    router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: { plugins: [router] }
    })

    await flushPromises()

    expect(wrapper.find('.setup-wizard-mock').exists()).toBe(false)
  })

  it('behandelt API-Fehler graceful (kein Setup-Wizard)', async () => {
    // Mock: API wirft Fehler
    ;(global.fetch as any).mockRejectedValueOnce(new Error('Network error'))

    router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: { plugins: [router] }
    })

    await flushPromises()

    // Bei Fehler: Kein Setup-Wizard
    expect(wrapper.find('.setup-wizard-mock').exists()).toBe(false)
  })
})
