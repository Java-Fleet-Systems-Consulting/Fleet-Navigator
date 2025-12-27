<template>
  <div>
    <!-- Fleet Mates Section -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <UsersIcon class="w-5 h-5 text-blue-500" />
        {{ $t('mates.title') }}
      </h3>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        {{ $t('settings.mates.subtitle') }}
      </p>

      <!-- Pending Pairing Requests -->
      <div v-if="pendingPairingRequests.length > 0" class="mb-6">
        <h4 class="text-md font-semibold text-amber-600 dark:text-amber-400 mb-3 flex items-center gap-2">
          <ExclamationTriangleIcon class="w-5 h-5" />
          {{ $t('settings.mates.pendingRequests') }} ({{ pendingPairingRequests.length }})
        </h4>
        <div class="space-y-3">
          <div
            v-for="request in pendingPairingRequests"
            :key="request.requestId"
            class="p-4 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700/50"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <div class="p-2 rounded-lg" :class="getMateTypeColor(request.mateType)">
                  <component :is="getMateTypeIcon(request.mateType)" class="w-5 h-5 text-white" />
                </div>
                <div>
                  <h5 class="font-semibold text-gray-900 dark:text-white">{{ request.mateName }}</h5>
                  <p class="text-xs text-gray-500 dark:text-gray-400">{{ getMateTypeLabel(request.mateType) }}</p>
                </div>
              </div>
              <div class="flex items-center gap-2">
                <!-- Pairing Code Display -->
                <div class="flex gap-1 mr-4">
                  <span
                    v-for="(digit, index) in request.pairingCode.split('')"
                    :key="index"
                    class="w-8 h-10 flex items-center justify-center text-lg font-mono font-bold bg-gradient-to-br from-blue-500 to-cyan-500 text-white rounded"
                  >
                    {{ digit }}
                  </span>
                </div>
                <button
                  @click="$emit('reject-pairing', request.requestId)"
                  :disabled="processingPairing"
                  class="px-3 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-all disabled:opacity-50"
                >
                  <XMarkIcon class="w-5 h-5" />
                </button>
                <button
                  @click="$emit('approve-pairing', request.requestId)"
                  :disabled="processingPairing"
                  class="px-3 py-2 bg-gradient-to-r from-green-500 to-emerald-600 text-white rounded-lg hover:from-green-600 hover:to-emerald-700 transition-all shadow-lg disabled:opacity-50"
                >
                  <CheckIcon class="w-5 h-5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Trusted Mates List -->
      <div>
        <h4 class="text-md font-semibold text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
          <LinkIcon class="w-5 h-5" />
          {{ $t('settings.mates.connectedMates') }} ({{ trustedMates.length }})
        </h4>

        <div v-if="trustedMates.length === 0" class="p-6 text-center rounded-xl bg-gray-100 dark:bg-gray-800 border border-dashed border-gray-300 dark:border-gray-600">
          <UsersIcon class="w-12 h-12 text-gray-400 mx-auto mb-3" />
          <p class="text-gray-500 dark:text-gray-400">{{ $t('mates.noMates') }}</p>
          <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ $t('settings.mates.connectViaSystem') }}</p>
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="mate in trustedMates"
            :key="mate.mateId"
            class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 flex items-center justify-between"
          >
            <div class="flex items-center gap-3">
              <div class="p-2 rounded-lg" :class="getMateTypeColor(mate.mateType)">
                <component :is="getMateTypeIcon(mate.mateType)" class="w-5 h-5 text-white" />
              </div>
              <div>
                <h5 class="font-semibold text-gray-900 dark:text-white">{{ mate.name }}</h5>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ getMateTypeLabel(mate.mateType) }}
                  <span v-if="mate.lastSeen" class="ml-2">• {{ $t('mates.lastSeen') }}: {{ formatDateAbsolute(mate.lastSeen) }}</span>
                </p>
              </div>
            </div>
            <button
              @click="$emit('remove-mate', mate.mateId)"
              :disabled="removingMateId === mate.mateId"
              class="px-3 py-2 text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all disabled:opacity-50"
              :title="$t('mates.forgetDevice')"
            >
              <TrashIcon v-if="removingMateId !== mate.mateId" class="w-5 h-5" />
              <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
            </button>
          </div>
        </div>

        <!-- Forget All Button -->
        <div v-if="trustedMates.length > 1" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="$emit('forget-all-mates')"
            :disabled="forgettingMates"
            class="w-full px-4 py-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all border border-red-200 dark:border-red-800 flex items-center justify-center gap-2"
          >
            <TrashIcon v-if="!forgettingMates" class="w-5 h-5" />
            <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
            {{ forgettingMates ? $t('settings.mates.forgetting') : $t('settings.mates.forgetAllButton') }}
          </button>
        </div>
      </div>
    </section>

    <!-- Fleet Mates Model Assignment -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm mt-6">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <CpuChipIcon class="w-5 h-5 text-purple-500" />
        {{ $t('settings.mates.modelAssignment') }}
      </h3>

      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        {{ $t('settings.mates.modelAssignmentDesc') }}
      </p>

      <div class="space-y-4">
        <!-- Email Model (Thunderbird Mate) -->
        <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
            <span class="text-xl">📧</span>
            {{ $t('settings.models.emailModel') }}
            <span class="text-xs text-gray-500 dark:text-gray-400">(Thunderbird Mate)</span>
          </label>
          <select
            v-model="localMateModels.emailModel"
            @change="$emit('save-email-model', localMateModels.emailModel)"
            class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-blue-500"
          >
            <option value="">-- {{ $t('common.default') }} (llama3.2:3b) --</option>
            <option v-for="model in fastModels" :key="model.name" :value="model.name">
              {{ model.name }} ({{ formatSize(model.size) }})
            </option>
          </select>
          <p class="text-xs text-gray-500 mt-1">{{ $t('settings.models.emailModelDesc') }}</p>
        </div>

        <!-- Document Model (Writer Mate) -->
        <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
            <span class="text-xl">✍️</span>
            {{ $t('settings.models.documentModel') }}
            <span class="text-xs text-gray-500 dark:text-gray-400">(Writer Mate)</span>
          </label>
          <select
            v-model="localMateModels.documentModel"
            @change="$emit('save-document-model', localMateModels.documentModel)"
            class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-green-500"
          >
            <option value="">-- {{ $t('common.default') }} --</option>
            <option v-for="model in availableModels" :key="model.name" :value="model.name">
              {{ model.name }} ({{ formatSize(model.size) }})
            </option>
          </select>
          <p class="text-xs text-gray-500 mt-1">{{ $t('settings.models.documentModelDesc') }}</p>
        </div>

        <!-- Log Analysis Model (OS Mate) -->
        <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
            <span class="text-xl">📊</span>
            Log-Analyse-Modell
            <span class="text-xs text-gray-500 dark:text-gray-400">(OS Mate)</span>
          </label>
          <select
            v-model="localMateModels.logAnalysisModel"
            @change="$emit('save-log-analysis-model', localMateModels.logAnalysisModel)"
            class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-orange-500"
          >
            <option value="">-- Standard --</option>
            <option v-for="model in availableModels" :key="model.name" :value="model.name">
              {{ model.name }} ({{ formatSize(model.size) }})
            </option>
          </select>
          <p class="text-xs text-gray-500 mt-1">Für Log-Datei-Analyse und Fehlersuche</p>
        </div>

        <!-- Coder Model (FleetCoder) -->
        <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
            <span class="text-xl">💻</span>
            Coder-Modell
            <span class="text-xs text-gray-500 dark:text-gray-400">(FleetCoder)</span>
          </label>
          <select
            v-model="localMateModels.coderModel"
            @change="$emit('save-coder-model', localMateModels.coderModel)"
            class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-cyan-500"
          >
            <option value="">-- Standard --</option>
            <option v-for="model in availableModels" :key="model.name" :value="model.name">
              {{ model.name }} ({{ formatSize(model.size) }})
            </option>
          </select>
          <p class="text-xs text-gray-500 mt-1">Für Code-Assistenz und Programmierung (größeres Modell empfohlen: 14B+)</p>
        </div>
      </div>

      <div class="mt-4 p-3 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700/50">
        <div class="flex items-start gap-2">
          <InformationCircleIcon class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
          <p class="text-xs text-blue-800 dark:text-blue-200">
            <strong>Tipp:</strong> Für Email-Klassifizierung eignen sich schnelle Modelle wie <code>llama3.2:3b</code> oder <code>qwen2.5:7b</code>.
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import {
  UsersIcon,
  ExclamationTriangleIcon,
  XMarkIcon,
  CheckIcon,
  LinkIcon,
  TrashIcon,
  ArrowPathIcon,
  CpuChipIcon,
  InformationCircleIcon,
  ComputerDesktopIcon,
  EnvelopeIcon,
  DocumentIcon,
  GlobeAltIcon
} from '@heroicons/vue/24/outline'
import { formatDateAbsolute } from '../../composables/useFormatters'

const props = defineProps({
  trustedMates: {
    type: Array,
    default: () => []
  },
  pendingPairingRequests: {
    type: Array,
    default: () => []
  },
  processingPairing: {
    type: Boolean,
    default: false
  },
  removingMateId: {
    type: String,
    default: null
  },
  forgettingMates: {
    type: Boolean,
    default: false
  },
  mateModels: {
    type: Object,
    default: () => ({
      emailModel: '',
      documentModel: '',
      logAnalysisModel: '',
      coderModel: ''
    })
  },
  fastModels: {
    type: Array,
    default: () => []
  },
  availableModels: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits([
  'approve-pairing',
  'reject-pairing',
  'remove-mate',
  'forget-all-mates',
  'save-email-model',
  'save-document-model',
  'save-log-analysis-model',
  'save-coder-model'
])

// Local copy of mate models
const localMateModels = ref({ ...props.mateModels })

watch(() => props.mateModels, (newVal) => {
  localMateModels.value = { ...newVal }
}, { deep: true })

// Helper functions
function getMateTypeIcon(type) {
  switch (type) {
    case 'os': return ComputerDesktopIcon
    case 'mail': return EnvelopeIcon
    case 'office': return DocumentIcon
    case 'browser': return GlobeAltIcon
    default: return ComputerDesktopIcon
  }
}

function getMateTypeColor(type) {
  switch (type) {
    case 'os': return 'bg-blue-500'
    case 'mail': return 'bg-purple-500'
    case 'office': return 'bg-green-500'
    case 'browser': return 'bg-orange-500'
    default: return 'bg-gray-500'
  }
}

function getMateTypeLabel(type) {
  switch (type) {
    case 'os': return 'System-Agent'
    case 'mail': return 'E-Mail-Agent'
    case 'office': return 'Office-Agent'
    case 'browser': return 'Browser-Agent'
    default: return 'Fleet Mate'
  }
}

function formatSize(bytes) {
  if (!bytes) return '?'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return gb.toFixed(1) + ' GB'
  const mb = bytes / (1024 * 1024)
  return mb.toFixed(0) + ' MB'
}
</script>
