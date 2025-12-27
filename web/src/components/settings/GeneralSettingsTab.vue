<template>
  <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
      <GlobeAltIcon class="w-5 h-5 text-blue-500" />
      {{ $t('settings.general.title') }}
    </h3>

    <!-- Language -->
    <div class="mb-4">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
        <LanguageIcon class="w-4 h-4" />
        {{ $t('settings.general.language') }}
      </label>
      <select
        v-model="localSettings.language"
        @change="onLanguageChange"
        class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-fleet-orange-500 focus:border-transparent"
      >
        <option value="de">🇩🇪 Deutsch</option>
        <option value="en">🇬🇧 English</option>
        <option value="tr">🇹🇷 Türkçe</option>
      </select>

      <!-- Voice Download Dialog -->
      <div v-if="showVoiceDownloadDialog" class="mt-3 p-4 bg-amber-50 dark:bg-amber-900/30 rounded-xl border border-amber-200 dark:border-amber-700">
        <div class="flex items-start gap-3">
          <SpeakerWaveIcon class="w-6 h-6 text-amber-500 flex-shrink-0 mt-0.5" />
          <div class="flex-1">
            <h4 class="font-medium text-amber-800 dark:text-amber-200">
              {{ $t('settings.voice.voicesNeeded') || 'Stimmen für diese Sprache benötigt' }}
            </h4>
            <p class="text-sm text-amber-700 dark:text-amber-300 mt-1">
              {{ voiceDownloadInfo.availableVoices?.map(v => v.name).join(', ') }}
            </p>
            <div class="flex gap-2 mt-3">
              <button
                @click="$emit('download-voices')"
                :disabled="isDownloadingVoices"
                class="px-4 py-2 bg-amber-500 hover:bg-amber-600 text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
              >
                <span v-if="isDownloadingVoices">{{ $t('common.downloading') || 'Wird heruntergeladen...' }}</span>
                <span v-else>{{ $t('settings.voice.downloadVoices') || 'Stimmen herunterladen' }}</span>
              </button>
              <button
                @click="showVoiceDownloadDialog = false"
                class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg text-sm font-medium transition-colors"
              >
                {{ $t('common.later') || 'Später' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Theme -->
    <div class="mb-4">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
        <SunIcon class="w-4 h-4" />
        {{ $t('settings.general.theme') }}
      </label>
      <select
        v-model="localSettings.theme"
        class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-fleet-orange-500 focus:border-transparent"
      >
        <option value="light">☀️ {{ $t('common.light') }}</option>
        <option value="dark">🌙 {{ $t('common.dark') }}</option>
        <option value="auto">🔄 {{ $t('common.auto') }}</option>
      </select>
    </div>

    <!-- Modus-Wechsel-Nachrichten Toggle -->
    <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
      <div class="flex items-center justify-between">
        <div class="flex-1">
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
            <ArrowsRightLeftIcon class="w-4 h-4 text-purple-500" />
            {{ $t('settings.general.showModeSwitchMessages') }}
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {{ $t('settings.general.showModeSwitchMessagesDesc') }}
          </p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input type="checkbox" v-model="localSettings.showModeSwitchMessages" class="sr-only peer">
          <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-purple-300 dark:peer-focus:ring-purple-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-purple-500"></div>
        </label>
      </div>
    </div>

    <!-- Schriftgröße -->
    <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
      <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16m-7 6h7" />
        </svg>
        {{ $t('settings.general.fontSize') }}
      </label>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
        {{ $t('settings.general.fontSizeDesc') }}
      </p>
      <!-- Slider mit Wertanzeige -->
      <div class="space-y-3">
        <div class="flex items-center gap-4">
          <span class="text-xs text-gray-500 dark:text-gray-400 w-8">50%</span>
          <input
            type="range"
            min="50"
            max="150"
            step="5"
            :value="localSettings.fontSize || 100"
            @input="setFontSize(Number($event.target.value))"
            class="font-size-slider flex-1 h-3 rounded-lg appearance-none cursor-pointer"
          />
          <span class="text-xs text-gray-500 dark:text-gray-400 w-10">150%</span>
        </div>
        <!-- Aktuelle Größe und Reset-Button -->
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ localSettings.fontSize || 100 }}%</span>
            <span class="text-xs text-gray-500" :style="{ fontSize: (localSettings.fontSize || 100) * 0.14 + 'px' }">
              {{ $t('settings.general.sampleText') }}
            </span>
          </div>
          <button
            type="button"
            @click="setFontSize(100)"
            class="px-3 py-1 text-xs rounded-lg bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 transition-colors"
          >
            {{ $t('common.reset') }}
          </button>
        </div>
      </div>
    </div>

    <!-- CPU-Only Mode Toggle -->
    <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
      <div class="flex items-center justify-between">
        <div class="flex-1">
          <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
            <CpuChipIcon class="w-4 h-4 text-orange-500" />
            {{ $t('settings.hardware.cpuOnly') }}
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {{ $t('settings.hardware.cpuOnlyDesc') }}
          </p>
        </div>
        <ToggleSwitch v-model="localSettings.cpuOnly" color="orange" />
      </div>
      <!-- Info Box -->
      <div v-if="localSettings.cpuOnly" class="mt-2 p-2 rounded-lg bg-orange-50 dark:bg-orange-900/30 border border-orange-200 dark:border-orange-800">
        <p class="text-xs text-orange-800 dark:text-orange-200 flex items-center gap-2">
          <ExclamationTriangleIcon class="w-4 h-4 flex-shrink-0" />
          <span>{{ $t('settings.hardware.cpuOnlyActive') }}</span>
        </p>
      </div>

      <!-- VRAM Settings -->
      <div class="mt-4">
        <VRAMSettings />
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, watch } from 'vue'
import {
  GlobeAltIcon,
  LanguageIcon,
  SunIcon,
  CpuChipIcon,
  ArrowsRightLeftIcon,
  ExclamationTriangleIcon,
  SpeakerWaveIcon
} from '@heroicons/vue/24/outline'
import ToggleSwitch from '../ToggleSwitch.vue'
import VRAMSettings from './VRAMSettings.vue'

const props = defineProps({
  settings: {
    type: Object,
    required: true
  },
  voiceDownloadInfo: {
    type: Object,
    default: () => ({ availableVoices: [], installedVoices: [] })
  },
  isDownloadingVoices: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:settings', 'language-change', 'download-voices'])

// Local copy for v-model binding
const localSettings = ref({ ...props.settings })
const showVoiceDownloadDialog = ref(false)

// Sync with parent
watch(() => props.settings, (newVal) => {
  localSettings.value = { ...newVal }
}, { deep: true })

watch(localSettings, (newVal) => {
  emit('update:settings', newVal)
}, { deep: true })

// Show voice dialog when voiceDownloadInfo has available voices
watch(() => props.voiceDownloadInfo, (newVal) => {
  if (newVal?.availableVoices?.length > 0) {
    showVoiceDownloadDialog.value = true
  }
}, { deep: true })

function onLanguageChange() {
  emit('language-change', localSettings.value.language)
}

function setFontSize(size) {
  localSettings.value.fontSize = size
}
</script>

<style scoped>
.font-size-slider {
  background: linear-gradient(to right, #3b82f6 0%, #60a5fa 50%, #93c5fd 100%);
}

.font-size-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 20px;
  height: 20px;
  background: white;
  border: 2px solid #3b82f6;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.font-size-slider::-moz-range-thumb {
  width: 20px;
  height: 20px;
  background: white;
  border: 2px solid #3b82f6;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}
</style>
