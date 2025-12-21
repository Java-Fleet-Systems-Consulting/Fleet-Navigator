<template>
  <div class="voice-store">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
        <span>Piper Voice Store</span>
        <span class="text-sm font-normal text-gray-500">({{ filteredVoices.length }} Stimmen)</span>
      </h3>
      <button
        @click="loadVoices"
        :disabled="loading"
        class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
        title="Neu laden"
      >
        <span :class="{ 'animate-spin': loading }">🔄</span>
      </button>
    </div>

    <!-- Sprach-Filter -->
    <div class="mb-4 flex flex-wrap gap-2">
      <button
        v-for="lang in availableLanguages"
        :key="lang.code"
        @click="selectedLanguage = lang.code"
        :class="[
          'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
          selectedLanguage === lang.code
            ? 'bg-indigo-500 text-white'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
        ]"
      >
        {{ lang.flag }} {{ lang.name }} ({{ lang.count }})
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-500"></div>
      <span class="ml-3 text-gray-600 dark:text-gray-400">Lade Stimmen...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="p-4 bg-red-50 dark:bg-red-900/20 rounded-lg text-red-600 dark:text-red-400">
      {{ error }}
    </div>

    <!-- Voice Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 max-h-96 overflow-y-auto">
      <div
        v-for="voice in filteredVoices"
        :key="voice.key"
        :class="[
          'p-3 rounded-lg border transition-all',
          voice.installed
            ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/20'
            : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-indigo-300 dark:hover:border-indigo-600'
        ]"
      >
        <!-- Voice Header -->
        <div class="flex items-start justify-between mb-2">
          <div>
            <h4 class="font-medium text-gray-900 dark:text-white text-sm">
              {{ voice.name || voice.key }}
            </h4>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ voice.key }}
            </p>
          </div>
          <span
            v-if="voice.installed"
            class="px-2 py-0.5 text-xs font-medium bg-green-100 dark:bg-green-800 text-green-700 dark:text-green-300 rounded"
          >
            Installiert
          </span>
        </div>

        <!-- Voice Info -->
        <div class="flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400 mb-2">
          <span>{{ voice.quality }}</span>
          <span>•</span>
          <span>{{ formatSize(voice.files) }}</span>
          <span v-if="voice.num_speakers > 1">• {{ voice.num_speakers }} Sprecher</span>
        </div>

        <!-- Actions -->
        <div class="flex gap-2">
          <button
            v-if="!voice.installed"
            @click="downloadVoice(voice)"
            :disabled="downloading === voice.key"
            class="flex-1 px-3 py-1.5 text-xs font-medium bg-indigo-500 hover:bg-indigo-600 text-white rounded-lg disabled:opacity-50 flex items-center justify-center gap-1"
          >
            <span v-if="downloading === voice.key" class="animate-spin">⏳</span>
            <span v-else>📥</span>
            {{ downloading === voice.key ? `${downloadProgress}%` : 'Installieren' }}
          </button>
          <button
            v-if="voice.installed"
            @click="testVoice(voice)"
            :disabled="testing === voice.key"
            class="flex-1 px-3 py-1.5 text-xs font-medium bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg disabled:opacity-50 flex items-center justify-center gap-1"
          >
            <span v-if="testing === voice.key" class="animate-pulse">🔊</span>
            <span v-else>🔊</span>
            Testen
          </button>
        </div>
      </div>
    </div>

    <!-- Download Progress -->
    <div v-if="downloading" class="mt-4 p-3 bg-indigo-50 dark:bg-indigo-900/20 rounded-lg">
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm text-indigo-700 dark:text-indigo-300">
          Lade {{ downloading }}...
        </span>
        <span class="text-sm font-medium text-indigo-700 dark:text-indigo-300">
          {{ downloadProgress }}%
        </span>
      </div>
      <div class="w-full bg-indigo-200 dark:bg-indigo-800 rounded-full h-2">
        <div
          class="bg-indigo-500 h-2 rounded-full transition-all"
          :style="{ width: `${downloadProgress}%` }"
        ></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'

// State
const voices = ref({})
const loading = ref(false)
const error = ref('')
const selectedLanguage = ref('de')
const downloading = ref(null)
const downloadProgress = ref(0)
const testing = ref(null)

// Sprach-Mapping
const languageNames = {
  'ar': { name: 'Arabisch', flag: '🇸🇦' },
  'ca': { name: 'Katalanisch', flag: '🏴' },
  'cs': { name: 'Tschechisch', flag: '🇨🇿' },
  'cy': { name: 'Walisisch', flag: '🏴󠁧󠁢󠁷󠁬󠁳󠁿' },
  'da': { name: 'Dänisch', flag: '🇩🇰' },
  'de': { name: 'Deutsch', flag: '🇩🇪' },
  'el': { name: 'Griechisch', flag: '🇬🇷' },
  'en': { name: 'Englisch', flag: '🇬🇧' },
  'es': { name: 'Spanisch', flag: '🇪🇸' },
  'fa': { name: 'Persisch', flag: '🇮🇷' },
  'fi': { name: 'Finnisch', flag: '🇫🇮' },
  'fr': { name: 'Französisch', flag: '🇫🇷' },
  'hi': { name: 'Hindi', flag: '🇮🇳' },
  'hu': { name: 'Ungarisch', flag: '🇭🇺' },
  'is': { name: 'Isländisch', flag: '🇮🇸' },
  'it': { name: 'Italienisch', flag: '🇮🇹' },
  'ka': { name: 'Georgisch', flag: '🇬🇪' },
  'kk': { name: 'Kasachisch', flag: '🇰🇿' },
  'lb': { name: 'Luxemburgisch', flag: '🇱🇺' },
  'lv': { name: 'Lettisch', flag: '🇱🇻' },
  'ml': { name: 'Malayalam', flag: '🇮🇳' },
  'ne': { name: 'Nepali', flag: '🇳🇵' },
  'nl': { name: 'Niederländisch', flag: '🇳🇱' },
  'no': { name: 'Norwegisch', flag: '🇳🇴' },
  'pl': { name: 'Polnisch', flag: '🇵🇱' },
  'pt': { name: 'Portugiesisch', flag: '🇵🇹' },
  'ro': { name: 'Rumänisch', flag: '🇷🇴' },
  'ru': { name: 'Russisch', flag: '🇷🇺' },
  'sk': { name: 'Slowakisch', flag: '🇸🇰' },
  'sl': { name: 'Slowenisch', flag: '🇸🇮' },
  'sr': { name: 'Serbisch', flag: '🇷🇸' },
  'sv': { name: 'Schwedisch', flag: '🇸🇪' },
  'sw': { name: 'Suaheli', flag: '🇰🇪' },
  'tr': { name: 'Türkisch', flag: '🇹🇷' },
  'uk': { name: 'Ukrainisch', flag: '🇺🇦' },
  'vi': { name: 'Vietnamesisch', flag: '🇻🇳' },
  'zh': { name: 'Chinesisch', flag: '🇨🇳' },
}

// Computed
const availableLanguages = computed(() => {
  const langCounts = {}
  Object.keys(voices.value).forEach(key => {
    const lang = key.substring(0, 2)
    langCounts[lang] = (langCounts[lang] || 0) + 1
  })

  return Object.entries(langCounts)
    .map(([code, count]) => ({
      code,
      count,
      name: languageNames[code]?.name || code.toUpperCase(),
      flag: languageNames[code]?.flag || '🌐'
    }))
    .sort((a, b) => {
      // Deutsch und Englisch zuerst
      if (a.code === 'de') return -1
      if (b.code === 'de') return 1
      if (a.code === 'en') return -1
      if (b.code === 'en') return 1
      return a.name.localeCompare(b.name)
    })
})

const filteredVoices = computed(() => {
  return Object.entries(voices.value)
    .filter(([key]) => key.startsWith(selectedLanguage.value + '_'))
    .map(([key, data]) => ({
      key,
      ...data,
      name: extractName(key),
      quality: extractQuality(key)
    }))
    .sort((a, b) => {
      // Installierte zuerst
      if (a.installed && !b.installed) return -1
      if (!a.installed && b.installed) return 1
      return a.key.localeCompare(b.key)
    })
})

// Helper Functions
function extractName(key) {
  // de_DE-thorsten-medium -> Thorsten
  const parts = key.split('-')
  if (parts.length >= 2) {
    const name = parts[1].replace(/_/g, ' ')
    return name.charAt(0).toUpperCase() + name.slice(1)
  }
  return key
}

function extractQuality(key) {
  // de_DE-thorsten-medium -> Medium
  const parts = key.split('-')
  if (parts.length >= 3) {
    const quality = parts[parts.length - 1]
    return quality.charAt(0).toUpperCase() + quality.slice(1)
  }
  return 'Standard'
}

function formatSize(files) {
  if (!files) return '?'
  // Sum all file sizes
  let totalBytes = 0
  Object.values(files).forEach(file => {
    if (file.size_bytes) {
      totalBytes += file.size_bytes
    }
  })
  if (totalBytes === 0) return '?'
  const mb = totalBytes / (1024 * 1024)
  return `${mb.toFixed(0)} MB`
}

// API Functions
async function loadVoices() {
  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/api/voice-store/voices')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }
    voices.value = await response.json()
  } catch (e) {
    error.value = `Fehler beim Laden: ${e.message}`
    console.error('Voice store error:', e)
  } finally {
    loading.value = false
  }
}

async function downloadVoice(voice) {
  downloading.value = voice.key
  downloadProgress.value = 0

  try {
    const eventSource = new EventSource(`/api/voice/download-model?component=piper&model=${voice.key}`)

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        console.log('Download progress:', data)

        if (data.percent) {
          downloadProgress.value = Math.round(data.percent)
        }

        if (data.status === 'done' || data.status === 'complete') {
          eventSource.close()
          downloading.value = null
          downloadProgress.value = 0
          // Voices neu laden um Status zu aktualisieren
          loadVoices()
        } else if (data.status === 'error') {
          eventSource.close()
          downloading.value = null
          error.value = data.error || 'Download fehlgeschlagen'
        }
      } catch (e) {
        console.error('Error parsing SSE:', e)
      }
    }

    eventSource.onerror = (err) => {
      console.error('SSE error:', err)
      eventSource.close()
      downloading.value = null
      error.value = 'Download-Verbindung unterbrochen'
    }
  } catch (e) {
    downloading.value = null
    error.value = `Download-Fehler: ${e.message}`
  }
}

async function testVoice(voice) {
  testing.value = voice.key

  try {
    const testText = selectedLanguage.value === 'de'
      ? 'Hallo! Dies ist ein Test der Sprachausgabe.'
      : selectedLanguage.value === 'en'
      ? 'Hello! This is a speech output test.'
      : 'Hello! Test.'

    const response = await fetch('/api/voice/tts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: testText,
        voice: voice.key
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`)
    }

    const audioBlob = await response.blob()
    const audioUrl = URL.createObjectURL(audioBlob)
    const audio = new Audio(audioUrl)
    audio.onended = () => {
      URL.revokeObjectURL(audioUrl)
      testing.value = null
    }
    audio.onerror = () => {
      testing.value = null
    }
    await audio.play()
  } catch (e) {
    console.error('Test error:', e)
    testing.value = null
  }
}

// Lifecycle
onMounted(() => {
  loadVoices()
})

// Expose for parent
defineExpose({
  loadVoices
})
</script>

<style scoped>
.voice-store {
  @apply w-full;
}
</style>
