/**
 * Chat Store - Simplified to Chat CRUD operations
 *
 * This store has been refactored from 1375 lines to ~450 lines.
 * Expert/Model management → expertStore.js
 * Streaming logic → streamingStore.js
 */
import { defineStore, storeToRefs } from 'pinia'
import { ref, computed, watch } from 'vue'
import api from '../services/api'
import { useExpertStore } from './expertStore'
import { useStreamingStore } from './streamingStore'

const CHAT_CACHE_KEY = 'fleet-navigator-chat-cache'

export const useChatStore = defineStore('chat', () => {
  // Lazy store accessors to avoid circular dependency issues
  // These are called only when needed, not during store initialization
  let _expertStore = null
  let _streamingStore = null

  const getExpertStore = () => {
    if (!_expertStore) _expertStore = useExpertStore()
    return _expertStore
  }

  const getStreamingStore = () => {
    if (!_streamingStore) _streamingStore = useStreamingStore()
    return _streamingStore
  }

  // Cache helper functions
  const saveChatToCache = (chatId, msgs) => {
    try {
      const cache = JSON.parse(localStorage.getItem(CHAT_CACHE_KEY) || '{}')
      cache[chatId] = msgs
      localStorage.setItem(CHAT_CACHE_KEY, JSON.stringify(cache))
    } catch (e) {
      console.error('Failed to save chat to cache:', e)
    }
  }

  const getChatFromCache = (chatId) => {
    try {
      const cache = JSON.parse(localStorage.getItem(CHAT_CACHE_KEY) || '{}')
      return cache[chatId] || null
    } catch (e) {
      console.error('Failed to get chat from cache:', e)
      return null
    }
  }

  // State
  const chats = ref([])
  const currentChat = ref(null)
  const messages = ref([])
  const error = ref(null)
  const globalStats = ref({
    totalTokens: 0,
    totalMessages: 0,
    chatCount: 0
  })
  const systemStatus = ref({
    cpuUsage: 0,
    totalMemory: 0,
    freeMemory: 0,
    usedMemory: 0
  })

  // Computed
  const currentChatTokens = computed(() => {
    if (!messages.value || !messages.value.length) return 0
    return messages.value.reduce((sum, msg) => sum + (msg.tokens || 0), 0)
  })

  const memoryUsagePercent = computed(() => {
    if (!systemStatus.value.totalMemory) return 0
    return Math.round((systemStatus.value.usedMemory / systemStatus.value.totalMemory) * 100)
  })

  // ============================================
  // Chat CRUD Actions
  // ============================================

  async function loadChats() {
    try {
      chats.value = await api.getAllChats() || []
    } catch (err) {
      error.value = 'Failed to load chats'
      console.error(err)
      chats.value = []
    }
  }

  async function createNewChat(title = 'New Chat') {
    const expertStore = getExpertStore()
    const streamingStore = getStreamingStore()
    try {
      streamingStore.isLoading = true
      const chat = await api.createNewChat({
        title,
        model: expertStore.selectedModel,
        expertId: expertStore.selectedExpertId
      })
      chats.value.unshift(chat)
      currentChat.value = chat
      messages.value = []
      saveChatToCache(chat.id, [])
      console.log('[Cache-Sync] Initialized cache for new chat with expertId:', expertStore.selectedExpertId)
      return chat
    } catch (err) {
      error.value = 'Failed to create chat'
      console.error(err)
    } finally {
      streamingStore.isLoading = false
    }
  }

  async function loadChatHistory(chatId) {
    const expertStore = getExpertStore()
    const streamingStore = getStreamingStore()
    try {
      streamingStore.isLoading = true
      const chat = await api.getChatHistory(chatId)
      currentChat.value = chat
      messages.value = chat.messages || []

      // Restore expert if chat has one
      await expertStore.restoreExpertFromChat(chat.expertId)

      saveChatToCache(chatId, messages.value)
      console.log('[Cache-Sync] Chat history loaded from H2 DB')
    } catch (err) {
      error.value = 'Failed to load chat history'
      console.error(err)
    } finally {
      streamingStore.isLoading = false
    }
  }

  async function renameChat(chatId, newTitle) {
    try {
      await api.renameChat(chatId, newTitle)
      const index = chats.value.findIndex(c => c.id === chatId)
      if (index !== -1) {
        chats.value[index] = { ...chats.value[index], title: newTitle }
      }
      if (currentChat.value?.id === chatId) {
        currentChat.value.title = newTitle
      }
    } catch (err) {
      error.value = 'Failed to rename chat'
      console.error(err)
      throw err
    }
  }

  async function deleteChat(chatId) {
    try {
      console.log('[Chat-Delete] Deleting chat:', chatId)
      await api.deleteChat(chatId)

      const chatIdNum = Number(chatId)
      chats.value = chats.value.filter(c => Number(c.id) !== chatIdNum)

      if (currentChat.value && Number(currentChat.value.id) === chatIdNum) {
        currentChat.value = null
        messages.value = []
      }

      // Remove from cache
      try {
        const cache = JSON.parse(localStorage.getItem(CHAT_CACHE_KEY) || '{}')
        delete cache[chatId]
        delete cache[String(chatIdNum)]
        localStorage.setItem(CHAT_CACHE_KEY, JSON.stringify(cache))
      } catch (e) {
        console.error('Failed to remove chat from cache:', e)
      }

      console.log('[Chat-Delete] Success!')
    } catch (err) {
      error.value = 'Failed to delete chat'
      console.error(err)
      throw err
    }
  }

  function startNewChat() {
    currentChat.value = null
    messages.value = []
  }

  // ============================================
  // Stats & Status
  // ============================================

  async function loadGlobalStats() {
    try {
      globalStats.value = await api.getGlobalStats()
    } catch (err) {
      console.error('Failed to load global stats', err)
    }
  }

  async function loadSystemStatus() {
    try {
      systemStatus.value = await api.getSystemStatus()
    } catch (err) {
      console.error('Failed to load system status', err)
    }
  }

  function clearError() {
    error.value = null
  }

  // ============================================
  // Message Operations (delegates to streamingStore)
  // ============================================

  async function sendMessage(messageData) {
    const expertStore = getExpertStore()
    const streamingStore = getStreamingStore()
    const messageText = typeof messageData === 'string' ? messageData : messageData.text
    const files = typeof messageData === 'object' ? messageData.files || [] : []
    const webSearchEnabled = typeof messageData === 'object' ? messageData.webSearchEnabled || false : false

    if (!messageText.trim()) return

    error.value = null

    return await streamingStore.sendMessage({
      messageText,
      files,
      webSearchEnabled,
      currentChat: currentChat.value,
      addMessage: (msg) => messages.value.push(msg),
      updateMessage: (oldMsg, newMsg, isNew = false) => {
        if (isNew) {
          messages.value.push(newMsg)
        } else if (oldMsg) {
          const index = messages.value.findIndex(m => m === oldMsg)
          if (index !== -1) {
            messages.value[index] = newMsg
            if (currentChat.value?.id) {
              saveChatToCache(currentChat.value.id, messages.value)
            }
          }
        }
      },
      onChatCreated: async (chatId) => {
        if (!currentChat.value) {
          currentChat.value = { id: chatId }
          await loadChats()
        }
      },
      onError: (errorMsg) => {
        error.value = errorMsg
      },
      onComplete: () => {
        loadGlobalStats()
        // Sync with database after delay
        if (currentChat.value?.id) {
          setTimeout(async () => {
            try {
              const chat = await api.getChatHistory(currentChat.value.id)
              messages.value = chat.messages || []
              saveChatToCache(currentChat.value.id, messages.value)
            } catch (e) {
              console.error('[Cache-Sync] Failed to sync:', e)
            }
          }, 500)
        }
      },
      onDelegation: (expertId, expertName) => {
        if (expertId) {
          expertStore.selectedExpertId = expertId
          console.log('[Delegation] Expert changed to:', expertName)
        }
      }
    })
  }

  // ============================================
  // Model/Expert Operations (delegates to expertStore)
  // ============================================

  async function loadModels() {
    try {
      await getExpertStore().loadModels()
    } catch (err) {
      error.value = 'Failed to load models'
      console.error(err)
    }
  }

  async function setSelectedModel(model) {
    await getExpertStore().setSelectedModel(model, currentChat.value?.id)
  }

  async function selectExpert(expert) {
    const result = await getExpertStore().selectExpert(expert, currentChat.value?.id)
    if (result?.maxContextTokens) {
      getStreamingStore().contextUsage.maxContextTokens = result.maxContextTokens
    }
  }

  async function updateCurrentChatModel(modelName) {
    if (!currentChat.value?.id) {
      console.warn('No current chat to update model')
      return
    }

    try {
      await api.updateChatModel(currentChat.value.id, modelName)
      currentChat.value.model = modelName
      const chatIndex = chats.value.findIndex(c => c.id === currentChat.value.id)
      if (chatIndex !== -1) {
        chats.value[chatIndex].model = modelName
      }
      console.log('✅ Chat model updated successfully')
    } catch (err) {
      console.error('Failed to update chat model:', err)
      throw err
    }
  }

  // ============================================
  // Backward Compatibility: Re-export state from other stores
  // Using computed() for reactive proxying
  // ============================================

  // Expert State - computed proxies (lazy loaded)
  const models = computed(() => getExpertStore().models)
  const experts = computed(() => getExpertStore().experts)
  const customModels = computed(() => getExpertStore().customModels)
  const selectedModel = computed({
    get: () => getExpertStore().selectedModel,
    set: (v) => { getExpertStore().selectedModel = v }
  })
  const selectedExpertId = computed({
    get: () => getExpertStore().selectedExpertId,
    set: (v) => { getExpertStore().selectedExpertId = v }
  })
  const systemPrompt = computed({
    get: () => getExpertStore().systemPrompt,
    set: (v) => { getExpertStore().systemPrompt = v }
  })
  const systemPromptTitle = computed({
    get: () => getExpertStore().systemPromptTitle,
    set: (v) => { getExpertStore().systemPromptTitle = v }
  })
  const isSwitchingExpert = computed(() => getExpertStore().isSwitchingExpert)
  const switchingExpertMessage = computed(() => getExpertStore().switchingExpertMessage)

  // Streaming State - computed proxies (lazy loaded)
  const isLoading = computed({
    get: () => getStreamingStore().isLoading,
    set: (v) => { getStreamingStore().isLoading = v }
  })
  const isWebSearching = computed(() => getStreamingStore().isWebSearching)
  const isSwappingModel = computed(() => getStreamingStore().isSwappingModel)
  const modelSwapMessage = computed(() => getStreamingStore().modelSwapMessage)
  const modelSwapProgress = computed(() => getStreamingStore().modelSwapProgress)
  const streamingEnabled = computed({
    get: () => getStreamingStore().streamingEnabled,
    set: (v) => { getStreamingStore().streamingEnabled = v }
  })
  const currentRequestId = computed(() => getStreamingStore().currentRequestId)
  const contextUsage = computed(() => getStreamingStore().contextUsage)

  // Vision Chaining State
  const isAnalyzingVision = computed(() => getStreamingStore().isAnalyzingVision)
  const visionChainMessage = computed(() => getStreamingStore().visionChainMessage)
  const visionChainProgress = computed(() => getStreamingStore().visionChainProgress)

  // This ensures existing components continue to work without changes
  return {
    // Chat State (this store)
    chats,
    currentChat,
    messages,
    error,
    globalStats,
    systemStatus,

    // Expert State (computed proxies)
    models,
    experts,
    customModels,
    selectedModel,
    selectedExpertId,
    systemPrompt,
    systemPromptTitle,
    isSwitchingExpert,
    switchingExpertMessage,

    // Streaming State (computed proxies)
    isLoading,
    isWebSearching,
    isSwappingModel,
    modelSwapMessage,
    modelSwapProgress,
    streamingEnabled,
    currentRequestId,
    contextUsage,

    // Vision Chaining State
    isAnalyzingVision,
    visionChainMessage,
    visionChainProgress,

    // Computed
    currentChatTokens,
    memoryUsagePercent,

    // Helper Functions (from expertStore) - wrapped for lazy access
    isCustomModel: (...args) => getExpertStore().isCustomModel(...args),
    getExpertById: (...args) => getExpertStore().getExpertById(...args),

    // Chat CRUD Actions
    loadChats,
    createNewChat,
    loadChatHistory,
    renameChat,
    deleteChat,
    startNewChat,
    loadGlobalStats,
    loadSystemStatus,
    clearError,

    // Message Actions (delegates)
    sendMessage,

    // Model/Expert Actions (delegates)
    loadModels,
    setSelectedModel,
    selectExpert,
    updateCurrentChatModel,
    setSystemPrompt: (...args) => getExpertStore().setSystemPrompt(...args),

    // Streaming Actions (delegates)
    toggleStreaming: () => getStreamingStore().toggleStreaming(),
    abortCurrentRequest: () => getStreamingStore().abortCurrentRequest()
  }
})
