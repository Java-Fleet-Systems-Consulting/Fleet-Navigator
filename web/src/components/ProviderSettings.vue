<template>
  <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
    <div class="flex items-start justify-between mb-4">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          🖥️ llama-server
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ $t('providerSettings.subtitle') }}
        </p>
      </div>
      <button
        @click="refreshStatus"
        class="px-3 py-1 text-sm bg-blue-100 dark:bg-blue-800 text-blue-700 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700"
      >
        🔄 {{ $t('providerSettings.refresh') }}
      </button>
    </div>

    <!-- Status Badge -->
    <div class="mb-6 p-4 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
      <div class="flex items-center space-x-3">
        <div class="flex-shrink-0">
          <div class="w-3 h-3 rounded-full animate-pulse" :class="llamaServerOnline ? 'bg-green-500' : 'bg-red-500'"></div>
        </div>
        <div>
          <p class="text-sm font-medium text-blue-900 dark:text-blue-100">
            {{ $t('providerSettings.status') }}: <span class="font-bold">{{ llamaServerOnline ? $t('providerSettings.online') : $t('providerSettings.offline') }}</span>
          </p>
          <p class="text-xs text-blue-700 dark:text-blue-300">
            {{ $t('providerSettings.serverOnPort', { port: llamaServerPort }) }}
          </p>
        </div>
      </div>
    </div>

    <!-- FleetCode Info Box -->
    <div class="mb-6 p-4 bg-gradient-to-r from-cyan-50 to-blue-50 dark:from-cyan-900/20 dark:to-blue-900/20 border border-cyan-300 dark:border-cyan-700 rounded-lg">
      <div class="flex items-start gap-3">
        <span class="text-2xl">⚡</span>
        <div>
          <h4 class="font-semibold text-cyan-900 dark:text-cyan-100">{{ $t('providerSettings.fleetcodeTitle') }}</h4>
          <p class="text-sm text-cyan-700 dark:text-cyan-300 mt-1">
            {{ $t('providerSettings.fleetcodeDesc', { port: llamaServerPort }) }}
          </p>
          <div class="mt-2 flex items-center gap-2">
            <span class="text-xs px-2 py-1 rounded-full" :class="llamaServerOnline ? 'bg-green-500 text-white' : 'bg-red-500 text-white'">
              {{ llamaServerOnline ? '✓ ' + $t('providerSettings.serverOnline') : '✗ ' + $t('providerSettings.serverOffline') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- llama-server Configuration Card -->
    <div class="mb-6 p-4 border-2 border-green-500 bg-green-50 dark:bg-green-900/20 rounded-lg">
      <div class="flex items-center justify-between mb-3">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
          🖥️ {{ $t('providerSettings.serverConfig') }}
          <span v-if="llamaServerOnline" class="text-xs px-2 py-0.5 bg-green-500 text-white rounded-full">{{ $t('providerSettings.active') }}</span>
        </h3>
        <div class="flex items-center gap-2">
          <span v-if="llamaServerPid" class="text-xs text-gray-500 dark:text-gray-400">
            PID: {{ llamaServerPid }}
          </span>
          <div class="w-3 h-3 rounded-full" :class="llamaServerOnline ? 'bg-green-500 animate-pulse' : 'bg-red-500'"></div>
        </div>
      </div>

      <!-- Aktuell laufendes Modell -->
      <div v-if="currentModel" class="mb-4 p-3 bg-green-100 dark:bg-green-900/40 border border-green-300 dark:border-green-700 rounded-lg">
        <p class="text-sm text-green-800 dark:text-green-200">
          <strong>🟢 {{ $t('providerSettings.running') }}:</strong> {{ currentModel.split('/').pop() }}
        </p>
      </div>

      <!-- Configuration Grid -->
      <div class="space-y-4 mt-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <!-- Modell-Auswahl -->
          <div class="md:col-span-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              🦙 {{ $t('providerSettings.selectModel') }}
            </label>
            <select
              v-model="selectedModel"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              :disabled="llamaServerOnline"
            >
              <option value="" disabled>-- {{ $t('providerSettings.chooseModel') }} --</option>
              <option v-for="model in availableModels" :key="getModelName(model)" :value="getModelPath(model)">
                {{ getModelName(model) }}
              </option>
            </select>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('providerSettings.modelsAvailable', { count: availableModels.length }) }}
            </p>
          </div>

          <!-- Port -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              🔌 {{ $t('providerSettings.serverPort') }}
            </label>
            <input
              v-model.number="llamaServerPort"
              type="number"
              min="1024"
              max="65535"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              placeholder="2026"
              :disabled="llamaServerOnline"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('providerSettings.defaultPort') }}
            </p>
          </div>

          <!-- Status -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              📊 {{ $t('providerSettings.serverStatus') }}
            </label>
            <div class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-800">
              <span v-if="checkingLlamaServer || startingServer" class="text-sm text-gray-500">
                🔄 {{ startingServer ? $t('providerSettings.starting') : $t('providerSettings.checking') }}
              </span>
              <span v-else-if="llamaServerOnline" class="text-sm text-green-600 dark:text-green-400 font-medium">
                ✓ {{ $t('providerSettings.onlineOnPort', { port: llamaServerPort }) }}
              </span>
              <span v-else class="text-sm text-red-600 dark:text-red-400 font-medium">
                ✗ {{ $t('providerSettings.offline') }}
              </span>
            </div>
          </div>

          <!-- Context Size -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              📝 {{ $t('providerSettings.contextSize') }}
            </label>
            <input
              v-model.number="contextSize"
              type="number"
              min="512"
              max="131072"
              step="512"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              :disabled="llamaServerOnline"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('providerSettings.contextSizeHint') }}
            </p>
          </div>

          <!-- GPU Layers -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              🎮 {{ $t('providerSettings.gpuLayers') }}
            </label>
            <input
              v-model.number="gpuLayers"
              type="number"
              min="0"
              max="999"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              :disabled="llamaServerOnline"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('providerSettings.gpuLayersHint') }}
            </p>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex flex-wrap gap-3 mt-4">
          <!-- Start Button -->
          <button
            v-if="!llamaServerOnline"
            @click="startLlamaServer"
            :disabled="startingServer || !selectedModel"
            class="px-4 py-2 bg-green-500 hover:bg-green-600 text-white rounded-lg text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ startingServer ? '🔄 ' + $t('providerSettings.starting') : '▶️ ' + $t('providerSettings.startServer') }}
          </button>

          <!-- Stop Button -->
          <button
            v-if="llamaServerOnline"
            @click="stopLlamaServer"
            :disabled="stoppingServer"
            class="px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {{ stoppingServer ? '🔄 ' + $t('providerSettings.stopping') : '⏹️ ' + $t('providerSettings.stopServer') }}
          </button>

          <!-- Status prüfen -->
          <button
            @click="checkLlamaServerStatus"
            :disabled="checkingLlamaServer"
            class="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            🔍 {{ $t('providerSettings.checkStatus') }}
          </button>

          <!-- Restart Button (nur wenn online) -->
          <button
            v-if="llamaServerOnline"
            @click="restartLlamaServer"
            :disabled="restartingServer"
            class="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-lg text-sm font-medium disabled:opacity-50"
          >
            {{ restartingServer ? '🔄 ' + $t('providerSettings.restarting') : '🔄 ' + $t('providerSettings.restart') }}
          </button>
        </div>

        <!-- Info Box -->
        <div v-if="!llamaServerOnline && availableModels.length === 0" class="mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/30 border border-yellow-300 dark:border-yellow-700 rounded-lg">
          <p class="text-xs text-yellow-800 dark:text-yellow-200">
            ⚠️ <strong>{{ $t('providerSettings.noModelsFound') }}</strong> {{ $t('providerSettings.pleaseDownload') }}
          </p>
        </div>

        <!-- Success Info -->
        <div v-if="llamaServerOnline" class="mt-4 p-3 bg-green-50 dark:bg-green-900/30 border border-green-300 dark:border-green-700 rounded-lg">
          <p class="text-xs text-green-800 dark:text-green-200">
            ✅ <strong>{{ $t('providerSettings.serverRunning') }}</strong> {{ $t('providerSettings.serverAvailable') }}
          </p>
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="flex justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'
import axios from 'axios'

const { t } = useI18n()

// State
const llamaServerPort = ref(2026)
const llamaServerOnline = ref(false)
const checkingLlamaServer = ref(false)
const restartingServer = ref(false)
const startingServer = ref(false)
const stoppingServer = ref(false)
const llamaServerPid = ref(null)
const currentModel = ref(null)
const availableModels = ref([])
const selectedModel = ref('')
const contextSize = ref(8192)
const gpuLayers = ref(99)
const loading = ref(false)

// Helper functions for model objects
function getModelName(model) {
  if (typeof model === 'string') return model
  return model?.name || model?.path?.split('/').pop() || 'Unknown'
}

function getModelPath(model) {
  if (typeof model === 'string') return model
  return model?.path || model?.name || ''
}

// Methods
async function refreshStatus() {
  await loadLlamaServerStatus()
  await checkLlamaServerStatus()
  await loadAvailableModels()
}

async function checkLlamaServerStatus() {
  checkingLlamaServer.value = true
  try {
    const result = await api.checkLlamaServerHealth(llamaServerPort.value || 2026)
    llamaServerOnline.value = result.online === true
  } catch (error) {
    console.log('llama-server check failed:', error.message)
    llamaServerOnline.value = false
  } finally {
    checkingLlamaServer.value = false
  }
}

async function restartLlamaServer() {
  restartingServer.value = true
  try {
    await axios.post(`/api/llm/providers/llama-server/restart`, {
      port: llamaServerPort.value,
      model: selectedModel.value || null,
      contextSize: contextSize.value,
      gpuLayers: gpuLayers.value
    })
    setTimeout(async () => {
      await checkLlamaServerStatus()
      await loadLlamaServerStatus()
      restartingServer.value = false
    }, 3000)
  } catch (error) {
    console.error('Failed to restart llama-server:', error)
    restartingServer.value = false
  }
}

async function startLlamaServer() {
  if (!selectedModel.value) {
    alert(t('providerSettings.pleaseSelectModel'))
    return
  }

  startingServer.value = true
  try {
    const response = await axios.post(`/api/llm/providers/llama-server/start`, {
      model: selectedModel.value,
      port: llamaServerPort.value,
      contextSize: contextSize.value,
      gpuLayers: gpuLayers.value
    })

    if (response.data.success) {
      llamaServerPid.value = response.data.pid
      currentModel.value = response.data.model

      setTimeout(async () => {
        await checkLlamaServerStatus()
        await loadLlamaServerStatus()
        startingServer.value = false
      }, 2000)
    } else {
      alert(response.data.message)
      startingServer.value = false
    }
  } catch (error) {
    console.error('Failed to start llama-server:', error)
    alert(t('providerSettings.startError') + ': ' + (error.response?.data?.message || error.message))
    startingServer.value = false
  }
}

async function stopLlamaServer() {
  stoppingServer.value = true
  try {
    const response = await axios.post(`/api/llm/providers/llama-server/stop`)

    if (response.data.success) {
      llamaServerOnline.value = false
      llamaServerPid.value = null
      currentModel.value = null
    }

    setTimeout(async () => {
      await checkLlamaServerStatus()
      stoppingServer.value = false
    }, 1000)
  } catch (error) {
    console.error('Failed to stop llama-server:', error)
    stoppingServer.value = false
  }
}

async function loadLlamaServerStatus() {
  try {
    const response = await axios.get(`/api/llm/providers/llama-server/status`)
    llamaServerOnline.value = response.data.online
    llamaServerPid.value = response.data.pid || null
    currentModel.value = response.data.model || null
    if (response.data.port) {
      llamaServerPort.value = response.data.port
    }
  } catch (error) {
    console.log('llama-server status check failed:', error.message)
  }
}

async function loadAvailableModels() {
  try {
    const response = await axios.get(`/api/llm/providers/llama-server/models`)
    availableModels.value = response.data.models || []

    if (currentModel.value) {
      const modelName = currentModel.value.split('/').pop()
      const matchingModel = availableModels.value.find(m =>
        getModelName(m) === modelName || getModelPath(m).endsWith(modelName)
      )
      if (matchingModel) {
        selectedModel.value = getModelPath(matchingModel)
      }
    } else if (availableModels.value.length > 0 && !selectedModel.value) {
      selectedModel.value = getModelPath(availableModels.value[0])
    }
  } catch (error) {
    console.log('Failed to load available models:', error.message)
  }
}

// Status-Polling (alle 3 Sekunden)
let statusPollInterval = null

onMounted(async () => {
  loading.value = true
  await loadLlamaServerStatus()
  await checkLlamaServerStatus()
  await loadAvailableModels()
  loading.value = false

  statusPollInterval = setInterval(async () => {
    if (!startingServer.value && !stoppingServer.value) {
      await loadLlamaServerStatus()
      await checkLlamaServerStatus()
    }
  }, 3000)
})

onUnmounted(() => {
  if (statusPollInterval) {
    clearInterval(statusPollInterval)
    statusPollInterval = null
  }
})
</script>
