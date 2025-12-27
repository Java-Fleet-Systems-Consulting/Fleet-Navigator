<template>
  <div>
    <!-- Main Voice Section -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <MicrophoneIcon class="w-5 h-5 text-purple-500" />
        {{ $t('settings.voice.title') }}
      </h3>

      <!-- TTS Global Toggle -->
      <div class="mb-4 p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <SpeakerWaveIcon class="w-5 h-5 text-indigo-500" />
            <div>
              <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.ttsEnabled') }}</span>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('settings.voice.ttsEnabledDesc') }}</p>
            </div>
          </div>
          <button
            @click="$emit('toggle-tts')"
            :class="[
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              ttsEnabled ? 'bg-indigo-500' : 'bg-gray-300 dark:bg-gray-600'
            ]"
          >
            <span
              :class="[
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                ttsEnabled ? 'translate-x-6' : 'translate-x-1'
              ]"
            />
          </button>
        </div>
      </div>

      <!-- Download Progress -->
      <div v-if="voiceDownloading" class="mb-4 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
        <div class="flex items-center gap-3 mb-2">
          <div class="animate-spin rounded-full h-5 w-5 border-2 border-blue-500 border-t-transparent"></div>
          <span class="font-medium text-blue-700 dark:text-blue-300">{{ voiceDownloadStatus }}</span>
        </div>
        <div v-if="voiceDownloadProgress > 0" class="w-full bg-blue-200 dark:bg-blue-800 rounded-full h-2">
          <div class="bg-blue-500 h-2 rounded-full transition-all" :style="{ width: voiceDownloadProgress + '%' }"></div>
        </div>
        <p v-if="voiceDownloadSpeed" class="text-xs text-blue-600 dark:text-blue-400 mt-1">{{ voiceDownloadSpeed }}</p>
      </div>

      <!-- Whisper STT Section -->
      <div class="mb-6 p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <div class="flex items-center gap-2 mb-3">
          <MicrophoneIcon class="w-5 h-5 text-purple-500" />
          <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.whisperStt') }}</span>
          <span v-if="voiceModels.whisperBinary" class="ml-auto text-xs px-2 py-0.5 rounded-full bg-green-200 dark:bg-green-800 text-green-800 dark:text-green-200">{{ $t('settings.voice.ready') }}</span>
          <span v-else class="ml-auto text-xs px-2 py-0.5 rounded-full bg-red-200 dark:bg-red-800 text-red-800 dark:text-red-200">{{ $t('settings.voice.notInstalled') }}</span>
        </div>

        <!-- Install Whisper Button -->
        <div v-if="!voiceModels.whisperBinary" class="mb-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
          <p class="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
            <strong>{{ $t('settings.voice.whisperNotInstalled') }}</strong> {{ $t('settings.voice.whisperInstallHint') }}
          </p>
          <button
            @click="$emit('download-whisper')"
            :disabled="voiceDownloading"
            class="w-full px-4 py-2 rounded-lg bg-purple-500 hover:bg-purple-600 text-white font-medium disabled:opacity-50 flex items-center justify-center gap-2"
          >
            <ArrowDownTrayIcon v-if="voiceDownloadComponent !== 'whisper'" class="w-5 h-5" />
            <div v-else class="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
            {{ voiceDownloadComponent === 'whisper' ? $t('settings.voice.installing') : $t('settings.voice.installWhisper') }}
          </button>
        </div>

        <!-- Whisper Models Grid -->
        <template v-if="voiceModels.whisperBinary">
          <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ $t('settings.voice.selectModelHint') }}</p>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            <div
              v-for="model in voiceModels.whisper"
              :key="model.id"
              class="p-3 rounded-lg border-2 cursor-pointer transition-all"
              :class="[
                voiceModels.currentWhisper === model.id
                  ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                  : model.installed
                    ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/10 hover:border-green-400'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
              ]"
              @click="model.installed && $emit('select-whisper-model', model.id)"
            >
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-gray-900 dark:text-white">{{ model.name }}</span>
                <div class="flex items-center gap-1">
                  <span v-if="voiceModels.currentWhisper === model.id" class="text-xs px-1.5 py-0.5 rounded bg-purple-500 text-white">{{ $t('settings.voice.active') }}</span>
                  <span v-else-if="model.installed" class="text-xs px-1.5 py-0.5 rounded bg-green-500 text-white">{{ $t('settings.voice.installed') }}</span>
                </div>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">{{ model.description }}</p>
              <div class="flex items-center justify-between">
                <span class="text-xs text-gray-400">{{ model.sizeMB }} MB</span>
                <button
                  v-if="!model.installed"
                  @click.stop="$emit('download-model', 'whisper', model.id)"
                  :disabled="voiceDownloading"
                  class="text-xs px-2 py-1 rounded bg-purple-500 hover:bg-purple-600 text-white disabled:opacity-50"
                >
                  {{ $t('settings.voice.download') }}
                </button>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- Piper TTS Section -->
      <div class="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
        <div class="flex items-center gap-2 mb-3">
          <SpeakerWaveIcon class="w-5 h-5 text-indigo-500" />
          <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.piperTts') }}</span>
          <span v-if="voiceModels.piperBinary" class="ml-auto text-xs px-2 py-0.5 rounded-full bg-green-200 dark:bg-green-800 text-green-800 dark:text-green-200">{{ $t('settings.voice.ready') }}</span>
          <span v-else class="ml-auto text-xs px-2 py-0.5 rounded-full bg-red-200 dark:bg-red-800 text-red-800 dark:text-red-200">{{ $t('settings.voice.notInstalled') }}</span>
        </div>

        <!-- Install Piper Button -->
        <div v-if="!voiceModels.piperBinary" class="mb-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
          <p class="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
            <strong>{{ $t('settings.voice.piperNotInstalled') }}</strong> {{ $t('settings.voice.piperInstallHint') }}
          </p>
          <button
            @click="$emit('download-piper')"
            :disabled="voiceDownloading"
            class="w-full px-4 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white font-medium disabled:opacity-50 flex items-center justify-center gap-2"
          >
            <ArrowDownTrayIcon v-if="voiceDownloadComponent !== 'piper'" class="w-5 h-5" />
            <div v-else class="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
            {{ voiceDownloadComponent === 'piper' ? $t('settings.voice.downloading') : $t('settings.voice.installPiper') }}
          </button>
        </div>

        <!-- Piper Language Filter and Voices Grid -->
        <template v-if="voiceModels.piperBinary">
          <div class="flex gap-2 mb-3">
            <button
              v-for="lang in ['de', 'en', 'all']"
              :key="lang"
              @click="piperLanguageFilter = lang"
              class="px-3 py-1 rounded-full text-sm transition-all"
              :class="piperLanguageFilter === lang ? 'bg-indigo-500 text-white' : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-200'"
            >
              {{ lang === 'de' ? $t('settings.voice.german') : lang === 'en' ? $t('settings.voice.english') : $t('settings.voice.all') }}
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-80 overflow-y-auto pr-1">
            <div
              v-for="voice in filteredPiperVoices"
              :key="voice.id"
              class="p-3 rounded-lg border-2 cursor-pointer transition-all"
              :class="[
                voiceModels.currentPiper === voice.id
                  ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                  : voice.installed
                    ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/10 hover:border-green-400'
                    : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
              ]"
              @click="voice.installed && $emit('select-piper-voice', voice.id)"
            >
              <div class="flex items-center justify-between mb-1">
                <span class="font-medium text-gray-900 dark:text-white">{{ voice.name }}</span>
                <div class="flex items-center gap-1">
                  <span v-if="voiceModels.currentPiper === voice.id" class="text-xs px-1.5 py-0.5 rounded bg-indigo-500 text-white">{{ $t('settings.voice.active') }}</span>
                  <span v-else-if="voice.installed" class="text-xs px-1.5 py-0.5 rounded bg-green-500 text-white">{{ $t('settings.voice.installed') }}</span>
                </div>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">{{ voice.description }}</p>
              <div class="flex items-center justify-between mt-2">
                <div class="flex items-center gap-2 text-xs text-gray-400">
                  <span>{{ voice.language }}</span>
                  <span class="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700">{{ voice.quality }}</span>
                  <span>{{ voice.sizeMB }} MB</span>
                </div>
                <button
                  v-if="!voice.installed"
                  @click.stop="$emit('download-model', 'piper', voice.id)"
                  :disabled="voiceDownloading"
                  class="text-xs px-2 py-1 rounded bg-indigo-500 hover:bg-indigo-600 text-white disabled:opacity-50"
                >
                  {{ $t('settings.voice.download') }}
                </button>
              </div>
            </div>
          </div>
        </template>
      </div>

      <!-- Voice Store -->
      <div class="mt-6 p-4 rounded-lg border border-indigo-200 dark:border-indigo-700 bg-indigo-50/50 dark:bg-indigo-900/10">
        <VoiceStore ref="voiceStoreRef" />
      </div>

      <!-- Info -->
      <div class="mt-4 p-3 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm text-gray-600 dark:text-gray-400">
        <p><strong>{{ $t('common.note') }}:</strong> {{ $t('settings.voice.voiceStorageInfo') }}</p>
        <p class="mt-1">{{ $t('settings.voice.clickToTest') }}</p>
      </div>
    </section>

    <!-- Voice Assistant Section -->
    <section class="mt-6 bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-900/30 dark:to-emerald-800/30 p-5 rounded-xl border border-green-200/50 dark:border-green-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <span class="text-2xl">🎙️</span>
        Voice Assistant (Ewa)
      </h3>

      <!-- Voice Assistant Enable/Disable -->
      <div class="mb-4 p-4 rounded-xl border border-green-200 dark:border-green-700 bg-white dark:bg-gray-800">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <span class="text-xl">{{ voiceAssistantSettings.enabled ? '🟢' : '⚪' }}</span>
            <div>
              <span class="font-semibold text-gray-900 dark:text-white">Voice Assistant aktivieren</span>
              <p class="text-xs text-gray-500 dark:text-gray-400">Sprachsteuerung per Wake Word</p>
            </div>
          </div>
          <ToggleSwitch v-model="localVoiceAssistant.enabled" color="green" @update:modelValue="onVoiceAssistantChange" />
        </div>
      </div>

      <!-- Wake Word Settings (visible when enabled) -->
      <div v-if="localVoiceAssistant.enabled" class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">🗣️ Wake Word auswählen</label>
          <div class="grid grid-cols-3 gap-3">
            <button
              v-for="ww in ['hey_ewa', 'ewa', 'custom']"
              :key="ww"
              @click="localVoiceAssistant.wakeWord = ww; onVoiceAssistantChange()"
              :class="[
                'p-3 rounded-lg border-2 transition-all text-center',
                localVoiceAssistant.wakeWord === ww
                  ? 'border-green-500 bg-green-500/10'
                  : 'border-gray-300 dark:border-gray-600 hover:border-green-400'
              ]"
            >
              <span class="block font-medium text-gray-900 dark:text-white">
                {{ ww === 'hey_ewa' ? '"Hey Ewa"' : ww === 'ewa' ? '"Ewa"' : 'Eigenes' }}
              </span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ ww === 'hey_ewa' ? 'Standard' : ww === 'ewa' ? 'Kurz' : 'Benutzerdefiniert' }}
              </span>
            </button>
          </div>
        </div>

        <!-- Custom Wake Word -->
        <div v-if="localVoiceAssistant.wakeWord === 'custom'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Eigenes Wake Word</label>
          <input
            v-model="localVoiceAssistant.customWakeWord"
            type="text"
            placeholder="z.B. 'Computer', 'Jarvis', ..."
            @change="onVoiceAssistantChange"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-green-500"
          >
        </div>

        <!-- Auto-Stop -->
        <div class="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white/50 dark:bg-gray-800/50">
          <div class="flex items-center justify-between">
            <div>
              <label class="text-sm font-medium text-gray-700 dark:text-gray-300">⏹️ Auto-Stopp nach Antwort</label>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">Beendet Lauschen nach einer vollständigen Antwort</p>
            </div>
            <ToggleSwitch v-model="localVoiceAssistant.autoStop" color="green" @update:modelValue="onVoiceAssistantChange" />
          </div>
        </div>

        <!-- Info -->
        <div class="p-4 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 border border-green-200 dark:border-green-700 rounded-lg">
          <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">💡 So funktioniert's</h4>
          <ul class="space-y-1 text-xs text-gray-600 dark:text-gray-400">
            <li>1. Sage <strong>"{{ getWakeWordDisplay(localVoiceAssistant.wakeWord) }}"</strong> um Ewa zu aktivieren</li>
            <li>2. Stelle deine Frage oder gib einen Befehl</li>
            <li>3. Ewa antwortet per Sprache (TTS)</li>
            <li>4. Wiederhole oder sage "Stop" zum Beenden</li>
          </ul>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import {
  MicrophoneIcon,
  SpeakerWaveIcon,
  ArrowDownTrayIcon
} from '@heroicons/vue/24/outline'
import ToggleSwitch from '../ToggleSwitch.vue'
import VoiceStore from '../VoiceStore.vue'

const props = defineProps({
  ttsEnabled: {
    type: Boolean,
    default: true
  },
  voiceDownloading: {
    type: Boolean,
    default: false
  },
  voiceDownloadComponent: {
    type: String,
    default: ''
  },
  voiceDownloadStatus: {
    type: String,
    default: ''
  },
  voiceDownloadProgress: {
    type: Number,
    default: 0
  },
  voiceDownloadSpeed: {
    type: String,
    default: ''
  },
  voiceModels: {
    type: Object,
    default: () => ({
      whisper: [],
      piper: [],
      currentWhisper: '',
      currentPiper: '',
      whisperBinary: false,
      piperBinary: false
    })
  },
  voiceAssistantSettings: {
    type: Object,
    default: () => ({
      enabled: false,
      wakeWord: 'hey_ewa',
      customWakeWord: '',
      autoStop: true
    })
  }
})

const emit = defineEmits([
  'toggle-tts',
  'download-whisper',
  'download-piper',
  'download-model',
  'select-whisper-model',
  'select-piper-voice',
  'update:voiceAssistantSettings'
])

const voiceStoreRef = ref(null)
const piperLanguageFilter = ref('de')

// Local copy of voice assistant settings
const localVoiceAssistant = ref({ ...props.voiceAssistantSettings })

// Sync with parent
watch(() => props.voiceAssistantSettings, (newVal) => {
  localVoiceAssistant.value = { ...newVal }
}, { deep: true })

// Filtered Piper voices
const filteredPiperVoices = computed(() => {
  if (!props.voiceModels.piper) return []
  if (piperLanguageFilter.value === 'all') return props.voiceModels.piper
  return props.voiceModels.piper.filter(v =>
    v.language?.startsWith(piperLanguageFilter.value === 'de' ? 'de' : 'en')
  )
})

function onVoiceAssistantChange() {
  emit('update:voiceAssistantSettings', localVoiceAssistant.value)
}

function getWakeWordDisplay(wakeWord) {
  switch (wakeWord) {
    case 'hey_ewa': return 'Hey Ewa'
    case 'ewa': return 'Ewa'
    case 'custom': return localVoiceAssistant.value.customWakeWord || 'Eigenes Wake Word'
    default: return 'Hey Ewa'
  }
}
</script>
