<template>
  <div class="model-store bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">

    <!-- Download Modal -->
    <ModelDownloadModal
      :isVisible="showDownloadModal"
      :currentModel="currentDownloadModel"
      :progress="currentDownloadProgress"
      :downloadedSize="currentDownloadedSize"
      :totalSize="currentTotalSize"
      :speed="currentSpeed"
      :statusMessages="downloadStatusMessages"
      @cancel="cancelCurrentDownload"
    />

    <!-- Header -->
    <div class="flex items-start justify-between mb-6">
      <div>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white">
          🏪 {{ $t('modelStore.title') }}
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ $t('modelStore.subtitle') }}
        </p>
      </div>
      <button
        @click="loadModels"
        class="px-4 py-2 text-sm bg-blue-100 dark:bg-blue-800 text-blue-700 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700"
      >
        🔄 {{ $t('modelStore.refresh') }}
      </button>
    </div>

    <!-- Filter & Search -->
    <div class="mb-6 flex flex-wrap gap-3">
      <div class="flex-1 min-w-[200px]">
        <input
          v-model="searchQuery"
          type="text"
          :placeholder="$t('modelStore.searchPlaceholder')"
          class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
        />
      </div>
      <select
        v-model="filterCategory"
        class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
      >
        <option value="">{{ $t('modelStore.allCategories') }}</option>
        <option value="chat">💬 {{ $t('modelStore.chatAssistants') }}</option>
        <option value="code">💻 {{ $t('modelStore.codeGeneration') }}</option>
        <option value="compact">📦 {{ $t('modelStore.compactModels') }}</option>
      </select>
      <select
        v-model="filterRam"
        class="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
      >
        <option :value="0">{{ $t('modelStore.allRamSizes') }}</option>
        <option :value="4">{{ $t('modelStore.maxRam', { size: 4 }) }}</option>
        <option :value="8">{{ $t('modelStore.maxRam', { size: 8 }) }}</option>
        <option :value="16">{{ $t('modelStore.maxRam', { size: 16 }) }}</option>
      </select>
    </div>

    <!-- Featured Models -->
    <div v-if="!searchQuery && featuredModels.length > 0" class="mb-8">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        ⭐ {{ $t('modelStore.featuredModels') }}
      </h3>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <ModelCard
          v-for="model in featuredModels"
          :key="model.id"
          :model="model"
          :downloaded="isDownloaded(model.filename)"
          :downloading="isDownloading(model.id)"
          :progress="downloadProgress[model.id]"
          :disabled="showDownloadModal"
          @download="startDownload(model.id)"
          @cancel="cancelDownload(model.id)"
          @delete="deleteModel(model.id)"
          @select="selectModel"
        />
      </div>
    </div>

    <!-- All Models -->
    <div>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
        {{ searchQuery ? `🔍 ${$t('modelStore.searchResults')} (${filteredModels.length})` : `📚 ${$t('modelStore.allModels')} (${filteredModels.length})` }}
      </h3>
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <ModelCard
          v-for="model in filteredModels"
          :key="model.id"
          :model="model"
          :downloaded="isDownloaded(model.filename)"
          :downloading="isDownloading(model.id)"
          :progress="downloadProgress[model.id]"
          :disabled="showDownloadModal"
          @download="startDownload(model.id)"
          @cancel="cancelDownload(model.id)"
          @delete="deleteModel(model.id)"
          @select="selectModel"
        />
      </div>
      <div v-if="filteredModels.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
        {{ $t('modelStore.noModelsFound') }}
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import ModelCard from './ModelCard.vue'
import ModelDownloadModal from './ModelDownloadModal.vue'
import { useChatStore } from '../stores/chatStore'
import { useConfirmDialog } from '../composables/useConfirmDialog'

const { t } = useI18n()
const chatStore = useChatStore()
const { confirmDelete } = useConfirmDialog()

const allModels = ref([])
const featuredModels = ref([])
const downloadedModels = ref(new Set())
const searchQuery = ref('')
const filterCategory = ref('')
const filterRam = ref(0)

const downloadProgress = ref({})
const activeEventSources = ref({})

// Modal state
const showDownloadModal = ref(false)
const currentDownloadModel = ref('')
const currentDownloadProgress = ref(0)
const currentDownloadedSize = ref('0 MB')
const currentTotalSize = ref('0 MB')
const currentSpeed = ref('0.0')
const downloadStatusMessages = ref([])
const currentDownloadModelId = ref('')

onMounted(async () => {
  await loadModels()
  await loadDownloadedModels()
})

onUnmounted(() => {
  // Close all active EventSource connections
  Object.values(activeEventSources.value).forEach(source => source.close())
})

async function loadModels() {
  try {
    const [allResponse, featuredResponse] = await Promise.all([
      axios.get('/api/model-store/all'),
      axios.get('/api/model-store/featured')
    ])
    allModels.value = allResponse.data
    featuredModels.value = featuredResponse.data
  } catch (error) {
    console.error('Failed to load models:', error)
  }
}

async function loadDownloadedModels() {
  try {
    const response = await axios.get('/api/llm/models')
    const llamacppModels = response.data.filter(m => m.provider === 'llamacpp')
    downloadedModels.value = new Set(llamacppModels.map(m => m.name))
  } catch (error) {
    console.error('Failed to load downloaded models:', error)
  }
}

const filteredModels = computed(() => {
  let models = allModels.value

  // Category filter
  if (filterCategory.value) {
    models = models.filter(m => m.category === filterCategory.value)
  }

  // RAM filter
  if (filterRam.value > 0) {
    models = models.filter(m => m.minRamGB <= filterRam.value)
  }

  // Search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    models = models.filter(m =>
      m.displayName.toLowerCase().includes(query) ||
      m.description.toLowerCase().includes(query) ||
      m.languages.some(lang => lang.toLowerCase().includes(query)) ||
      m.useCases.some(use => use.toLowerCase().includes(query))
    )
  }

  return models
})

function isDownloaded(filename) {
  return downloadedModels.value.has(filename)
}

function isDownloading(modelId) {
  return modelId in downloadProgress.value
}

async function startDownload(modelId) {
  // Prevent multiple downloads
  if (showDownloadModal.value) {
    alert(t('modelStore.downloadInProgress'))
    return
  }

  console.log('Starting download for:', modelId)

  // Get model info
  const model = allModels.value.find(m => m.id === modelId)
  if (!model) {
    alert(t('modelStore.modelNotFound'))
    return
  }

  // =========================================================
  // IDIOTENSICHERE GPU-PRUEFUNG VOR DOWNLOAD
  // =========================================================
  try {
    const gpuCheckResponse = await axios.get(`/api/model-store/${modelId}/check-gpu`)
    const gpuCheck = gpuCheckResponse.data

    console.log('GPU Check Result:', gpuCheck)

    if (!gpuCheck.canRun) {
      // BLOCKIEREN: Modell passt NICHT auf die GPU
      const proceed = confirm(
        `${t('modelStore.warningModelTooLarge')}\n\n` +
        `${gpuCheck.warning}\n\n` +
        `${t('modelStore.yourGpu')} ${gpuCheck.gpuName} (${gpuCheck.gpuTotalGB.toFixed(1)} GB VRAM)\n` +
        `${t('modelStore.modelRequires')} ca. ${gpuCheck.estimatedVramGB.toFixed(1)} GB VRAM\n\n` +
        `${t('modelStore.downloadAnyway')}\n` +
        `${t('modelStore.cpuModeWarning')}`
      )

      if (!proceed) {
        console.log('Download cancelled by user due to GPU incompatibility')
        return
      }
    } else if (gpuCheck.hasWarning && gpuCheck.warning) {
      // WARNUNG: Aktuell wenig freier VRAM
      const proceed = confirm(
        `${t('modelStore.note')}\n\n${gpuCheck.warning}\n\n` +
        `${t('modelStore.continueAnyway')}`
      )

      if (!proceed) {
        console.log('Download cancelled by user due to low free VRAM')
        return
      }
    }
  } catch (error) {
    console.warn('GPU check failed, proceeding with download:', error)
    // Bei Fehler in der GPU-Pruefung trotzdem fortfahren
  }

  // Show modal
  showDownloadModal.value = true
  currentDownloadModelId.value = modelId
  currentDownloadModel.value = model.displayName
  currentDownloadProgress.value = 0
  currentDownloadedSize.value = '0 MB'
  currentTotalSize.value = model.sizeHuman
  currentSpeed.value = '0.0'
  downloadStatusMessages.value = ['📥 ' + t('modelStore.startingDownload')]

  // Create EventSource for SSE
  const eventSource = new EventSource(`/api/model-store/download/${modelId}`)
  activeEventSources.value[modelId] = eventSource

  // Initialize progress
  downloadProgress.value[modelId] = {
    displayName: model?.displayName || modelId,
    percentComplete: 0,
    speedMBps: 0
  }

  eventSource.addEventListener('progress', (event) => {
    const message = event.data
    console.log('Progress:', message)

    // Add to status messages
    downloadStatusMessages.value.push(message)

    // Parse progress from message like "⬇️ 45% - 1.2 GB / 2.0 GB - 5.3 MB/s"
    const percentMatch = message.match(/(\d+)%/)
    const downloadedMatch = message.match(/([\d.]+\s+[GM]B)\s+\//)
    const totalMatch = message.match(/\/\s+([\d.]+\s+[GM]B)/)
    const speedMatch = message.match(/([\d.]+)\s+MB\/s/)

    if (percentMatch) {
      const percent = parseInt(percentMatch[1])
      downloadProgress.value[modelId].percentComplete = percent
      currentDownloadProgress.value = percent
    }
    if (downloadedMatch) {
      currentDownloadedSize.value = downloadedMatch[1]
    }
    if (totalMatch) {
      currentTotalSize.value = totalMatch[1]
    }
    if (speedMatch) {
      const speed = parseFloat(speedMatch[1])
      downloadProgress.value[modelId].speedMBps = speed
      currentSpeed.value = speed.toFixed(1)
    }
  })

  eventSource.addEventListener('complete', (event) => {
    console.log('Download complete:', event.data)
    downloadStatusMessages.value.push('✅ ' + event.data)

    delete downloadProgress.value[modelId]
    eventSource.close()
    delete activeEventSources.value[modelId]

    // Reload downloaded models
    loadDownloadedModels()

    // Close modal after 2 seconds
    setTimeout(() => {
      showDownloadModal.value = false
      currentDownloadModelId.value = ''
    }, 2000)

    // Show success notification
    alert('✅ ' + event.data)
  })

  eventSource.addEventListener('error', (event) => {
    console.error('Download error:', event)
    downloadStatusMessages.value.push('❌ ' + t('modelStore.downloadFailed'))

    delete downloadProgress.value[modelId]
    eventSource.close()
    delete activeEventSources.value[modelId]

    showDownloadModal.value = false
    currentDownloadModelId.value = ''

    alert('❌ ' + t('modelStore.downloadFailed'))
  })
}

async function cancelDownload(modelId) {
  try {
    await axios.post(`/api/model-store/download/${modelId}/cancel`)

    // Close EventSource
    if (activeEventSources.value[modelId]) {
      activeEventSources.value[modelId].close()
      delete activeEventSources.value[modelId]
    }

    delete downloadProgress.value[modelId]
  } catch (error) {
    console.error('Failed to cancel download:', error)
  }
}

function cancelCurrentDownload() {
  if (currentDownloadModelId.value) {
    cancelDownload(currentDownloadModelId.value)
    showDownloadModal.value = false
    currentDownloadModelId.value = ''
  }
}

async function selectModel(filename) {
  try {
    // Auto-switch to llama.cpp provider when selecting a GGUF model
    await axios.post('/api/settings/llm-provider', { provider: 'llamacpp' })
    console.log('🔌 Switched to llama.cpp provider')

    // Set the selected model
    await chatStore.setSelectedModel(filename)
    console.log('✅ Model selected:', filename)

    alert('✅ ' + t('modelStore.modelSelected', { name: filename }))
  } catch (error) {
    console.error('Failed to select model:', error)
    alert('❌ ' + t('modelStore.selectError'))
  }
}

async function deleteModel(modelId) {
  const model = allModels.value.find(m => m.id === modelId)
  const confirmed = await confirmDelete(model?.displayName || t('models.title'))
  if (!confirmed) return

  try {
    await axios.delete(`/api/model-store/${modelId}`)
    await loadDownloadedModels()
    alert('✅ ' + t('modelStore.modelDeleted'))
  } catch (error) {
    console.error('Failed to delete model:', error)
    alert('❌ ' + t('modelStore.deleteError'))
  }
}
</script>

<style scoped>
/* Additional custom styles if needed */
</style>
