import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import api from '../services/api'

const PROMPT_STORAGE_KEY = 'fleet-navigator-last-prompt'
const MODEL_STORAGE_KEY = 'fleet-navigator-selected-model'

export const useExpertStore = defineStore('expert', () => {
  // Load last used system prompt from localStorage
  const loadLastPrompt = () => {
    try {
      const stored = localStorage.getItem(PROMPT_STORAGE_KEY)
      if (stored) {
        const { content, title } = JSON.parse(stored)
        console.log('✅ Loaded last system prompt:', title)
        return { content, title }
      }
    } catch (e) {
      console.error('Failed to load last prompt', e)
    }
    return { content: null, title: null }
  }

  const lastPrompt = loadLastPrompt()

  // State
  const models = ref([])
  const experts = ref([])
  const customModels = ref([])
  const selectedModel = ref(null)
  const selectedExpertId = ref(null)
  const systemPrompt = ref(lastPrompt.content)
  const systemPromptTitle = ref(lastPrompt.title)
  const isSwitchingExpert = ref(false)
  const switchingExpertMessage = ref('')

  // Watch system prompt changes and save to localStorage
  watch([systemPrompt, systemPromptTitle], ([newPrompt, newTitle]) => {
    if (newPrompt) {
      try {
        localStorage.setItem(PROMPT_STORAGE_KEY, JSON.stringify({
          content: newPrompt,
          title: newTitle
        }))
        console.log('💾 Saved system prompt:', newTitle || '(custom)')
      } catch (e) {
        console.error('Failed to save prompt', e)
      }
    }
  })

  // Watch selected model changes and save to localStorage
  watch(selectedModel, (newModel) => {
    if (newModel) {
      try {
        localStorage.setItem(MODEL_STORAGE_KEY, newModel)
        console.log('💾 Saved selected model:', newModel)
      } catch (e) {
        console.error('Failed to save selected model', e)
      }
    }
  })

  // Helper function: Check if model is a custom model
  function isCustomModel(modelName) {
    if (!modelName) return false
    return customModels.value.some(cm =>
      cm.name === modelName ||
      cm.name.toLowerCase() === modelName.toLowerCase()
    )
  }

  // Get expert by ID (uses Number() for type-safe comparison)
  function getExpertById(expertId) {
    if (!expertId) return null
    const numId = Number(expertId)
    return experts.value.find(e => Number(e.id) === numId) || null
  }

  // Build combined prompt from expert
  function buildExpertPrompt(expert) {
    let combinedPrompt = expert.basePrompt
    if (expert.personalityPrompt && expert.personalityPrompt.trim()) {
      combinedPrompt = `${expert.basePrompt}\n\n## Kommunikationsstil:\n${expert.personalityPrompt}`
    }
    return combinedPrompt
  }

  // Load models and experts
  async function loadModels() {
    try {
      models.value = await api.getAvailableModels()

      // Load custom models from database
      try {
        const customModelsData = await api.getAllCustomModels()
        customModels.value = customModelsData || []
        console.log(`📦 Loaded ${customModels.value.length} custom model configurations from database`)
      } catch (e) {
        console.warn('⚠️ Could not load custom models:', e.message)
        customModels.value = []
      }

      // Load experts
      try {
        const expertsData = await api.getAllExperts()
        experts.value = expertsData || []
        console.log(`🎓 Loaded ${experts.value.length} experts for model selection`)
      } catch (e) {
        console.warn('⚠️ Could not load experts:', e.message)
        experts.value = []
      }

      // Load user's last selected expert from database (FIRST, before model)
      try {
        const savedExpertId = await api.getSelectedExpert()
        if (savedExpertId) {
          const numId = Number(savedExpertId)
          const expert = experts.value.find(e => Number(e.id) === numId)
          if (expert) {
            selectedExpertId.value = expert.id
            selectedModel.value = expert.baseModel
            systemPrompt.value = buildExpertPrompt(expert)
            systemPromptTitle.value = `🎓 ${expert.name}`
            console.log(`🎓 Restored selected expert from database: ${expert.name} (${expert.role})`)
          } else {
            console.warn(`⚠️ Saved expert ID ${savedExpertId} not found in experts list`)
          }
        }
      } catch (e) {
        console.warn('⚠️ Could not load selected expert:', e.message)
      }

      // Load model from database (only if no expert selected)
      if (!selectedExpertId.value) {
        const savedModel = await api.getSelectedModel()
        if (savedModel) {
          selectedModel.value = savedModel
          console.log('✅ Loaded selected model from database:', savedModel)
        } else {
          const defaultModelResponse = await api.getDefaultModel()
          if (defaultModelResponse && defaultModelResponse.model) {
            selectedModel.value = defaultModelResponse.model
            console.log('📥 Using backend default model:', defaultModelResponse.model)
          }
        }
      }

      // Sync system prompt with database (only if no expert selected)
      if (!selectedExpertId.value) {
        try {
          console.log('🔄 Synchronizing system prompt with database...')
          const defaultPrompt = await api.getDefaultSystemPrompt()
          if (defaultPrompt && defaultPrompt.content) {
            systemPrompt.value = defaultPrompt.content
            systemPromptTitle.value = defaultPrompt.name
            localStorage.setItem(PROMPT_STORAGE_KEY, JSON.stringify({
              content: defaultPrompt.content,
              title: defaultPrompt.name
            }))
            console.log('✅ System prompt synchronized with DB:', defaultPrompt.name)
          }
        } catch (promptError) {
          console.warn('⚠️ Could not sync system prompt from DB, using localStorage:', promptError.message)
        }
      } else {
        console.log('🎓 Skipping system prompt sync - expert selected')
      }
    } catch (err) {
      console.error('Failed to load models:', err)
      throw err
    }
  }

  // Set selected model (clears expert)
  async function setSelectedModel(model, currentChatId = null) {
    selectedModel.value = model
    selectedExpertId.value = null

    // Reset system prompt to default
    try {
      const defaultPrompt = await api.getDefaultSystemPrompt()
      if (defaultPrompt && defaultPrompt.content) {
        systemPrompt.value = defaultPrompt.content
        systemPromptTitle.value = defaultPrompt.name || 'Standard'
        console.log('🔄 Reset system prompt to default:', defaultPrompt.name)
      } else {
        systemPrompt.value = 'Du bist ein hilfreicher Assistent.'
        systemPromptTitle.value = 'Standard'
      }
    } catch (e) {
      console.error('Failed to load default system prompt', e)
      systemPrompt.value = 'Du bist ein hilfreicher Assistent.'
      systemPromptTitle.value = 'Standard'
    }

    // Save to database
    try {
      await api.saveSelectedModel(model)
      await api.saveSelectedExpert(null)
      console.log('💾 Saved selected model to database:', model)
    } catch (e) {
      console.error('Failed to save model to database', e)
    }

    // Update current chat if provided
    if (currentChatId) {
      try {
        await api.updateChatModel(currentChatId, model)
        await api.updateChatExpert(currentChatId, null)
        console.log('🎓 Updated current chat model and cleared expert')
      } catch (e) {
        console.error('Failed to update chat model/expert', e)
      }
    }
  }

  // Select an expert with context change handling
  async function selectExpert(expert, currentChatId = null) {
    if (!expert) {
      selectedExpertId.value = null
      try {
        await api.saveSelectedExpert(null)
        if (currentChatId) {
          await api.updateChatExpert(currentChatId, null)
          console.log('🎓 Cleared expert from current chat')
        }
      } catch (e) {
        console.error('Failed to clear expert in database', e)
      }
      return
    }

    const modelName = expert.model || expert.baseModel

    // Check if context change is needed
    try {
      const contextInfo = await api.getModelContextInfo(modelName)
      console.log(`📊 Context info for ${modelName}:`, contextInfo)

      if (contextInfo.restartNeeded) {
        isSwitchingExpert.value = true
        switchingExpertMessage.value = `Wechsle zu ${expert.name}...`

        console.log(`🔄 Context change needed: ${contextInfo.currentContext} → ${contextInfo.effectiveContext}`)

        const result = await api.changeContextSize(contextInfo.effectiveContext)

        if (result.success) {
          console.log(`✅ Context changed to ${result.contextSize}`)
          if (result.estimatedSeconds > 0) {
            switchingExpertMessage.value = `${expert.name} wird geladen...`
            await new Promise(resolve => setTimeout(resolve, result.estimatedSeconds * 1000))
          }
        } else {
          console.error('Context change failed:', result.error)
        }
      }
    } catch (e) {
      console.error('Failed to check/change context:', e)
    } finally {
      isSwitchingExpert.value = false
      switchingExpertMessage.value = ''
    }

    selectedExpertId.value = expert.id
    selectedModel.value = modelName
    systemPrompt.value = buildExpertPrompt(expert)
    systemPromptTitle.value = `🎓 ${expert.name}`

    console.log(`🎓 Selected expert: ${expert.name} (${expert.role}) using model: ${modelName}`)

    // Save to database
    try {
      await api.saveSelectedExpert(expert.id)
      await api.saveSelectedModel(modelName)
      console.log(`💾 Saved selected expert to database: ${expert.name} (ID: ${expert.id})`)
    } catch (e) {
      console.error('Failed to save expert to database', e)
    }

    // Update current chat if provided
    if (currentChatId) {
      try {
        await api.updateChatModel(currentChatId, modelName)
        await api.updateChatExpert(currentChatId, expert.id)
        console.log(`💾 Updated current chat expert to: ${expert.name}`)
      } catch (e) {
        console.error('Failed to update chat expert', e)
      }
    }

    return { maxContextTokens: null } // Return for contextUsage update if needed
  }

  // Restore expert from chat (when loading chat history)
  async function restoreExpertFromChat(chatExpertId) {
    if (!chatExpertId) {
      if (selectedExpertId.value !== null) {
        console.log('[Chat] Clearing expert selection for non-expert chat')
        selectedExpertId.value = null
        try {
          const defaultPrompt = await api.getDefaultSystemPrompt()
          if (defaultPrompt && defaultPrompt.content) {
            systemPrompt.value = defaultPrompt.content
            systemPromptTitle.value = defaultPrompt.name || 'Standard'
            console.log('🔄 Restored default system prompt:', defaultPrompt.name)
          }
        } catch (e) {
          console.error('Failed to load default system prompt', e)
          systemPrompt.value = 'Du bist ein hilfreicher Assistent.'
          systemPromptTitle.value = 'Standard'
        }
      }
      return null
    }

    const chatExpertIdNum = Number(chatExpertId)
    const expert = experts.value.find(e => Number(e.id) === chatExpertIdNum)

    console.log(`🔍 Looking for expert ID ${chatExpertId} (type: ${typeof chatExpertId})`)

    if (expert) {
      selectedExpertId.value = expert.id
      selectedModel.value = expert.baseModel
      systemPrompt.value = buildExpertPrompt(expert)
      systemPromptTitle.value = `🎓 ${expert.name}`
      console.log(`🎓 Restored expert from chat: ${expert.name} (ID: ${expert.id})`)
      return expert
    } else {
      console.warn(`Expert with ID ${chatExpertId} not found in loaded experts`)
      selectedExpertId.value = null
      try {
        const defaultPrompt = await api.getDefaultSystemPrompt()
        if (defaultPrompt && defaultPrompt.content) {
          systemPrompt.value = defaultPrompt.content
          systemPromptTitle.value = defaultPrompt.name || 'Standard'
        }
      } catch (e) {
        console.error('Failed to load default system prompt', e)
      }
      return null
    }
  }

  // Set system prompt manually
  function setSystemPrompt(prompt, title = null) {
    systemPrompt.value = prompt
    systemPromptTitle.value = title
    localStorage.setItem(PROMPT_STORAGE_KEY, JSON.stringify({
      content: prompt,
      title: title
    }))
    console.log('✅ System prompt saved to localStorage:', title)
  }

  return {
    // State
    models,
    experts,
    customModels,
    selectedModel,
    selectedExpertId,
    systemPrompt,
    systemPromptTitle,
    isSwitchingExpert,
    switchingExpertMessage,

    // Helper Functions
    isCustomModel,
    getExpertById,
    buildExpertPrompt,

    // Actions
    loadModels,
    setSelectedModel,
    selectExpert,
    restoreExpertFromChat,
    setSystemPrompt
  }
})
