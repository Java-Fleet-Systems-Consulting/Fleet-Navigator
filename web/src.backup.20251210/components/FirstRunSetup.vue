<template>
  <!-- Full-Screen Modal - kann nicht geschlossen werden -->
  <div class="fixed inset-0 bg-gradient-to-br from-gray-900 via-blue-900 to-gray-900 flex items-center justify-center z-[200] p-4">

    <!-- Setup Card -->
    <div class="bg-white dark:bg-gray-800 rounded-3xl shadow-2xl w-full max-w-2xl overflow-hidden">

      <!-- Header mit Logo -->
      <div class="bg-gradient-to-r from-blue-600 to-blue-700 px-8 py-8 text-center">
        <div class="flex justify-center mb-4">
          <div class="p-4 bg-white/20 rounded-2xl">
            <svg class="w-16 h-16 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
        </div>
        <h1 class="text-3xl font-bold text-white mb-2">Willkommen bei Fleet Navigator</h1>
        <p class="text-blue-100 text-lg">Das Experten-System für Ihr Büro</p>
      </div>

      <!-- Content -->
      <div class="p-8">

        <!-- Step 1: Ollama Check -->
        <div v-if="currentStep === 1" class="space-y-6">
          <div class="text-center">
            <div class="inline-flex items-center justify-center w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900 mb-4">
              <span class="text-2xl font-bold text-blue-600 dark:text-blue-400">1</span>
            </div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Ollama-Verbindung prüfen</h2>
            <p class="text-gray-600 dark:text-gray-400">
              Fleet Navigator benötigt Ollama für die KI-Funktionen.
            </p>
          </div>

          <!-- Status Check -->
          <div class="bg-gray-50 dark:bg-gray-900 rounded-xl p-6">
            <div v-if="checkingOllama" class="flex items-center justify-center gap-3">
              <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
              <span class="text-gray-600 dark:text-gray-400">Prüfe Ollama-Verbindung...</span>
            </div>

            <div v-else-if="ollamaAvailable" class="text-center">
              <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-900 mb-4">
                <svg class="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              </div>
              <p class="text-lg font-semibold text-green-700 dark:text-green-400">Ollama ist verbunden!</p>
            </div>

            <div v-else class="text-center">
              <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-red-100 dark:bg-red-900 mb-4">
                <svg class="w-8 h-8 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </div>
              <p class="text-lg font-semibold text-red-700 dark:text-red-400 mb-4">Ollama nicht erreichbar</p>
              <div class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg p-4 text-left">
                <p class="text-sm text-red-800 dark:text-red-200 mb-2">Bitte starte Ollama:</p>
                <code class="block bg-gray-800 text-green-400 p-2 rounded text-sm font-mono">ollama serve</code>
              </div>
              <button
                @click="checkOllamaStatus"
                class="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
              >
                Erneut prüfen
              </button>
            </div>
          </div>
        </div>

        <!-- Step 2: Model Download -->
        <div v-if="currentStep === 2" class="space-y-6">
          <div class="text-center">
            <div class="inline-flex items-center justify-center w-12 h-12 rounded-full bg-blue-100 dark:bg-blue-900 mb-4">
              <span class="text-2xl font-bold text-blue-600 dark:text-blue-400">2</span>
            </div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-2">KI-Modell herunterladen</h2>
            <p class="text-gray-600 dark:text-gray-400">
              Wähle ein Modell für deinen Start. Du kannst später weitere hinzufügen.
            </p>
          </div>

          <!-- Model Selection -->
          <div class="space-y-3">
            <div
              v-for="model in recommendedModels"
              :key="model.name"
              @click="selectedModel = model.name"
              :class="[
                'p-4 rounded-xl border-2 cursor-pointer transition-all',
                selectedModel === model.name
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
                  : 'border-gray-200 dark:border-gray-700 hover:border-blue-300'
              ]"
            >
              <div class="flex items-start gap-4">
                <div :class="[
                  'w-5 h-5 rounded-full border-2 flex items-center justify-center mt-1',
                  selectedModel === model.name ? 'border-blue-500 bg-blue-500' : 'border-gray-300'
                ]">
                  <svg v-if="selectedModel === model.name" class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                  </svg>
                </div>
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <h3 class="font-semibold text-gray-900 dark:text-white">{{ model.displayName }}</h3>
                    <span v-if="model.recommended" class="px-2 py-0.5 bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-400 text-xs rounded-full">
                      Empfohlen
                    </span>
                  </div>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">{{ model.description }}</p>
                  <p class="text-xs text-gray-500 dark:text-gray-500 mt-1">{{ model.size }}</p>
                </div>
              </div>
            </div>
          </div>

          <!-- Download Button -->
          <button
            v-if="!downloading"
            @click="downloadModel"
            :disabled="!selectedModel"
            :class="[
              'w-full py-4 rounded-xl font-semibold text-lg transition-all',
              selectedModel
                ? 'bg-blue-600 hover:bg-blue-700 text-white'
                : 'bg-gray-300 text-gray-500 cursor-not-allowed'
            ]"
          >
            Modell herunterladen
          </button>

          <!-- Download Progress -->
          <div v-if="downloading" class="space-y-4">
            <div class="text-center">
              <p class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
                {{ downloadProgress }}%
              </p>
              <p class="text-sm text-gray-600 dark:text-gray-400">{{ downloadStatus }}</p>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden">
              <div
                class="bg-gradient-to-r from-blue-500 to-blue-600 h-4 rounded-full transition-all duration-300"
                :style="{ width: downloadProgress + '%' }"
              ></div>
            </div>
          </div>
        </div>

        <!-- Step 3: Complete -->
        <div v-if="currentStep === 3" class="space-y-6 text-center">
          <div class="inline-flex items-center justify-center w-20 h-20 rounded-full bg-green-100 dark:bg-green-900 mb-4">
            <svg class="w-10 h-10 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Alles bereit!</h2>
          <p class="text-gray-600 dark:text-gray-400">
            Fleet Navigator ist jetzt einsatzbereit. Viel Erfolg!
          </p>
          <button
            @click="$emit('complete')"
            class="w-full py-4 bg-green-600 hover:bg-green-700 text-white rounded-xl font-semibold text-lg transition-all"
          >
            Los geht's!
          </button>
        </div>
      </div>

      <!-- Footer mit Steps -->
      <div class="px-8 py-4 bg-gray-50 dark:bg-gray-900 border-t border-gray-200 dark:border-gray-700">
        <div class="flex justify-center gap-2">
          <div
            v-for="step in 3"
            :key="step"
            :class="[
              'w-3 h-3 rounded-full transition-colors',
              currentStep >= step ? 'bg-blue-600' : 'bg-gray-300 dark:bg-gray-600'
            ]"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '../services/api'

const emit = defineEmits(['complete'])

const currentStep = ref(1)
const checkingOllama = ref(true)
const ollamaAvailable = ref(false)
const selectedModel = ref('qwen2.5:3b')
const downloading = ref(false)
const downloadProgress = ref(0)
const downloadStatus = ref('')

const recommendedModels = [
  {
    name: 'qwen2.5:3b',
    displayName: 'Qwen 2.5 (3B)',
    description: 'Exzellentes Deutsch, schnelle Antworten, ideal für Büroaufgaben',
    size: '~2.1 GB',
    recommended: true
  },
  {
    name: 'llama3.2:3b',
    displayName: 'Llama 3.2 (3B)',
    description: 'Metas neuestes Modell, vielseitig einsetzbar',
    size: '~2.0 GB',
    recommended: false
  },
  {
    name: 'qwen2.5:7b',
    displayName: 'Qwen 2.5 (7B)',
    description: 'Leistungsstärker, benötigt mehr Speicher',
    size: '~4.7 GB',
    recommended: false
  }
]

onMounted(() => {
  checkOllamaStatus()
})

async function checkOllamaStatus() {
  checkingOllama.value = true
  try {
    const health = await api.getSystemHealth()
    ollamaAvailable.value = health.ollama_available

    // Wenn Ollama verfügbar und Modelle vorhanden, direkt fertig
    if (health.hasModels) {
      currentStep.value = 3
    } else if (ollamaAvailable.value) {
      currentStep.value = 2
    }
  } catch (e) {
    console.error('Health check failed:', e)
    ollamaAvailable.value = false
  }
  checkingOllama.value = false
}

async function downloadModel() {
  if (!selectedModel.value) return

  downloading.value = true
  downloadProgress.value = 0
  downloadStatus.value = 'Starte Download...'

  try {
    const response = await fetch('/api/models/pull', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: selectedModel.value })
    })

    const reader = response.body.getReader()
    const decoder = new TextDecoder()

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      const text = decoder.decode(value)
      const lines = text.split('\n').filter(line => line.startsWith('data: '))

      for (const line of lines) {
        try {
          const data = JSON.parse(line.slice(6))

          if (data.status) {
            downloadStatus.value = data.status
          }

          if (data.completed && data.total) {
            downloadProgress.value = Math.round((data.completed / data.total) * 100)
          }

          if (data.status === 'success' || data.status === 'done') {
            downloadProgress.value = 100
            downloadStatus.value = 'Download abgeschlossen!'
            setTimeout(() => {
              currentStep.value = 3
            }, 1000)
          }

          if (data.error) {
            downloadStatus.value = 'Fehler: ' + data.error
            downloading.value = false
          }
        } catch (e) {
          // Ignore parse errors
        }
      }
    }
  } catch (error) {
    console.error('Download failed:', error)
    downloadStatus.value = 'Download fehlgeschlagen: ' + error.message
    downloading.value = false
  }
}
</script>
