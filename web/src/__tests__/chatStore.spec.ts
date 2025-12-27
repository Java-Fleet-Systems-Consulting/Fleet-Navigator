import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock Vite globals - must be hoisted before any imports
globalThis.__APP_VERSION__ = '1.0.0-test'

// Mock settingsStore to avoid __APP_VERSION__ issue
vi.mock('../stores/settingsStore', () => ({
  useSettingsStore: () => ({
    settings: {},
    getSetting: vi.fn(() => null),
    isVisionModel: vi.fn(() => false)
  })
}))

// Mock API before importing stores
vi.mock('../services/api', () => ({
  default: {
    getAllChats: vi.fn(() => Promise.resolve([])),
    createNewChat: vi.fn(() => Promise.resolve({ id: 1, title: 'Test Chat' })),
    getChatHistory: vi.fn(() => Promise.resolve({ id: 1, messages: [] })),
    getGlobalStats: vi.fn(() => Promise.resolve({ totalTokens: 0, totalMessages: 0, chatCount: 0 })),
    getSystemStatus: vi.fn(() => Promise.resolve({ cpuUsage: 0, totalMemory: 8000, freeMemory: 4000 }))
  }
}))

vi.mock('../utils/secureFetch', () => ({
  secureFetch: vi.fn(() => Promise.resolve({ ok: true, json: () => Promise.resolve({}) }))
}))

// Import stores after mocks are set up
import { useChatStore } from '../stores/chatStore'
import { useExpertStore } from '../stores/expertStore'
import { useStreamingStore } from '../stores/streamingStore'

describe('chatStore - Lazy Store Loading', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('lädt ohne Fehler (keine zirkuläre Abhängigkeit)', () => {
    // This would fail before the fix with "Cannot read properties of null (reading 'effect')"
    expect(() => useChatStore()).not.toThrow()
  })

  it('expertStore wird erst bei Zugriff geladen', () => {
    const chatStore = useChatStore()

    // Access expertStore property - should work without error
    expect(() => chatStore.models).not.toThrow()
    expect(() => chatStore.experts).not.toThrow()
    expect(() => chatStore.selectedModel).not.toThrow()
  })

  it('streamingStore wird erst bei Zugriff geladen', () => {
    const chatStore = useChatStore()

    // Access streamingStore properties - should work without error
    expect(() => chatStore.isLoading).not.toThrow()
    expect(() => chatStore.isWebSearching).not.toThrow()
    expect(() => chatStore.streamingEnabled).not.toThrow()
  })

  it('isCustomModel delegiert korrekt an expertStore', () => {
    const chatStore = useChatStore()
    const expertStore = useExpertStore()

    // isCustomModel should delegate to expertStore
    const result = chatStore.isCustomModel('test-model')
    expect(result).toBe(false) // Default: no custom models
  })

  it('getExpertById delegiert korrekt an expertStore', () => {
    const chatStore = useChatStore()

    // getExpertById should work without error
    const result = chatStore.getExpertById(1)
    expect(result).toBeNull() // No experts loaded yet
  })

  it('toggleStreaming delegiert korrekt an streamingStore', () => {
    const chatStore = useChatStore()
    const streamingStore = useStreamingStore()

    const before = streamingStore.streamingEnabled
    chatStore.toggleStreaming()
    expect(streamingStore.streamingEnabled).toBe(!before)
  })
})

describe('chatStore - Basic Functionality', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('hat initiale State-Werte', () => {
    const chatStore = useChatStore()

    expect(chatStore.chats).toEqual([])
    expect(chatStore.currentChat).toBeNull()
    expect(chatStore.messages).toEqual([])
    expect(chatStore.error).toBeNull()
  })

  it('currentChatTokens berechnet korrekt', () => {
    const chatStore = useChatStore()

    // Initial: no messages
    expect(chatStore.currentChatTokens).toBe(0)
  })

  it('memoryUsagePercent berechnet korrekt', () => {
    const chatStore = useChatStore()

    // Initial: systemStatus has default values
    expect(chatStore.memoryUsagePercent).toBe(0)
  })

  it('startNewChat setzt currentChat und messages zurück', () => {
    const chatStore = useChatStore()

    // Simulate having a current chat
    chatStore.currentChat = { id: 1, title: 'Test' }
    chatStore.messages = [{ role: 'USER', content: 'Hello' }]

    chatStore.startNewChat()

    expect(chatStore.currentChat).toBeNull()
    expect(chatStore.messages).toEqual([])
  })

  it('clearError setzt error zurück', () => {
    const chatStore = useChatStore()

    chatStore.error = 'Some error'
    chatStore.clearError()

    expect(chatStore.error).toBeNull()
  })
})
