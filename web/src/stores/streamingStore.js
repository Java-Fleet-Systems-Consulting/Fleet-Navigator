import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import api from '../services/api'
import { secureFetch } from '../utils/secureFetch'
import { useSettingsStore } from './settingsStore'
import { useExpertStore } from './expertStore'

const SAMPLING_PARAMS_KEY = 'defaultSamplingParams'

export const useStreamingStore = defineStore('streaming', () => {
  // Load sampling parameters from localStorage
  const loadSamplingParams = () => {
    try {
      const stored = localStorage.getItem(SAMPLING_PARAMS_KEY)
      if (stored) {
        return JSON.parse(stored)
      }
    } catch (e) {
      console.error('Failed to load sampling params', e)
    }
    return null
  }

  // State
  const isLoading = ref(false)
  const isWebSearching = ref(false)
  const isSwappingModel = ref(false)
  const modelSwapMessage = ref('')
  const modelSwapProgress = ref(0)
  const currentRequestId = ref(null)
  const streamingEnabled = ref(true)

  // Context usage tracking (for progressbar)
  const contextUsage = ref({
    totalChatTokens: 0,
    maxContextTokens: null
  })

  // Toggle streaming mode
  function toggleStreaming() {
    streamingEnabled.value = !streamingEnabled.value
  }

  // Abort current request
  async function abortCurrentRequest() {
    if (!currentRequestId.value) {
      console.warn('No active request to abort')
      return false
    }

    try {
      console.log('Aborting request:', currentRequestId.value)
      await api.abortRequest(currentRequestId.value)
      isLoading.value = false
      currentRequestId.value = null
      return true
    } catch (err) {
      console.error('Failed to abort request', err)
      return false
    }
  }

  /**
   * Send a message to the LLM
   * @param {Object} params
   * @param {string} params.messageText - The message text
   * @param {Array} params.files - Uploaded files
   * @param {boolean} params.webSearchEnabled - Enable web search
   * @param {Object} params.currentChat - Current chat object
   * @param {Function} params.addMessage - Function to add message to messages array
   * @param {Function} params.updateMessage - Function to update a message
   * @param {Function} params.onChatCreated - Callback when new chat is created
   * @param {Function} params.onError - Error callback
   * @param {Function} params.onComplete - Completion callback
   */
  async function sendMessage(params) {
    const {
      messageText,
      files = [],
      webSearchEnabled = false,
      currentChat,
      addMessage,
      updateMessage,
      onChatCreated,
      onError,
      onComplete,
      onDelegation
    } = params

    if (!messageText.trim()) return

    const expertStore = useExpertStore()
    const settingsStore = useSettingsStore()

    try {
      isLoading.value = true
      isWebSearching.value = webSearchEnabled

      // Process uploaded files
      const images = []
      let documentContext = ''
      const fileMetadata = []

      for (const file of files) {
        fileMetadata.push({
          name: file.name,
          type: file.type,
          size: file.size
        })

        if (file.type === 'image' && file.base64Content) {
          images.push(file.base64Content)
        } else if (file.textContent) {
          documentContext += `\n\n=== ${file.name} ===\n${file.textContent}`
        }
      }

      // Vision chaining check
      let visionChainEnabled = false
      let visionModel = null

      if (images.length > 0) {
        const isCurrentModelVision = settingsStore.isVisionModel(expertStore.selectedModel)
        if (!isCurrentModelVision) {
          visionChainEnabled = true
          visionModel = settingsStore.getSetting('preferredVisionModel')
        }
      }

      // Add user message optimistically
      const userMessage = {
        role: 'USER',
        content: messageText,
        createdAt: new Date().toISOString(),
        attachments: fileMetadata.length > 0 ? JSON.stringify(fileMetadata) : null
      }
      addMessage(userMessage)

      if (streamingEnabled.value) {
        await sendMessageStreaming({
          messageText,
          images,
          documentContext,
          visionChainEnabled,
          visionModel,
          fileMetadata,
          webSearchEnabled,
          currentChat,
          addMessage,
          updateMessage,
          onChatCreated,
          onError,
          onComplete,
          onDelegation
        })
      } else {
        // Non-streaming mode
        await sendMessageNonStreaming({
          messageText,
          images,
          documentContext,
          fileMetadata,
          webSearchEnabled,
          currentChat,
          addMessage,
          onChatCreated,
          onComplete
        })
      }

      return true
    } catch (err) {
      if (err.message && err.message.includes('cancelled')) {
        console.log('Request was cancelled by user')
      } else {
        console.error(err)
        if (onError) onError('Failed to send message')
      }
    } finally {
      isLoading.value = false
      isWebSearching.value = false
      currentRequestId.value = null
    }
  }

  // Non-streaming message sending
  async function sendMessageNonStreaming(params) {
    const {
      messageText,
      images,
      documentContext,
      fileMetadata,
      webSearchEnabled,
      currentChat,
      addMessage,
      onChatCreated,
      onComplete
    } = params

    const expertStore = useExpertStore()
    const settingsStore = useSettingsStore()
    const samplingParams = loadSamplingParams()
    const useCustomModelPrompt = expertStore.isCustomModel(expertStore.selectedModel)

    const request = {
      chatId: currentChat?.id,
      message: messageText,
      model: expertStore.selectedModel,
      systemPrompt: useCustomModelPrompt ? null : expertStore.systemPrompt,
      stream: false,
      expertId: expertStore.selectedExpertId || null
    }

    if (useCustomModelPrompt) {
      console.log('🤖 Custom Model detected - skipping external system prompt')
    }

    // Add sampling parameters
    if (samplingParams && Object.keys(samplingParams).length > 0) {
      request.samplingParameters = samplingParams
    } else {
      request.maxTokens = settingsStore.settings.maxTokens
      request.temperature = settingsStore.settings.temperature
      request.topP = settingsStore.settings.topP
      request.topK = settingsStore.settings.topK
      request.repeatPenalty = settingsStore.settings.repeatPenalty
    }

    if (settingsStore.settings.cpuOnly) {
      request.cpuOnly = true
    }

    if (images.length > 0) {
      request.images = images
    }

    if (documentContext.trim()) {
      request.documentContext = documentContext.trim()
    }

    if (fileMetadata.length > 0) {
      request.fileMetadata = fileMetadata
    }

    // Web search settings
    if (webSearchEnabled) {
      request.webSearchEnabled = true
      request.includeSourceUrls = true
      request.maxSearchResults = 5
      isWebSearching.value = true
    }

    // Expert web search settings
    if (expertStore.selectedExpertId) {
      const expert = expertStore.getExpertById(expertStore.selectedExpertId)
      if (expert && expert.autoWebSearch) {
        request.webSearchEnabled = true
        request.includeSourceUrls = true
        request.maxSearchResults = expert.maxSearchResults || 5
        if (expert.webSearchShowLinks === false) {
          request.webSearchHideLinks = true
        }
        if (expert.searchDomains) {
          request.searchDomains = expert.searchDomains.split(',').map(d => d.trim()).filter(d => d)
        }
        isWebSearching.value = true
      }
    }

    const response = await api.sendMessage(request)
    currentRequestId.value = response.requestId

    if (!currentChat) {
      if (onChatCreated) onChatCreated(response.chatId)
    }

    const assistantMessage = {
      role: 'ASSISTANT',
      content: response.response,
      tokens: response.tokens,
      createdAt: new Date().toISOString(),
      downloadUrl: response.downloadUrl
    }
    addMessage(assistantMessage)

    currentRequestId.value = null
    if (onComplete) onComplete()
  }

  // Streaming message sending
  async function sendMessageStreaming(params) {
    const {
      messageText,
      images = [],
      documentContext = '',
      visionChainEnabled = false,
      visionModel = null,
      fileMetadata = [],
      webSearchEnabled = false,
      currentChat,
      addMessage,
      updateMessage,
      onChatCreated,
      onError,
      onComplete,
      onDelegation
    } = params

    return new Promise((resolve, reject) => {
      const baseURL = '/api/chat/send-stream'
      const expertStore = useExpertStore()
      const settingsStore = useSettingsStore()
      const samplingParams = loadSamplingParams()
      const useCustomModelPrompt = expertStore.isCustomModel(expertStore.selectedModel)

      const requestBody = {
        chatId: currentChat?.id,
        message: messageText,
        model: expertStore.selectedModel,
        systemPrompt: useCustomModelPrompt ? null : expertStore.systemPrompt,
        stream: true,
        expertId: expertStore.selectedExpertId || null
      }

      if (useCustomModelPrompt) {
        console.log('🤖 Custom Model detected - skipping external system prompt')
      }

      // Add sampling parameters
      if (samplingParams && Object.keys(samplingParams).length > 0) {
        requestBody.samplingParameters = samplingParams
      } else {
        requestBody.maxTokens = settingsStore.settings.maxTokens
        requestBody.temperature = settingsStore.settings.temperature
        requestBody.topP = settingsStore.settings.topP
        requestBody.topK = settingsStore.settings.topK
        requestBody.repeatPenalty = settingsStore.settings.repeatPenalty
      }

      if (settingsStore.settings.cpuOnly) {
        requestBody.cpuOnly = true
      }

      if (images.length > 0) {
        requestBody.images = images
      }

      if (documentContext.trim()) {
        requestBody.documentContext = documentContext.trim()
      }

      if (fileMetadata.length > 0) {
        requestBody.fileMetadata = fileMetadata
      }

      // Vision chaining
      if (visionChainEnabled && visionModel) {
        requestBody.visionChainEnabled = true
        requestBody.visionModel = visionModel
        try {
          const chainingSettingsStr = localStorage.getItem('chainingSettings')
          if (chainingSettingsStr) {
            const chainingSettings = JSON.parse(chainingSettingsStr)
            requestBody.showIntermediateOutput = chainingSettings.showIntermediateOutput || false
          }
        } catch (e) {
          requestBody.showIntermediateOutput = false
        }

        if (!expertStore.isCustomModel(expertStore.selectedModel)) {
          const deutschPrompt = 'Du antwortest IMMER auf Deutsch.'
          requestBody.systemPrompt = requestBody.systemPrompt
            ? deutschPrompt + '\n\n' + requestBody.systemPrompt
            : deutschPrompt
        }
      }

      // Web search settings
      if (webSearchEnabled) {
        requestBody.webSearchEnabled = true
        requestBody.includeSourceUrls = true
        requestBody.maxSearchResults = 5
        isWebSearching.value = true
      }

      // Expert web search settings
      if (expertStore.selectedExpertId) {
        const expert = expertStore.getExpertById(expertStore.selectedExpertId)
        if (expert && expert.autoWebSearch) {
          requestBody.webSearchEnabled = true
          requestBody.includeSourceUrls = true
          requestBody.maxSearchResults = expert.maxSearchResults || 5
          if (expert.webSearchShowLinks === false) {
            requestBody.webSearchHideLinks = true
          }
          if (expert.searchDomains) {
            requestBody.searchDomains = expert.searchDomains.split(',').map(d => d.trim()).filter(d => d)
          }
          isWebSearching.value = true
        }
      }

      secureFetch(baseURL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody)
      }).then(async response => {
        if (!response.ok) {
          let errorMessage = handleHttpError(response.status)
          if (onError) onError(errorMessage)
          throw new Error(errorMessage)
        }

        const reader = response.body.getReader()
        const decoder = new TextDecoder()

        const streamingMessage = reactive({
          role: 'ASSISTANT',
          content: '',
          tokens: 0,
          createdAt: new Date().toISOString(),
          isStreaming: true,
          modelName: expertStore.selectedModel
        })
        addMessage(streamingMessage)

        let buffer = ''

        function processChunk({ done, value }) {
          if (done) {
            streamingMessage.isStreaming = false
            isLoading.value = false
            isWebSearching.value = false
            if (onComplete) onComplete()
            resolve()
            return
          }

          buffer += decoder.decode(value, { stream: true })
          const lines = buffer.split('\n')
          buffer = lines.pop() || ''

          for (const line of lines) {
            if (line.startsWith('data:')) {
              const data = line.substring(5)

              try {
                const parsed = JSON.parse(data)
                handleSSEEvent(parsed, streamingMessage, {
                  onChatCreated,
                  updateMessage,
                  onDelegation
                })
              } catch (e) {
                // Plain text chunk
                streamingMessage.content += data
              }
            }
          }

          reader.read().then(processChunk).catch(err => {
            streamingMessage.isStreaming = false
            isLoading.value = false
            isWebSearching.value = false
            reject(err)
          })
        }

        reader.read().then(processChunk).catch(reject)
      }).catch(err => {
        console.error('Failed to initiate streaming:', err)
        reject(err)
      })
    })
  }

  // Handle SSE event types
  function handleSSEEvent(parsed, streamingMessage, callbacks) {
    const { onChatCreated, updateMessage, onDelegation } = callbacks
    const settingsStore = useSettingsStore()

    if (parsed.chatId) {
      // Start event
      console.log('[SSE] Start event - chatId:', parsed.chatId)
      if (onChatCreated) onChatCreated(parsed.chatId)
      currentRequestId.value = parsed.requestId
      if (parsed.isDocumentRequest) {
        streamingMessage.isDocumentRequest = true
        streamingMessage.documentType = parsed.documentType
      }
    } else if (parsed.tokens !== undefined) {
      // Done event
      const finalMessage = {
        role: streamingMessage.role,
        content: streamingMessage.content,
        tokens: parsed.tokens,
        createdAt: streamingMessage.createdAt,
        modelName: streamingMessage.modelName,
        isStreaming: false,
        isDocumentRequest: streamingMessage.isDocumentRequest || false,
        documentType: streamingMessage.documentType || null,
        downloadUrl: parsed.downloadUrl || null
      }

      // Update context usage
      if (parsed.totalChatTokens !== undefined) {
        contextUsage.value.totalChatTokens = parsed.totalChatTokens
        contextUsage.value.maxContextTokens = parsed.maxContextTokens || null
      }

      if (updateMessage) updateMessage(streamingMessage, finalMessage)
      currentRequestId.value = null
    } else if (parsed.type === 'mode_switch') {
      if (settingsStore.settings.showModeSwitchMessages) {
        const modeSwitchMsg = {
          role: 'SYSTEM',
          content: parsed.message,
          createdAt: new Date().toISOString(),
          isModeSwitchMessage: true
        }
        if (updateMessage) updateMessage(null, modeSwitchMsg, true)
      }
    } else if (parsed.type === 'model_swap') {
      handleModelSwap(parsed)
    } else if (parsed.type === 'delegation') {
      console.log('[SSE] Delegation:', parsed.expertName)
      const delegationMsg = {
        role: 'SYSTEM',
        content: `🔄 ${parsed.message || `Delegation an ${parsed.expertName}`}`,
        createdAt: new Date().toISOString(),
        isDelegationMessage: true,
        expertAvatar: parsed.expertAvatar
      }
      if (updateMessage) updateMessage(null, delegationMsg, true)
      if (onDelegation) {
        setTimeout(() => onDelegation(parsed.expertId, parsed.expertName), 500)
      }
    } else if (parsed.error) {
      console.error('Streaming error:', parsed.error)
    } else if (parsed.content !== undefined) {
      // Content chunk
      streamingMessage.content += parsed.content
    }
  }

  // Handle model swap events
  function handleModelSwap(parsed) {
    switch (parsed.status) {
      case 'starting':
        isSwappingModel.value = true
        modelSwapMessage.value = parsed.message || '🔄 Lade Modell...'
        modelSwapProgress.value = 10
        break
      case 'progress':
        modelSwapProgress.value = parsed.percent || 50
        modelSwapMessage.value = parsed.message || '🔄 Modell wird geladen...'
        break
      case 'complete':
        modelSwapProgress.value = 100
        modelSwapMessage.value = parsed.message || '✅ Modell bereit!'
        setTimeout(() => {
          isSwappingModel.value = false
          modelSwapMessage.value = ''
          modelSwapProgress.value = 0
        }, 500)
        break
      case 'error':
        isSwappingModel.value = false
        modelSwapMessage.value = ''
        modelSwapProgress.value = 0
        break
    }
  }

  // Handle HTTP errors with user-friendly messages
  function handleHttpError(status) {
    switch (status) {
      case 400:
        return 'Die Anfrage konnte nicht verarbeitet werden. Möglicherweise ist der Chat-Verlauf zu lang.'
      case 500:
        return 'Server-Fehler beim KI-Modell. Bitte versuche es erneut.'
      case 502:
      case 503:
      case 504:
        return 'Der KI-Server ist gerade nicht erreichbar. Bitte warte einen Moment.'
      case 401:
      case 403:
        return 'Nicht autorisiert. Bitte melde dich erneut an.'
      default:
        return `Unerwarteter Fehler (${status}). Bitte versuche es erneut.`
    }
  }

  return {
    // State
    isLoading,
    isWebSearching,
    isSwappingModel,
    modelSwapMessage,
    modelSwapProgress,
    currentRequestId,
    streamingEnabled,
    contextUsage,

    // Actions
    sendMessage,
    abortCurrentRequest,
    toggleStreaming
  }
})
