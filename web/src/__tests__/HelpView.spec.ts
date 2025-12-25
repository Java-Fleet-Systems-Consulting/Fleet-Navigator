import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createMemoryHistory } from 'vue-router'
import HelpView from '../views/HelpView.vue'

// Mock Router
const router = createRouter({
  history: createMemoryHistory(),
  routes: [
    { path: '/help', component: HelpView }
  ]
})

describe('HelpView.vue', () => {
  beforeEach(async () => {
    router.push('/help')
    await router.isReady()
  })

  it('rendert die Hilfe-Seite', () => {
    const wrapper = mount(HelpView, {
      global: {
        plugins: [router]
      }
    })

    expect(wrapper.find('.help-page').exists()).toBe(true)
    expect(wrapper.find('.help-header').exists()).toBe(true)
    expect(wrapper.find('.topic-nav').exists()).toBe(true)
  })

  it('zeigt "Lokal vs. Cloud" als erstes Topic', () => {
    const wrapper = mount(HelpView, {
      global: {
        plugins: [router]
      }
    })

    const firstTopic = wrapper.find('.topic-btn.active')
    expect(firstTopic.text()).toContain('Lokal vs. Cloud')
  })

  it('hat 9 Topics in der Navigation', () => {
    const wrapper = mount(HelpView, {
      global: {
        plugins: [router]
      }
    })

    const topics = wrapper.findAll('.topic-btn')
    expect(topics).toHaveLength(9)
  })

  it('wechselt Topic bei Klick', async () => {
    const wrapper = mount(HelpView, {
      global: {
        plugins: [router]
      }
    })

    // Klick auf "Vision-Modelle"
    const visionBtn = wrapper.findAll('.topic-btn').find(btn =>
      btn.text().includes('Vision-Modelle')
    )
    await visionBtn?.trigger('click')

    // Prüfen ob Vision-Content angezeigt wird
    expect(wrapper.find('.topic-article h1').text()).toContain('Vision')
  })

  it('Topics haben korrekte Reihenfolge', () => {
    const wrapper = mount(HelpView, {
      global: {
        plugins: [router]
      }
    })

    const topicNames = wrapper.findAll('.topic-name').map(el => el.text())

    expect(topicNames[0]).toBe('Lokal vs. Cloud')
    expect(topicNames[1]).toBe('Instruct-Modelle')
    expect(topicNames[2]).toBe('Was bedeutet 8B?')
    expect(topicNames[3]).toBe('Fokussierte Experten')
    expect(topicNames[4]).toBe('Vision-Modelle')
    expect(topicNames[5]).toBe('Vision Chaining')
    expect(topicNames[6]).toBe('RAG & Websuche')
  })
})
