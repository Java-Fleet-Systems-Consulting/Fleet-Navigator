<template>
  <div>
    <!-- Vision Settings -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <PhotoIcon class="w-5 h-5 text-indigo-500" />
        {{ $t('settings.agents.visionModel') }}
      </h3>

      <!-- Auto-select Vision Model -->
      <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <EyeIcon class="w-4 h-4" />
              {{ $t('settings.agents.autoVisionModel') }}
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {{ $t('settings.agents.autoVisionModelDesc') }}
            </p>
          </div>
          <ToggleSwitch v-model="localSettings.autoSelectVisionModel" color="indigo" />
        </div>
      </div>

      <!-- Preferred Vision Model -->
      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          {{ $t('settings.agents.preferredVisionModel') }}
        </label>
        <select
          v-model="localModelSettings.visionModel"
          @change="onModelSettingsChange"
          class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-indigo-500"
        >
          <option v-for="model in visionModels" :key="model" :value="model">
            {{ model }}
          </option>
        </select>
        <p v-if="visionModels.length > 0" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
          {{ visionModels.length }} {{ $t('settings.agents.visionModelsAvailable') }}
        </p>
        <p v-else class="mt-2 text-xs text-yellow-600 dark:text-yellow-400">
          ⚠️ {{ $t('settings.agents.noVisionModels') }}
        </p>
      </div>

      <!-- Vision Chaining -->
      <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <LinkIcon class="w-4 h-4" />
              {{ $t('settings.agents.visionChaining') }}
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {{ $t('settings.agents.visionChainingDesc') }}
            </p>
          </div>
          <ToggleSwitch v-model="localModelSettings.visionChainingEnabled" color="indigo" @update:modelValue="onModelSettingsChange" />
        </div>
      </div>

      <!-- Vision-Server Idle Timeout (NEU: On-Demand Vision) -->
      <div class="mb-4 p-4 rounded-xl bg-cyan-50/50 dark:bg-cyan-900/20 border border-cyan-200 dark:border-cyan-700">
        <div class="flex items-center justify-between mb-3">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <ClockIcon class="w-4 h-4 text-cyan-500" />
              {{ $t('settings.agents.visionServerTimeout') || 'Vision-Server Timeout' }}
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {{ $t('settings.agents.visionServerTimeoutDesc') || 'Der Vision-Server läuft parallel zum Chat-Server und stoppt nach Inaktivität automatisch.' }}
            </p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <input
            type="range"
            v-model.number="localVisionIdleTimeout"
            @change="onVisionTimeoutChange"
            min="60"
            max="900"
            step="60"
            class="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-lg appearance-none cursor-pointer accent-cyan-500"
          />
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300 w-20 text-right">
            {{ formatTimeout(localVisionIdleTimeout) }}
          </span>
        </div>
        <!-- Vision-Server Status -->
        <div class="mt-3 flex items-center gap-2 text-xs">
          <span
            class="inline-flex items-center gap-1 px-2 py-1 rounded-full"
            :class="visionServerStatus?.running ? 'bg-cyan-100 dark:bg-cyan-900/50 text-cyan-700 dark:text-cyan-300' : 'bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400'"
          >
            <span class="w-1.5 h-1.5 rounded-full" :class="visionServerStatus?.running ? 'bg-cyan-500' : 'bg-gray-400'"></span>
            {{ visionServerStatus?.running ? 'Vision-Server aktiv' : 'Vision-Server gestoppt (On-Demand)' }}
          </span>
          <span v-if="visionServerStatus?.timeUntilStop" class="text-gray-400">
            Auto-Stop in {{ visionServerStatus.timeUntilStop }}
          </span>
        </div>
      </div>

      <!-- Web Search: Think First -->
      <div class="mb-4 p-4 rounded-xl bg-blue-50/50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <LightBulbIcon class="w-4 h-4 text-blue-500" />
              {{ $t('settings.agents.thinkFirst') || '🧠 Websuche: Erst nachdenken' }}
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              {{ $t('settings.agents.thinkFirstDesc') || 'LLM antwortet erst selbst, bei Unsicherheit → automatische Websuche' }}
            </p>
          </div>
          <ToggleSwitch v-model="localThinkFirst" color="blue" @update:modelValue="onThinkFirstChange" />
        </div>
        <!-- Explanation -->
        <div v-if="localThinkFirst" class="mt-3 p-2 bg-green-50 dark:bg-green-900/30 rounded-lg">
          <p class="text-xs text-green-700 dark:text-green-300">
            ✅ <strong>Aktiv:</strong> Das LLM versucht erst selbst zu antworten. Nur bei Unsicherheit wird automatisch eine Websuche durchgeführt.
          </p>
        </div>
        <div v-else class="mt-3 p-2 bg-orange-50 dark:bg-orange-900/30 rounded-lg">
          <p class="text-xs text-orange-700 dark:text-orange-300">
            ⚡ <strong>Sofort-Modus:</strong> Bei aktivierter Experten-Websuche wird sofort gesucht (schneller, aber mehr API-Calls).
          </p>
        </div>
      </div>
    </section>

    <!-- OS Mate - File Search (RAG) -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm mt-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <FolderIcon class="w-5 h-5 text-amber-500" />
        {{ $t('settings.agents.osMateTitle') }}
      </h3>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        {{ $t('settings.agents.osMateDesc') }}
      </p>

      <!-- File Search Status -->
      <div v-if="fileSearchStatus" class="mb-4 p-3 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800">
        <div class="flex items-center gap-2 text-sm text-blue-700 dark:text-blue-300">
          <span v-if="fileSearchStatus.indexingInProgress" class="animate-spin">⏳</span>
          <span v-else>📚</span>
          <span>
            {{ fileSearchStatus.indexedFileCount }} {{ $t('settings.agents.filesIndexed') }}
            <span v-if="fileSearchStatus.locateAvailable" class="text-xs text-green-600 dark:text-green-400 ml-2">({{ $t('settings.agents.locateAvailable') }})</span>
          </span>
        </div>
      </div>

      <!-- Search Folders List -->
      <div class="space-y-3 mb-4">
        <div v-for="folder in fileSearchFolders" :key="folder.folderId"
             class="p-3 rounded-lg bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <div class="flex items-center gap-3 flex-1">
            <FolderOpenIcon class="w-5 h-5 text-amber-500" />
            <div class="flex-1 min-w-0">
              <div class="font-medium text-gray-900 dark:text-white truncate">{{ folder.name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400 truncate">{{ folder.folderPath }}</div>
              <div class="text-xs text-gray-400 dark:text-gray-500">
                {{ folder.fileCount || 0 }} {{ $t('settings.agents.files') }}
                <span v-if="folder.lastIndexed" class="ml-2">• {{ $t('settings.agents.indexed') }}: {{ formatDateAbsolute(folder.lastIndexed) }}</span>
              </div>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button @click="$emit('reindex-folder', folder.folderId)"
                    class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-blue-600 dark:text-blue-400"
                    :title="$t('settings.agents.reindex')">
              <ArrowPathIcon class="w-4 h-4" />
            </button>
            <button @click="$emit('remove-folder', folder.folderId)"
                    class="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-600 dark:text-red-400"
                    :title="$t('common.remove')">
              <TrashIcon class="w-4 h-4" />
            </button>
          </div>
        </div>

        <div v-if="fileSearchFolders.length === 0"
             class="p-4 rounded-lg bg-gray-100/50 dark:bg-gray-800/30 border border-dashed border-gray-300 dark:border-gray-600 text-center">
          <FolderIcon class="w-8 h-8 text-gray-400 mx-auto mb-2" />
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('settings.agents.noFoldersConfigured') }}</p>
        </div>
      </div>

      <!-- Add Folder Form -->
      <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ $t('settings.agents.addFolder') }}</label>
        <div class="flex gap-2">
          <input type="text" v-model="newFolderPath"
                 :placeholder="$t('settings.agents.folderPathPlaceholder')"
                 class="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-amber-500" />
          <button @click="onAddFolder"
                  :disabled="!newFolderPath"
                  class="px-4 py-2 bg-amber-500 hover:bg-amber-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors">
            {{ $t('common.add') }}
          </button>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
          {{ $t('settings.agents.folderPathExample') }}
        </p>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import {
  PhotoIcon,
  EyeIcon,
  LinkIcon,
  LightBulbIcon,
  FolderIcon,
  FolderOpenIcon,
  ArrowPathIcon,
  TrashIcon,
  ClockIcon
} from '@heroicons/vue/24/outline'
import ToggleSwitch from '../ToggleSwitch.vue'
import { formatDateAbsolute } from '../../composables/useFormatters'

const props = defineProps({
  settings: {
    type: Object,
    required: true
  },
  modelSelectionSettings: {
    type: Object,
    default: () => ({
      visionModel: '',
      visionChainingEnabled: true
    })
  },
  webSearchThinkFirst: {
    type: Boolean,
    default: true
  },
  visionModels: {
    type: Array,
    default: () => []
  },
  fileSearchFolders: {
    type: Array,
    default: () => []
  },
  fileSearchStatus: {
    type: Object,
    default: null
  },
  visionServerStatus: {
    type: Object,
    default: null
  },
  visionIdleTimeout: {
    type: Number,
    default: 300
  }
})

const emit = defineEmits([
  'update:settings',
  'update:modelSelectionSettings',
  'update:webSearchThinkFirst',
  'update:visionIdleTimeout',
  'add-folder',
  'remove-folder',
  'reindex-folder'
])

// Local state
const localSettings = ref({ ...props.settings })
const localModelSettings = ref({ ...props.modelSelectionSettings })
const localThinkFirst = ref(props.webSearchThinkFirst)
const localVisionIdleTimeout = ref(props.visionIdleTimeout)
const newFolderPath = ref('')

// Sync with parent
watch(() => props.settings, (newVal) => {
  localSettings.value = { ...newVal }
}, { deep: true })

watch(() => props.modelSelectionSettings, (newVal) => {
  localModelSettings.value = { ...newVal }
}, { deep: true })

watch(() => props.webSearchThinkFirst, (newVal) => {
  localThinkFirst.value = newVal
})

watch(() => props.visionIdleTimeout, (newVal) => {
  localVisionIdleTimeout.value = newVal
})

watch(localSettings, (newVal) => {
  emit('update:settings', newVal)
}, { deep: true })

function onModelSettingsChange() {
  emit('update:modelSelectionSettings', localModelSettings.value)
}

function onThinkFirstChange(value) {
  emit('update:webSearchThinkFirst', value)
}

function onVisionTimeoutChange() {
  emit('update:visionIdleTimeout', localVisionIdleTimeout.value)
}

function formatTimeout(seconds) {
  if (seconds < 120) return `${seconds}s`
  return `${Math.floor(seconds / 60)} Min`
}

function onAddFolder() {
  if (newFolderPath.value) {
    emit('add-folder', newFolderPath.value)
    newFolderPath.value = ''
  }
}
</script>
