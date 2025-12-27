<template>
  <Transition name="modal">
    <div v-if="isOpen" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="close">
      <div class="bg-white/95 dark:bg-gray-800/95 backdrop-blur-xl rounded-2xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden border border-gray-200/50 dark:border-gray-700/50">
        <!-- Header with Gradient -->
        <div class="sticky top-0 bg-gradient-to-r from-fleet-orange-500/10 to-orange-500/10 dark:from-fleet-orange-500/20 dark:to-orange-500/20 backdrop-blur-sm border-b border-gray-200/50 dark:border-gray-700/50 px-6 py-4 flex justify-between items-center z-10">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-xl bg-gradient-to-br from-fleet-orange-500 to-orange-600 shadow-lg">
              <Cog6ToothIcon class="w-6 h-6 text-white" />
            </div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ $t('settings.title') }}</h2>
          </div>
          <button
            @click="close"
            class="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all transform hover:scale-110"
          >
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <!-- Tab Navigation -->
        <div class="flex flex-wrap border-b border-gray-200 dark:border-gray-700 px-6 bg-gray-50/50 dark:bg-gray-900/50 gap-1">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            @click="activeTab = tab.id"
            class="flex items-center gap-2 px-3 py-2 text-sm font-medium transition-all relative whitespace-nowrap rounded-lg"
            :class="activeTab === tab.id
              ? 'text-fleet-orange-600 dark:text-fleet-orange-400'
              : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'"
          >
            <component :is="tab.icon" class="w-4 h-4" />
            {{ tab.label }}
            <div
              v-if="activeTab === tab.id"
              class="absolute bottom-0 left-0 right-0 h-0.5 bg-fleet-orange-500"
            />
          </button>
        </div>

        <!-- Content with Custom Scrollbar -->
        <div class="overflow-y-auto p-6 space-y-6 custom-scrollbar" style="max-height: calc(90vh - 220px);">

          <!-- TAB: General Settings -->
          <div v-if="activeTab === 'general'">
            <GeneralSettingsTab
              :settings="settings"
              :voice-download-info="voiceDownloadInfo"
              :is-downloading-voices="isDownloadingVoices"
              @update:settings="settings = $event"
              @language-change="onLanguageChange"
              @download-voices="downloadVoicesForLanguage"
            />
          </div>

          <!-- TAB: Fleet Mates -->
          <div v-if="activeTab === 'mates'">
            <MatesSettingsTab
              :trusted-mates="trustedMates"
              :pending-pairing-requests="pendingPairingRequests"
              :processing-pairing="processingPairing"
              :removing-mate-id="removingMateId"
              :forgetting-mates="forgettingMates"
              :mate-models="mateModels"
              :fast-models="fastModels"
              :available-models="availableModels"
              @approve-pairing="approvePairingRequest"
              @reject-pairing="rejectPairingRequest"
              @remove-mate="removeTrustedMate"
              @forget-all-mates="forgetAllMates"
              @save-email-model="saveEmailModel"
              @save-document-model="saveDocumentModel"
              @save-log-analysis-model="saveLogAnalysisModel"
              @save-coder-model="saveCoderModel"
            />
          </div>

          <!-- TAB: LLM Provider -->
          <div v-if="activeTab === 'providers'">
            <ProviderSettings />
          </div>

          <!-- TAB: Custom Modell (System-Prompts + Sampling) -->
          <div v-if="activeTab === 'customModels'">
            <CustomModelsTab
              :system-prompts="systemPrompts"
              :sampling-params="samplingParams"
              @create-prompt="showPromptEditor = true; editingPrompt = null; resetPromptForm()"
              @edit-prompt="editSystemPrompt"
              @delete-prompt="deleteSystemPrompt"
              @activate-prompt="activateSystemPrompt"
              @update:sampling-params="samplingParams = $event"
            />
          </div>

          <!-- TAB: Personal Info -->
          <div v-if="activeTab === 'personal'">
            <PersonalInfoTab ref="personalInfoTabRef" />
          </div>

          <!-- TAB: Observer (Finanz- und Wirtschaftsdaten) -->
          <div v-if="activeTab === 'observer'">
            <ObserverSettings />
          </div>

          <!-- TAB: Agents -->
          <div v-if="activeTab === 'agents'">
            <AgentsSettingsTab
              :settings="settings"
              :model-selection-settings="modelSelectionSettings"
              :web-search-think-first="webSearchThinkFirst"
              :vision-models="visionModels"
              :file-search-folders="fileSearchFolders"
              :file-search-status="fileSearchStatus"
              @update:settings="settings = $event"
              @update:model-selection-settings="modelSelectionSettings = $event"
              @update:web-search-think-first="webSearchThinkFirst = $event"
              @add-folder="addSearchFolder"
              @remove-folder="removeSearchFolder"
              @reindex-folder="reindexFolder"
            />
          </div>

          <!-- TAB: Web Search Settings -->
          <div v-if="activeTab === 'web-search'">
            <WebSearchSettingsTab
              :web-search-settings="webSearchSettings"
              :testing-search="testingBraveSearch || testingSearxng"
              @update:web-search-settings="webSearchSettings = $event"
              @test-brave="testBraveSearch"
              @test-searxng="testCustomSearxng"
              @save-settings="saveWebSearchSettings"
            />
          </div>

          <!-- TAB: Voice (STT/TTS) -->
          <div v-if="activeTab === 'voice'">
            <VoiceSettingsTab
              :tts-enabled="settings.ttsEnabled"
              :voice-downloading="voiceDownloading"
              :voice-download-component="voiceDownloadComponent"
              :voice-download-status="voiceDownloadStatus"
              :voice-download-progress="voiceDownloadProgress"
              :voice-download-speed="voiceDownloadSpeed"
              :voice-models="voiceModels"
              :voice-assistant-settings="voiceAssistantSettings"
              @toggle-tts="toggleTtsEnabled"
              @download-whisper="downloadWhisper"
              @download-piper="downloadPiper"
              @download-model="downloadVoiceModel"
              @select-whisper-model="selectWhisperModel"
              @select-piper-voice="selectPiperVoice"
              @update:voice-assistant-settings="voiceAssistantSettings = $event"
            />
          </div>

          <!-- TAB: Addons/Erweiterungen -->
          <div v-if="activeTab === 'addons'">
            <AddonsSettingsTab
              :tesseract-status="tesseractStatus"
              :tesseract-downloading="tesseractDownloading"
              :tesseract-download-progress="tesseractDownloadProgress"
              :tesseract-download-message="tesseractDownloadMessage"
              @download-tesseract="downloadTesseract"
              @postgres-status-change="onPostgresStatusChange"
            />
          </div>

          <!-- TAB: Danger Zone -->
          <div v-if="activeTab === 'danger'">
            <DangerZoneTab
              v-model:resetSelection="resetSelection"
              :resetting="resetting"
              @reset-all="handleResetAll"
            />
          </div>

        </div>

        <!-- System Prompt Editor Modal -->
        <Transition name="modal">
          <div v-if="showPromptEditor" class="absolute inset-0 bg-black/70 flex items-center justify-center z-50 p-4" @click.self="showPromptEditor = false">
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
              <div class="p-5 border-b border-gray-200 dark:border-gray-700">
                <h4 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ editingPrompt ? $t('settings.prompts.editTitle') : $t('settings.prompts.newTitle') }}
                </h4>
              </div>

              <div class="p-5 space-y-4">
                <!-- Name -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {{ $t('settings.prompts.name') }}
                  </label>
                  <input
                    v-model="promptForm.name"
                    type="text"
                    :placeholder="$t('settings.prompts.namePlaceholder')"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                </div>

                <!-- Content -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {{ $t('settings.prompts.promptText') }}
                  </label>
                  <textarea
                    v-model="promptForm.content"
                    rows="8"
                    :placeholder="$t('settings.prompts.promptPlaceholder')"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
                  ></textarea>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ promptForm.content.length }} {{ $t('settings.prompts.characters') }}
                  </p>
                </div>

                <!-- Is Default -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ $t('settings.prompts.setAsDefaultLabel') }}
                    </label>
                    <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {{ $t('settings.prompts.setAsDefaultHint') }}
                    </p>
                  </div>
                  <input
                    type="checkbox"
                    v-model="promptForm.isDefault"
                    class="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                  >
                </div>
              </div>

              <div class="p-5 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
                <button
                  @click="showPromptEditor = false"
                  class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  {{ $t('common.cancel') }}
                </button>
                <button
                  @click="saveSystemPrompt"
                  :disabled="!promptForm.name.trim() || !promptForm.content.trim()"
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  <CheckIcon class="w-4 h-4" />
                  {{ editingPrompt ? $t('settings.prompts.update') : $t('common.create') }}
                </button>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Footer with Gradient -->
        <div class="sticky bottom-0 bg-gray-50/90 dark:bg-gray-900/90 backdrop-blur-sm border-t border-gray-200/50 dark:border-gray-700/50 px-6 py-4 flex justify-between">
          <button
            @click="resetToDefaults"
            class="px-4 py-2 rounded-xl text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-gray-700 transition-all flex items-center gap-2"
          >
            <ArrowPathIcon class="w-4 h-4" />
            {{ $t('common.reset') }}
          </button>
          <div class="flex gap-3">
            <button
              @click="close"
              class="px-5 py-2 rounded-xl border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all transform hover:scale-105"
            >
              {{ $t('common.cancel') }}
            </button>
            <button
              @click="save"
              :disabled="saving"
              class="px-6 py-2 rounded-xl bg-gradient-to-r from-fleet-orange-500 to-orange-600 hover:from-fleet-orange-400 hover:to-orange-500 text-white font-semibold shadow-lg hover:shadow-xl transition-all transform hover:scale-105 disabled:opacity-50 flex items-center gap-2"
            >
              <CheckIcon v-if="!saving" class="w-5 h-5" />
              <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
              {{ saving ? $t('common.saving') : $t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import {
  Cog6ToothIcon,
  XMarkIcon,
  GlobeAltIcon,
  LanguageIcon,
  SunIcon,
  CpuChipIcon,
  SparklesIcon,
  ExclamationTriangleIcon,
  CodeBracketIcon,
  BoltIcon,
  InformationCircleIcon,
  AdjustmentsHorizontalIcon,
  DocumentTextIcon,
  FireIcon,
  DocumentDuplicateIcon,
  PhotoIcon,
  EyeIcon,
  LinkIcon,
  WrenchScrewdriverIcon,
  Bars3Icon,
  HashtagIcon,
  BugAntIcon,
  ArrowPathIcon,
  CheckIcon,
  CheckCircleIcon,
  UserIcon,
  TrashIcon,
  ShieldExclamationIcon,
  UsersIcon,
  MagnifyingGlassIcon,
  StarIcon,
  ServerIcon,
  PlusIcon,
  FolderIcon,
  FolderOpenIcon,
  ComputerDesktopIcon,
  EnvelopeIcon,
  DocumentIcon,
  MicrophoneIcon,
  SpeakerWaveIcon,
  ArrowDownTrayIcon,
  ArrowsRightLeftIcon,
  LightBulbIcon,
  ChartBarIcon,
  PuzzlePieceIcon,
  DocumentMagnifyingGlassIcon
} from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../stores/settingsStore'
import { useChatStore } from '../stores/chatStore'
import PersonalInfoTab from './PersonalInfoTab.vue'
import ProviderSettings from './ProviderSettings.vue'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import { formatDateAbsolute } from '../composables/useFormatters'
import api from '../services/api'
import { secureFetch } from '../utils/secureFetch'
import ToggleSwitch from './ToggleSwitch.vue'
import SimpleSamplingParams from './SimpleSamplingParams.vue'
import VRAMSettings from './settings/VRAMSettings.vue'
import PostgreSQLMigration from './settings/PostgreSQLMigration.vue'
import VoiceStore from './VoiceStore.vue'
import ObserverSettings from './ObserverSettings.vue'
import { filterVisionModels, filterCodeModels } from '../utils/modelFilters'

// New refactored tab components
import GeneralSettingsTab from './settings/GeneralSettingsTab.vue'
import MatesSettingsTab from './settings/MatesSettingsTab.vue'
import AgentsSettingsTab from './settings/AgentsSettingsTab.vue'
import AddonsSettingsTab from './settings/AddonsSettingsTab.vue'
import DangerZoneTab from './settings/DangerZoneTab.vue'
import CustomModelsTab from './settings/CustomModelsTab.vue'
import WebSearchSettingsTab from './settings/WebSearchSettingsTab.vue'
import VoiceSettingsTab from './settings/VoiceSettingsTab.vue'

const { success, error: errorToast } = useToast()
const { confirm, confirmDelete } = useConfirmDialog()

const props = defineProps({
  isOpen: Boolean,
  initialTab: String
})

const emit = defineEmits(['close', 'save'])

const settingsStore = useSettingsStore()
const chatStore = useChatStore()

// PostgreSQL Status
const postgresConnected = ref(false)

function onPostgresStatusChange(connected) {
  postgresConnected.value = connected
  console.log('PostgreSQL Status:', connected ? 'Verbunden' : 'SQLite')
}

// Local copy of settings for editing
const settings = ref({ ...settingsStore.settings })
const saving = ref(false)
const resetting = ref(false)

// Voice Download Dialog für Sprachwechsel
const showVoiceDownloadDialog = ref(false)
const voiceDownloadInfo = ref({ availableVoices: [], installedVoices: [] })
const isDownloadingVoices = ref(false)

// Sampling Parameters
const samplingParams = ref({})

// Reset selection checkboxes
const resetSelection = ref({
  chats: true,
  projects: true,
  customModels: false,  // Custom Models standardmäßig NICHT löschen
  settings: true,
  personalInfo: true,
  templates: true,
  stats: true
})

// Active tab
const activeTab = ref(props.initialTab || 'general')

// Watch for tab changes to reload data when needed
watch(activeTab, async (newTab, oldTab) => {
  if (newTab === 'mates') {
    await loadTrustedMatesCount()
    startPairingPoll()
  } else if (oldTab === 'mates') {
    stopPairingPoll()
  }

  if (newTab === 'voice') {
    await loadVoiceStatus()
    await loadVoiceModels()
    await loadTtsSetting()
  }
})

// Sync cpuOnly with settingsStore (persistent) - für CPU-Only Mode Toggle
watch(() => settings.value.cpuOnly, (newValue) => {
  settingsStore.settings.cpuOnly = newValue
  console.log('🖥️ CPU-Only Mode:', newValue ? 'aktiviert' : 'deaktiviert')
}, { immediate: false })

// Ref to PersonalInfoTab
const personalInfoTabRef = ref(null)

// Fleet Mates Pairing
const trustedMatesCount = ref(0)
const trustedMates = ref([])
const forgettingMates = ref(false)
const removingMateId = ref(null)
const pendingPairingRequests = ref([])
const processingPairing = ref(false)
let pairingPollInterval = null

// Mate type helpers
const getMateTypeIcon = (type) => {
  switch (type) {
    case 'os': return ComputerDesktopIcon
    case 'mail': return EnvelopeIcon
    case 'office': return DocumentIcon
    case 'browser': return GlobeAltIcon
    default: return ComputerDesktopIcon
  }
}

const getMateTypeColor = (type) => {
  switch (type) {
    case 'os': return 'bg-blue-500'
    case 'mail': return 'bg-purple-500'
    case 'office': return 'bg-green-500'
    case 'browser': return 'bg-orange-500'
    default: return 'bg-gray-500'
  }
}

const getMateTypeLabel = (type) => {
  switch (type) {
    case 'os': return 'System-Agent'
    case 'mail': return 'E-Mail-Agent'
    case 'office': return 'Office-Agent'
    case 'browser': return 'Browser-Agent'
    default: return 'Fleet Mate'
  }
}

// File Search (OS Mate RAG)
const fileSearchFolders = ref([])
const fileSearchStatus = ref(null)
const newFolderPath = ref('')

async function loadFileSearchStatus() {
  try {
    const response = await fetch('/api/file-search/status')
    if (response.ok) {
      fileSearchStatus.value = await response.json()
      fileSearchFolders.value = fileSearchStatus.value.searchFolders || []
    }
  } catch (err) {
    console.error('Failed to load file search status:', err)
  }
}

// ========================================
// Tesseract OCR Status
// ========================================

const tesseractStatus = ref({
  installed: false,
  binaryPath: '',
  languages: [],
  dataDir: ''
})
const tesseractDownloading = ref(false)
const tesseractDownloadProgress = ref(0)
const tesseractDownloadMessage = ref('')

async function loadTesseractStatus() {
  try {
    const response = await fetch('/api/setup/tesseract/status')
    if (response.ok) {
      tesseractStatus.value = await response.json()
    }
  } catch (err) {
    console.error('Failed to load Tesseract status:', err)
  }
}

async function downloadTesseract() {
  if (tesseractDownloading.value) return

  tesseractDownloading.value = true
  tesseractDownloadProgress.value = 0
  tesseractDownloadMessage.value = 'Starte Download...'

  try {
    const eventSource = new EventSource('/api/setup/tesseract/download')

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)

        if (data.status === 'complete') {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadProgress.value = 100
          tesseractDownloadMessage.value = 'Installation abgeschlossen!'
          success('Tesseract OCR erfolgreich installiert!')
          // Status neu laden
          loadTesseractStatus()
          return
        }

        if (data.error) {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadMessage.value = 'Fehler: ' + data.error
          errorToast('Tesseract-Installation fehlgeschlagen: ' + data.error)
          return
        }

        tesseractDownloadProgress.value = data.percent || 0
        tesseractDownloadMessage.value = data.message || 'Downloading...'

        if (data.done) {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadProgress.value = 100
          success('Tesseract OCR erfolgreich installiert!')
          loadTesseractStatus()
        }
      } catch (e) {
        console.error('Error parsing SSE:', e)
      }
    }

    eventSource.onerror = (error) => {
      console.error('SSE Error:', error)
      eventSource.close()
      tesseractDownloading.value = false
      // Prüfe ob Download erfolgreich war
      if (tesseractDownloadProgress.value >= 99) {
        success('Tesseract OCR erfolgreich installiert!')
        loadTesseractStatus()
      } else {
        tesseractDownloadMessage.value = 'Verbindungsfehler'
        errorToast('Download-Verbindung unterbrochen')
      }
    }
  } catch (err) {
    console.error('Failed to download Tesseract:', err)
    tesseractDownloading.value = false
    tesseractDownloadMessage.value = 'Fehler: ' + err.message
    errorToast('Tesseract-Download fehlgeschlagen')
  }
}

// ========================================
// Sprachwechsel mit Voice-Download Dialog
// ========================================

async function onLanguageChange() {
  const newLocale = settings.value.language
  console.log('[Settings] Sprachwechsel zu:', newLocale)

  try {
    // Backend informieren
    const response = await fetch('/api/settings/language', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ locale: newLocale })
    })

    if (response.ok) {
      const result = await response.json()
      console.log('[Settings] Sprache gewechselt:', result)

      // i18n aktualisieren
      const { locale } = useI18n()
      locale.value = newLocale
      localStorage.setItem('fleet-navigator-locale', newLocale)
      document.documentElement.lang = newLocale

      // Prüfen ob Voice-Download nötig
      if (result.needsVoiceDownload && result.availableVoices?.length > 0) {
        voiceDownloadInfo.value = result
        showVoiceDownloadDialog.value = true
      }

      showToast(`Sprache auf ${newLocale.toUpperCase()} gewechselt`, 'success')
    }
  } catch (err) {
    console.error('[Settings] Sprachwechsel fehlgeschlagen:', err)
    showToast('Sprachwechsel fehlgeschlagen', 'error')
  }
}

async function downloadVoicesForLanguage() {
  const locale = settings.value.language
  const voices = voiceDownloadInfo.value.availableVoices || []

  if (voices.length === 0) {
    showVoiceDownloadDialog.value = false
    return
  }

  isDownloadingVoices.value = true

  try {
    // Piper-Stimme herunterladen (mit korrektem Endpoint via SSE)
    const voice = voices[0]
    // Voice-ID Format: de_DE-thorsten-medium oder tr_TR-fahrettin-medium
    const voiceId = voice.id

    console.log('[Settings] Lade Stimme herunter:', voiceId)

    // SSE-basierter Download wie im SetupWizard
    await new Promise((resolve, reject) => {
      const eventSource = new EventSource(
        `/api/setup/download-voice?component=piper&modelId=${encodeURIComponent(voiceId)}&lang=${locale}`
      )

      eventSource.onmessage = (event) => {
        try {
          const progress = JSON.parse(event.data)

          if (progress.message) {
            console.log('[Settings] Voice-Download:', progress.message)
          }

          if (progress.done) {
            eventSource.close()
            if (progress.error) {
              reject(new Error(progress.error))
            } else {
              resolve()
            }
          }
        } catch (e) {
          console.error('[Settings] Progress Parse Fehler:', e)
        }
      }

      eventSource.onerror = (error) => {
        console.error('[Settings] SSE Fehler:', error)
        eventSource.close()
        reject(new Error('Download-Verbindung unterbrochen'))
      }
    })

    showToast(`Stimme "${voice.name}" heruntergeladen!`, 'success')
    showVoiceDownloadDialog.value = false

    // Voice-Status neu laden
    await loadVoiceStatus()
  } catch (err) {
    console.error('[Settings] Voice-Download fehlgeschlagen:', err)
    showToast(`Download fehlgeschlagen: ${err.message}`, 'error')
  } finally {
    isDownloadingVoices.value = false
  }
}

async function addSearchFolder(folderPath = null) {
  const pathToAdd = folderPath || newFolderPath.value
  if (!pathToAdd) return

  try {
    const response = await secureFetch('/api/file-search/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderPath: pathToAdd })
    })

    if (response.ok) {
      newFolderPath.value = ''
      await loadFileSearchStatus()
      success('Ordner hinzugefügt und Indexierung gestartet')
    } else {
      const data = await response.json()
      errorToast(data.error || 'Fehler beim Hinzufügen')
    }
  } catch (err) {
    console.error('Failed to add search folder:', err)
    errorToast('Fehler beim Hinzufügen des Ordners')
  }
}

async function removeSearchFolder(folderId) {
  try {
    await secureFetch(`/api/file-search/folders/${folderId}`, { method: 'DELETE' })
    await loadFileSearchStatus()
    success('Ordner entfernt')
  } catch (err) {
    console.error('Failed to remove folder:', err)
    errorToast('Fehler beim Entfernen')
  }
}

async function reindexFolder(folderId) {
  try {
    await secureFetch(`/api/file-search/folders/${folderId}/reindex`, { method: 'POST' })
    success('Neu-Indexierung gestartet')
    // Refresh status after a delay
    setTimeout(loadFileSearchStatus, 2000)
  } catch (err) {
    console.error('Failed to reindex folder:', err)
    errorToast('Fehler beim Indexieren')
  }
}

// formatDate importiert aus useFormatters.js als formatDateAbsolute

// Tab configuration
const tabs = [
  { id: 'general', label: 'Allgemein', icon: GlobeAltIcon },
  { id: 'mates', label: 'Fleet Mates', icon: UsersIcon },
  { id: 'providers', label: 'LLM Provider', icon: CpuChipIcon },
  { id: 'customModels', label: 'Custom Modell', icon: AdjustmentsHorizontalIcon },
  { id: 'personal', label: 'Persönliche Daten', icon: UserIcon },
  { id: 'observer', label: 'Observer', icon: ChartBarIcon },
  { id: 'agents', label: 'Agents', icon: SparklesIcon },
  { id: 'web-search', label: 'Web-Suche', icon: MagnifyingGlassIcon },
  { id: 'voice', label: 'Sprache', icon: MicrophoneIcon },
  { id: 'addons', label: 'Erweiterungen', icon: PuzzlePieceIcon },
  { id: 'danger', label: 'Danger Zone', icon: ShieldExclamationIcon }
]

// Model selection settings
const modelSelectionSettings = ref({
  enabled: true,
  codeModel: 'qwen2.5-coder:7b',
  fastModel: 'llama3.2:3b',
  visionModel: 'llava:13b',
  defaultModel: 'qwen2.5-coder:7b',
  visionChainingEnabled: true,
  visionChainingSmartSelection: true
})

// Web Search Think First (LLM denkt erst nach, dann Websuche bei Unsicherheit)
const webSearchThinkFirst = ref(true)

// Load Think First setting from localStorage
function loadThinkFirstSetting() {
  try {
    const chainingSettings = JSON.parse(localStorage.getItem('chainingSettings') || '{}')
    webSearchThinkFirst.value = chainingSettings.webSearchThinkFirst ?? true
  } catch (e) {
    console.error('Failed to load Think First setting:', e)
  }
}

// Save Think First setting to localStorage
watch(webSearchThinkFirst, (newValue) => {
  try {
    const chainingSettings = JSON.parse(localStorage.getItem('chainingSettings') || '{}')
    chainingSettings.webSearchThinkFirst = newValue
    localStorage.setItem('chainingSettings', JSON.stringify(chainingSettings))
    console.log('💾 Think First setting saved:', newValue)
  } catch (e) {
    console.error('Failed to save Think First setting:', e)
  }
})

// Load on component mount
loadThinkFirstSetting()

// Fleet Mates Model Settings
const mateModels = ref({
  emailModel: '',
  documentModel: '',
  logAnalysisModel: '',
  coderModel: ''
})

// Web Search Settings (Brave API + SearXNG)
const webSearchSettings = ref({
  braveApiKey: '',
  braveConfigured: false,
  searchCount: 0,
  searchLimit: 2000,
  remainingSearches: 2000,
  currentMonth: '',
  customSearxngInstance: '',  // Eigene Instanz (Priorität 1)
  searxngInstances: [],       // Öffentliche Fallback-Instanzen
  searxngTotalCount: 0,       // Gesamte SearXNG-Suchen
  searxngMonthCount: 0,       // SearXNG-Suchen diesen Monat
  // Feature Flags
  queryOptimizationEnabled: true,
  contentScrapingEnabled: true,
  multiQueryEnabled: false,
  reRankingEnabled: true,
  queryOptimizationModel: 'llama3.2:3b',
  effectiveOptimizationModel: null,  // Das tatsächlich verwendete Modell (nach Fallback)
  // UI Animation
  webSearchAnimation: 'data-wave'  // Animation: data-wave, orbit, radar, constellation
})

// Animation Options für Dropdown
const animationOptions = [
  { value: 'data-wave', label: '🌊 Data Wave', description: 'Fließende Datenwelle' },
  { value: 'orbit', label: '🌐 Orbiting Network', description: 'Kreisende Datenpunkte' },
  { value: 'radar', label: '📡 Radar Scan', description: 'Radar-Scanning-Effekt' },
  { value: 'constellation', label: '✨ Constellation', description: 'Sternbild-Netzwerk' }
]

// Voice Settings (STT/TTS)
const voiceStatus = ref({
  initialized: false,
  whisper: { available: false, binaryFound: false, modelFound: false, model: '' },
  piper: { available: false, binaryFound: false, voiceFound: false, voice: '' }
})
const voiceModels = ref({
  whisper: [],
  piper: [],
  currentWhisper: '',
  currentPiper: '',
  whisperBinary: false,
  piperBinary: false
})
const voiceDownloading = ref(false)
const voiceDownloadComponent = ref('')  // 'whisper' oder 'piper' - welche Komponente wird gerade geladen
const voiceDownloadStatus = ref('')
const voiceDownloadProgress = ref(0)
const voiceDownloadSpeed = ref('')
const piperLanguageFilter = ref('de')
const ttsEnabled = ref(true)

// Voice Assistant Settings
const voiceAssistantSettings = ref({
  enabled: false,
  wakeWord: 'hey_ewa',
  customWakeWord: '',
  autoStop: true,
  quietHoursEnabled: false,
  quietHoursStart: '22:00',
  quietHoursEnd: '07:00'
})

// VoiceStore ref
const voiceStoreRef = ref(null)

// TTS Toggle
async function toggleTtsEnabled() {
  ttsEnabled.value = !ttsEnabled.value
  try {
    await api.updateSettings({ ttsEnabled: ttsEnabled.value })
    // Auch im localStorage für schnellen Zugriff
    localStorage.setItem('ttsEnabled', String(ttsEnabled.value))
  } catch (err) {
    console.error('Failed to save TTS setting:', err)
  }
}

// Load TTS setting on mount
async function loadTtsSetting() {
  // Erst localStorage prüfen für schnellen Start
  const stored = localStorage.getItem('ttsEnabled')
  if (stored !== null) {
    ttsEnabled.value = stored === 'true'
  }
  // Dann vom Backend laden
  try {
    const settings = await api.getVoiceAssistantSettings()
    if (settings?.ttsEnabled !== undefined) {
      ttsEnabled.value = settings.ttsEnabled
      localStorage.setItem('ttsEnabled', String(ttsEnabled.value))
    }
  } catch (err) {
    // Endpoint nicht implementiert - ignorieren, localStorage-Wert behalten
    console.debug('Voice assistant settings not available:', err.message)
  }
}

// Voice Assistant Functions
async function loadVoiceAssistantSettings() {
  try {
    const response = await fetch('/api/voice-assistant/settings')
    if (response.ok) {
      const data = await response.json()
      voiceAssistantSettings.value = {
        enabled: data.enabled ?? false,
        wakeWord: data.wakeWord ?? 'hey_ewa',
        customWakeWord: data.customWakeWord ?? '',
        autoStop: data.autoStop ?? true,
        quietHoursEnabled: data.quietHoursEnabled ?? false,
        quietHoursStart: data.quietHoursStart ?? '22:00',
        quietHoursEnd: data.quietHoursEnd ?? '07:00'
      }
    }
  } catch (err) {
    console.error('Failed to load voice assistant settings:', err)
  }
}

async function saveVoiceAssistantSettings() {
  try {
    await fetch('/api/voice-assistant/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(voiceAssistantSettings.value)
    })
  } catch (err) {
    console.error('Failed to save voice assistant settings:', err)
  }
}

function getWakeWordDisplay(wakeWord) {
  switch (wakeWord) {
    case 'hey_ewa':
      return 'Hey Ewa'
    case 'ewa':
      return 'Ewa'
    case 'custom':
      return voiceAssistantSettings.value.customWakeWord || 'Eigenes Wake Word'
    default:
      return 'Hey Ewa'
  }
}

// Download custom voice from HuggingFace
async function downloadCustomVoice() {
  if (!customVoiceId.value) return

  customVoiceError.value = ''
  customVoiceSuccess.value = ''
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'custom'
  voiceDownloadStatus.value = 'Lade Stimme herunter...'
  voiceDownloadProgress.value = 0

  try {
    // Parse voice ID - expected format: locale-name-quality (e.g., de_DE-mls-medium)
    let voiceId = customVoiceId.value.trim()

    // If it's a URL, extract the voice ID
    if (voiceId.includes('huggingface.co')) {
      const match = voiceId.match(/([a-z]{2}_[A-Z]{2}-[\w]+-[\w]+)/)
      if (match) {
        voiceId = match[1]
      }
    }

    // Validate format
    const voicePattern = /^[a-z]{2}_[A-Z]{2}-[\w]+-[\w]+$/
    if (!voicePattern.test(voiceId)) {
      throw new Error('Ungültiges Format. Erwartet: locale-name-quality (z.B. de_DE-mls-medium)')
    }

    // Use existing download API
    const eventSource = new EventSource(`/api/voice/download-model?component=piper&model=${voiceId}`)

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        console.log('Custom voice download progress:', data)

        if (data.percent) {
          voiceDownloadProgress.value = data.percent
        }
        if (data.speedMBps) {
          voiceDownloadSpeed.value = `${data.speedMBps.toFixed(1)} MB/s`
        }
        voiceDownloadStatus.value = data.status === 'downloading'
          ? `Lade ${data.file || voiceId}... ${Math.round(data.percent || 0)}%`
          : data.status

        if (data.status === 'done' || data.status === 'complete') {
          eventSource.close()
          voiceDownloading.value = false
          voiceDownloadComponent.value = ''
          customVoiceSuccess.value = `Stimme "${voiceId}" erfolgreich installiert!`
          customVoiceId.value = ''
          loadVoiceModels()
        } else if (data.status === 'error') {
          eventSource.close()
          voiceDownloading.value = false
          voiceDownloadComponent.value = ''
          customVoiceError.value = data.error || 'Download fehlgeschlagen'
        }
      } catch (e) {
        console.error('Error parsing SSE data:', e)
      }
    }

    eventSource.onerror = (err) => {
      console.error('SSE error:', err)
      eventSource.close()
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      customVoiceError.value = 'Verbindungsfehler beim Download'
    }

  } catch (err) {
    console.error('Custom voice download failed:', err)
    voiceDownloading.value = false
    voiceDownloadComponent.value = ''
    customVoiceError.value = err.message || 'Download fehlgeschlagen'
  }
}

// Filtered Piper voices based on language
const filteredPiperVoices = computed(() => {
  if (!voiceModels.value.piper) return []
  if (piperLanguageFilter.value === 'all') return voiceModels.value.piper
  return voiceModels.value.piper.filter(v =>
    v.language?.startsWith(piperLanguageFilter.value === 'de' ? 'de' : 'en')
  )
})

// Load Voice Status
async function loadVoiceStatus() {
  try {
    const status = await api.getVoiceStatus()
    voiceStatus.value = status
    console.log('🎤 Voice status loaded:', status)
  } catch (err) {
    console.warn('Failed to load voice status:', err)
  }
}

// Load Voice Models with installed status
async function loadVoiceModels() {
  try {
    const models = await api.getVoiceModels()
    voiceModels.value = models
    console.log('🎤 Voice models loaded:', models)
  } catch (err) {
    console.warn('Failed to load voice models:', err)
  }
}

// Select Whisper Model
async function selectWhisperModel(modelId) {
  try {
    await api.setVoiceConfig({ whisperModel: modelId })
    voiceModels.value.currentWhisper = modelId
    success(`Whisper-Modell "${modelId}" aktiviert`)
    await loadVoiceStatus()
  } catch (err) {
    errorToast('Fehler beim Aktivieren: ' + err.message)
  }
}

// Select Piper Voice
async function selectPiperVoice(voiceId) {
  try {
    await api.setVoiceConfig({ piperVoice: voiceId })
    voiceModels.value.currentPiper = voiceId
    success(`Stimme "${voiceId}" aktiviert`)
    await loadVoiceStatus()
  } catch (err) {
    errorToast('Fehler beim Aktivieren: ' + err.message)
  }
}

// Download specific model
function downloadSpecificModel(component, modelId) {
  voiceDownloading.value = true
  voiceDownloadStatus.value = `Lade ${component === 'whisper' ? 'Whisper-Modell' : 'Piper-Stimme'} "${modelId}"...`
  voiceDownloadProgress.value = 0

  api.downloadVoiceModel(component, modelId, (progress) => {
    voiceDownloadStatus.value = `${progress.component}: ${progress.file || modelId} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadStatus.value = ''
      voiceDownloadSpeed.value = ''
      loadVoiceModels()
      loadVoiceStatus()
      success(`${component === 'whisper' ? 'Modell' : 'Stimme'} erfolgreich installiert!`)
    } else if (progress.status === 'error') {
      voiceDownloading.value = false
      errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
    }
  })
}

// Download Whisper (Binary + Model)
async function downloadWhisper() {
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'whisper'
  voiceDownloadStatus.value = 'Starte Whisper Download...'
  voiceDownloadProgress.value = 0

  // Nur Whisper herunterladen (nicht Piper)
  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `Whisper: ${progress.file || 'binary'} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'error' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      if (progress.status === 'done' || progress.status === 'complete') {
        success('Whisper erfolgreich installiert!')
      } else if (progress.status === 'error') {
        errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
      }
    }
  }, 'whisper')  // <-- Nur Whisper herunterladen
}

// Download Piper (Binary + Voice)
async function downloadPiper() {
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'piper'
  voiceDownloadStatus.value = 'Starte Piper Download...'
  voiceDownloadProgress.value = 0

  // Nur Piper herunterladen (nicht Whisper)
  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `Piper: ${progress.file || 'binary'} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'error' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      if (progress.status === 'done' || progress.status === 'complete') {
        success('Piper erfolgreich installiert!')
      } else if (progress.status === 'error') {
        errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
      }
    }
  }, 'piper')  // <-- Nur Piper herunterladen
}

// Download All Voice Models
async function downloadAllVoice() {
  voiceDownloading.value = true
  voiceDownloadStatus.value = 'Starte Downloads...'
  voiceDownloadProgress.value = 0

  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `${progress.component}: ${progress.file} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done') {
      voiceDownloading.value = false
      voiceDownloadStatus.value = ''
      voiceDownloadSpeed.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      success('Voice-Modelle erfolgreich installiert!')
    } else if (progress.status === 'error') {
      voiceDownloading.value = false
      errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
    }
  })
}

const defaultSearxngInstances = [
  'https://search.sapti.me',
  'https://searx.tiekoetter.com',
  'https://priv.au',
  'https://search.ononoki.org',
  'https://search.bus-hit.me',
  'https://paulgo.io'
]

const testingSearch = ref(false)

// Computed: Farbe des Zählers basierend auf Verbrauch
const searchCountColor = computed(() => {
  const percent = (webSearchSettings.value.searchCount / webSearchSettings.value.searchLimit) * 100
  if (percent >= 90) return 'text-red-500'
  if (percent >= 70) return 'text-yellow-500'
  return 'text-green-500'
})

// Computed: Prozent für Progress Bar
const searchCountPercent = computed(() => {
  return Math.min(100, (webSearchSettings.value.searchCount / webSearchSettings.value.searchLimit) * 100)
})

// Available models
const availableModels = ref([])

// Fast models (< 10GB, good for Mates)
const fastModels = computed(() => {
  return availableModels.value.filter(m => m.size && m.size < 10 * 1024 * 1024 * 1024)
})

// Format file size
function formatSize(bytes) {
  if (!bytes) return '?'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)}GB`
  const mb = bytes / (1024 * 1024)
  return `${mb.toFixed(0)}MB`
}

// System Prompts Management
const systemPrompts = ref([])
const showPromptEditor = ref(false)
const editingPrompt = ref(null)
const promptForm = ref({
  name: '',
  content: '',
  isDefault: false
})

// Filtered models for specific use cases
const visionModels = computed(() => {
  const allModelNames = availableModels.value.map(m => m.name)
  const filtered = filterVisionModels(allModelNames)
  console.log('Vision Models:', filtered) // Debug
  return filtered
})
const codeModels = computed(() => {
  const allModelNames = availableModels.value.map(m => m.name)
  const filtered = filterCodeModels(allModelNames)
  console.log('Code Models:', filtered) // Debug
  return filtered
})

// Kleine/schnelle Modelle für Query-Optimierung (1B-7B Parameter)
const smallModels = computed(() => {
  const smallPatterns = [
    /llama.*[1-3]b/i,
    /qwen.*[1-3]b/i,
    /phi.*[1-3]/i,
    /gemma.*2b/i,
    /tinyllama/i,
    /smollm/i,
    /mistral.*7b/i,
    /llama.*7b/i,
    /qwen.*7b/i,
  ]

  return availableModels.value
    .map(m => m.name)
    .filter(name => smallPatterns.some(pattern => pattern.test(name)))
    .sort((a, b) => {
      // Sortiere nach Parametergröße (kleinste zuerst)
      const sizeA = parseInt(a.match(/(\d+)b/i)?.[1] || '99')
      const sizeB = parseInt(b.match(/(\d+)b/i)?.[1] || '99')
      return sizeA - sizeB
    })
})

// Check if at least one option is selected
const hasAnySelection = computed(() => {
  return Object.values(resetSelection.value).some(val => val === true)
})

// Load model selection settings and available models on mount
onMounted(async () => {
  // Initialize sampling params from settings store
  samplingParams.value = {
    maxTokens: settingsStore.settings.maxTokens || 512,
    temperature: settingsStore.settings.temperature || 0.7,
    topP: settingsStore.settings.topP || 0.9,
    topK: settingsStore.settings.topK || 40,
    minP: settingsStore.settings.minP || 0.05,
    repeatPenalty: settingsStore.settings.repeatPenalty || 1.18,
    repeatLastN: settingsStore.settings.repeatLastN || 64,
    presencePenalty: settingsStore.settings.presencePenalty || 0.0,
    frequencyPenalty: settingsStore.settings.frequencyPenalty || 0.0,
    mirostatMode: settingsStore.settings.mirostatMode || 0,
    mirostatTau: settingsStore.settings.mirostatTau || 5.0,
    mirostatEta: settingsStore.settings.mirostatEta || 0.1
  }

  // Set initial tab if provided
  if (props.initialTab) {
    activeTab.value = props.initialTab
  }

  await loadModelSelectionSettings()
  await loadAvailableModels()
  await loadMateModels()
  await loadSystemPrompts()
  await loadTrustedMatesCount()
  await loadWebSearchSettings()
  await loadFileSearchStatus()
  await loadVoiceAssistantSettings()
  await loadTesseractStatus()
})

// Schriftgröße setzen und persistieren (stufenlos)
function setFontSize(size) {
  // Konvertiere alte String-Werte zu Zahlen
  if (typeof size === 'string') {
    const sizeMap = { small: 85, medium: 100, large: 115, xlarge: 130 }
    size = sizeMap[size] || 100
  }
  settings.value.fontSize = size
  settingsStore.settings.fontSize = size
  // CSS Custom Property anwenden
  applyFontSize(size)
}

// Schriftgröße auf das root-Element anwenden (stufenlos via CSS Variable)
function applyFontSize(size) {
  // Konvertiere alte String-Werte zu Zahlen
  if (typeof size === 'string') {
    const sizeMap = { small: 85, medium: 100, large: 115, xlarge: 130 }
    size = sizeMap[size] || 100
  }
  const root = document.documentElement
  // Alte Klassen entfernen (für Kompatibilität)
  root.classList.remove('font-size-small', 'font-size-medium', 'font-size-large', 'font-size-xlarge')
  // CSS Custom Property setzen (stufenlos)
  root.style.setProperty('--font-scale', size / 100)
  root.style.fontSize = `${size}%`
}

async function loadModelSelectionSettings() {
  try {
    const loadedSettings = await api.getModelSelectionSettings()
    modelSelectionSettings.value = loadedSettings
  } catch (error) {
    console.error('Failed to load model selection settings:', error)
  }
}

async function loadAvailableModels() {
  try {
    const response = await api.getAvailableModels()
    // API gibt {current_model, models: [...]} zurück
    // Konvertiere zu Array von {name, size} Objekten
    const modelList = response.models || response || []
    if (Array.isArray(modelList)) {
      availableModels.value = modelList.map(m => {
        // Wenn es bereits ein Objekt mit name ist
        if (typeof m === 'object' && m.name) {
          return m
        }
        // Wenn es nur ein String ist
        return { name: m, size: 0 }
      })
    } else {
      availableModels.value = []
    }
    console.log('📦 Loaded', availableModels.value.length, 'models')
  } catch (error) {
    console.error('Failed to load available models:', error)
    availableModels.value = []
  }
}

// Fleet Mates Model Functions
async function loadMateModels() {
  try {
    const [emailRes, docRes, logRes, coderRes] = await Promise.all([
      fetch('/api/settings/email-model'),
      fetch('/api/settings/document-model'),
      fetch('/api/settings/log-analysis-model'),
      fetch('/api/settings/coder-model')
    ])

    mateModels.value.emailModel = emailRes.ok ? await emailRes.text() : ''
    mateModels.value.documentModel = docRes.ok ? await docRes.text() : ''
    mateModels.value.logAnalysisModel = logRes.ok ? await logRes.text() : ''
    mateModels.value.coderModel = coderRes.ok ? await coderRes.text() : ''
  } catch (error) {
    console.error('Failed to load mate models:', error)
  }
}

// Einzelne Modell-Speicher-Funktionen (um Race Conditions zu vermeiden)
async function saveEmailModel(model = null) {
  if (model !== null) mateModels.value.emailModel = model
  try {
    await fetch('/api/settings/email-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.emailModel
    })
    success('Email-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save email model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveDocumentModel(model = null) {
  if (model !== null) mateModels.value.documentModel = model
  try {
    await fetch('/api/settings/document-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.documentModel
    })
    success('Dokument-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save document model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveLogAnalysisModel(model = null) {
  if (model !== null) mateModels.value.logAnalysisModel = model
  try {
    await fetch('/api/settings/log-analysis-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.logAnalysisModel
    })
    success('Log-Analyse-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save log analysis model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveCoderModel(model = null) {
  if (model !== null) mateModels.value.coderModel = model
  try {
    await fetch('/api/settings/coder-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.coderModel
    })
    success('Coder-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save coder model:', error)
    errorToast('Fehler beim Speichern')
  }
}

// Fleet Mates Functions
async function loadTrustedMatesCount() {
  try {
    const response = await fetch('/api/pairing/trusted')
    if (response.ok) {
      const mates = await response.json()
      trustedMates.value = mates
      trustedMatesCount.value = mates.length
    }
  } catch (error) {
    console.error('Failed to load trusted mates count:', error)
  }
}

async function forgetAllMates() {
  const confirmed = await confirm({
    title: 'Alle Mates vergessen?',
    message: 'Wirklich ALLE gepairten Mates vergessen? Diese müssen danach erneut gepairt werden.',
    type: 'danger',
    confirmText: 'Alle vergessen'
  })
  if (!confirmed) return

  forgettingMates.value = true
  try {
    const response = await secureFetch('/api/pairing/trusted', { method: 'DELETE' })
    if (response.ok) {
      trustedMates.value = []
      trustedMatesCount.value = 0
      success('Alle Mates wurden vergessen!')
    } else {
      throw new Error('Failed to forget mates')
    }
  } catch (err) {
    console.error('Failed to forget all mates:', err)
    errorToast('Fehler beim Vergessen der Mates')
  } finally {
    forgettingMates.value = false
  }
}

async function removeTrustedMate(mateId) {
  const confirmed = await confirm({
    title: 'Mate vergessen?',
    message: 'Dieser Mate muss danach erneut gepairt werden.',
    type: 'warning',
    confirmText: 'Vergessen'
  })
  if (!confirmed) return

  removingMateId.value = mateId
  try {
    const response = await secureFetch(`/api/pairing/trusted/${mateId}`, { method: 'DELETE' })
    if (response.ok) {
      await loadTrustedMatesCount()
      success('Mate wurde vergessen!')
    } else {
      throw new Error('Failed to remove mate')
    }
  } catch (err) {
    console.error('Failed to remove mate:', err)
    errorToast('Fehler beim Entfernen des Mates')
  } finally {
    removingMateId.value = null
  }
}

// Pending Pairing Functions
async function loadPendingPairingRequests() {
  try {
    const response = await fetch('/api/pairing/pending')
    if (response.ok) {
      pendingPairingRequests.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load pending pairing requests:', error)
  }
}

async function approvePairingRequest(requestId) {
  processingPairing.value = true
  try {
    const response = await secureFetch(`/api/pairing/approve/${requestId}`, { method: 'POST' })
    if (response.ok) {
      pendingPairingRequests.value = pendingPairingRequests.value.filter(r => r.requestId !== requestId)
      await loadTrustedMatesCount()
      success('Mate erfolgreich verbunden!')
    } else {
      throw new Error('Failed to approve pairing')
    }
  } catch (error) {
    console.error('Failed to approve pairing:', error)
    errorToast('Fehler beim Genehmigen des Pairings')
  } finally {
    processingPairing.value = false
  }
}

async function rejectPairingRequest(requestId) {
  processingPairing.value = true
  try {
    const response = await secureFetch(`/api/pairing/reject/${requestId}`, { method: 'POST' })
    if (response.ok) {
      pendingPairingRequests.value = pendingPairingRequests.value.filter(r => r.requestId !== requestId)
      success('Pairing-Anfrage abgelehnt')
    }
  } catch (error) {
    console.error('Failed to reject pairing:', error)
  } finally {
    processingPairing.value = false
  }
}

// Start polling for pending pairings when mates tab is active
function startPairingPoll() {
  if (pairingPollInterval) return
  loadPendingPairingRequests()
  pairingPollInterval = setInterval(loadPendingPairingRequests, 3000)
}

function stopPairingPoll() {
  if (pairingPollInterval) {
    clearInterval(pairingPollInterval)
    pairingPollInterval = null
  }
}

function getMateTypeName(mateType) {
  const names = {
    'mail': 'Email Client',
    'os': 'Betriebssystem',
    'browser': 'Browser',
    'office': 'Office Suite'
  }
  return names[mateType] || mateType || 'Unbekannt'
}

// Watch for changes from store
watch(() => settingsStore.settings, (newSettings) => {
  settings.value = { ...newSettings }
}, { deep: true })

function close() {
  stopPairingPoll()
  emit('close')
}

async function save() {
  saving.value = true
  try {
    // Merge sampling params into settings before saving
    const mergedSettings = {
      ...settings.value,
      ...samplingParams.value
    }

    // Save general settings + sampling parameters
    settingsStore.updateSettings(mergedSettings)

    // Apply streaming setting to chatStore
    if (chatStore.streamingEnabled !== mergedSettings.streamingEnabled) {
      chatStore.toggleStreaming()
    }

    // Save model selection settings to backend
    await api.updateModelSelectionSettings(modelSelectionSettings.value)

    // Save web search settings
    await saveWebSearchSettings()

    // Save personal info if on that tab
    if (personalInfoTabRef.value && activeTab.value === 'personal') {
      await personalInfoTabRef.value.save()
    }

    success('Einstellungen gespeichert')
    emit('save')
    close()
  } catch (error) {
    console.error('Failed to save settings:', error)
    errorToast('Fehler beim Speichern der Einstellungen')
  } finally {
    saving.value = false
  }
}

async function handleResetAll() {
  // Build list of selected categories
  const selectedCategories = []
  if (resetSelection.value.chats) selectedCategories.push('Chats & Nachrichten')
  if (resetSelection.value.projects) selectedCategories.push('Projekte & Dateien')
  if (resetSelection.value.customModels) selectedCategories.push('Custom Models')
  if (resetSelection.value.settings) selectedCategories.push('Einstellungen')
  if (resetSelection.value.personalInfo) selectedCategories.push('Persönliche Informationen')
  if (resetSelection.value.templates) selectedCategories.push('Templates & Prompts')
  if (resetSelection.value.stats) selectedCategories.push('Statistiken')

  if (selectedCategories.length === 0) {
    errorToast('Bitte wählen Sie mindestens eine Kategorie aus!')
    return
  }

  // Confirmation with selected categories
  const confirmation1 = confirm(
    '⚠️ ACHTUNG! ⚠️\n\n' +
    'Sie sind dabei, folgende Daten zu löschen:\n\n' +
    selectedCategories.map(cat => `• ${cat}`).join('\n') + '\n\n' +
    'Diese Aktion kann NICHT rückgängig gemacht werden!\n\n' +
    'Möchten Sie wirklich fortfahren?'
  )

  if (!confirmation1) return

  const confirmation2 = confirm(
    '⚠️ LETZTE WARNUNG! ⚠️\n\n' +
    'Dies ist Ihre letzte Chance!\n\n' +
    'Die ausgewählten Daten werden unwiderruflich gelöscht.\n' +
    'Die Anwendung wird danach neu geladen.\n\n' +
    'Sind Sie ABSOLUT SICHER?'
  )

  if (!confirmation2) return

  resetting.value = true

  try {
    // Call backend to delete selected data
    await api.resetSelectedData(resetSelection.value)

    // Show success message
    success('Ausgewählte Daten wurden gelöscht. Die Anwendung wird jetzt neu geladen...')

    // Wait a moment to show the message
    await new Promise(resolve => setTimeout(resolve, 1500))

    // Reload the page to reset to initial state
    window.location.reload()
  } catch (error) {
    console.error('Failed to reset data:', error)
    errorToast('Fehler beim Löschen der Daten: ' + (error.message || 'Unbekannter Fehler'))
    resetting.value = false
  }
}

async function resetToDefaults() {
  const confirmed = await confirm({
    title: 'Einstellungen zurücksetzen?',
    message: 'Alle Einstellungen auf Standard zurücksetzen?',
    type: 'warning',
    confirmText: 'Zurücksetzen'
  })
  if (confirmed) {
    settingsStore.resetToDefaults()
    settings.value = { ...settingsStore.settings }
    success('Einstellungen zurückgesetzt')
  }
}

// System Prompts Functions
async function loadSystemPrompts() {
  try {
    const prompts = await api.getAllSystemPrompts()
    systemPrompts.value = prompts
  } catch (error) {
    console.error('Failed to load system prompts:', error)
    errorToast('Fehler beim Laden der System-Prompts')
  }
}

function resetPromptForm() {
  promptForm.value = {
    name: '',
    content: '',
    isDefault: false
  }
}

function editSystemPrompt(prompt) {
  editingPrompt.value = prompt
  promptForm.value = {
    name: prompt.name,
    content: prompt.content,
    isDefault: prompt.isDefault || false
  }
  showPromptEditor.value = true
}

async function saveSystemPrompt() {
  try {
    if (!promptForm.value.name.trim() || !promptForm.value.content.trim()) {
      errorToast('Name und Inhalt dürfen nicht leer sein')
      return
    }

    if (editingPrompt.value) {
      // Update existing prompt
      await api.updateSystemPrompt(editingPrompt.value.id, promptForm.value)
      success('System-Prompt erfolgreich aktualisiert!')
    } else {
      // Create new prompt
      await api.createSystemPrompt(promptForm.value)
      success('System-Prompt erfolgreich erstellt!')
    }

    // Reload prompts and close editor
    await loadSystemPrompts()
    showPromptEditor.value = false
    editingPrompt.value = null
    resetPromptForm()
  } catch (error) {
    console.error('Failed to save system prompt:', error)
    errorToast('Fehler beim Speichern des System-Prompts')
  }
}

async function deleteSystemPrompt(promptId) {
  const confirmed = await confirmDelete('System-Prompt', 'Diese Aktion kann nicht rückgängig gemacht werden.')
  if (!confirmed) return

  try {
    await api.deleteSystemPrompt(promptId)
    success('System-Prompt erfolgreich gelöscht!')
    await loadSystemPrompts()
  } catch (error) {
    console.error('Failed to delete system prompt:', error)
    errorToast('Fehler beim Löschen des System-Prompts')
  }
}

async function activateSystemPrompt(prompt) {
  // Wenn bereits aktiv, nichts tun
  if (prompt.isDefault) {
    return
  }

  try {
    // 1. Als Standard in DB speichern (neuer dedizierter Endpoint)
    await api.setDefaultSystemPrompt(prompt.id)

    // 2. chatStore aktualisieren (für TopBar und Chat-Anfragen)
    chatStore.systemPrompt = prompt.content
    chatStore.systemPromptTitle = prompt.name

    // 3. Liste neu laden (UI aktualisieren)
    await loadSystemPrompts()

    success(`"${prompt.name}" aktiviert!`)
    console.log(`✅ System-Prompt "${prompt.name}" aktiviert und in chatStore gesetzt`)
  } catch (error) {
    console.error('Failed to activate system prompt:', error)
    errorToast('Fehler beim Aktivieren des System-Prompts')
  }
}

// Web Search Functions
async function loadWebSearchSettings() {
  try {
    const response = await fetch('/api/search/settings')
    if (response.ok) {
      const data = await response.json()
      webSearchSettings.value = {
        braveApiKey: data.braveApiKey || '',
        braveConfigured: data.braveConfigured || false,
        searchCount: data.searchCount || 0,
        searchLimit: data.searchLimit || 2000,
        remainingSearches: data.remainingSearches || 2000,
        currentMonth: data.currentMonth || '',
        customSearxngInstance: data.customSearxngInstance || '',
        searxngInstances: data.searxngInstances?.length > 0 ? data.searxngInstances : [...defaultSearxngInstances],
        searxngTotalCount: data.searxngTotalCount || 0,
        searxngMonthCount: data.searxngMonthCount || 0,
        // Feature Flags
        queryOptimizationEnabled: data.queryOptimizationEnabled ?? true,
        contentScrapingEnabled: data.contentScrapingEnabled ?? true,
        multiQueryEnabled: data.multiQueryEnabled ?? false,
        reRankingEnabled: data.reRankingEnabled ?? true,
        queryOptimizationModel: data.queryOptimizationModel || 'llama3.2:3b',
        effectiveOptimizationModel: data.effectiveOptimizationModel || null,
        // UI Animation
        webSearchAnimation: data.webSearchAnimation || 'data-wave'
      }
    }
  } catch (error) {
    console.error('Failed to load web search settings:', error)
    // Fallback auf Defaults
    webSearchSettings.value.searxngInstances = [...defaultSearxngInstances]
  }
}

async function saveWebSearchSettings() {
  try {
    const response = await secureFetch('/api/search/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        braveApiKey: webSearchSettings.value.braveApiKey,
        customSearxngInstance: webSearchSettings.value.customSearxngInstance,
        searxngInstances: webSearchSettings.value.searxngInstances.filter(i => i && i.trim()),
        // Feature Flags
        queryOptimizationEnabled: webSearchSettings.value.queryOptimizationEnabled,
        contentScrapingEnabled: webSearchSettings.value.contentScrapingEnabled,
        multiQueryEnabled: webSearchSettings.value.multiQueryEnabled,
        reRankingEnabled: webSearchSettings.value.reRankingEnabled,
        queryOptimizationModel: webSearchSettings.value.queryOptimizationModel,
        // UI Animation
        webSearchAnimation: webSearchSettings.value.webSearchAnimation
      })
    })
    if (!response.ok) {
      throw new Error('Failed to save')
    }
    // Reload to get updated status
    await loadWebSearchSettings()
  } catch (error) {
    console.error('Failed to save web search settings:', error)
    throw error
  }
}

async function testBraveSearch() {
  testingSearch.value = true
  try {
    await saveWebSearchSettings()
    const response = await secureFetch('/api/search/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'test' })
    })
    const result = await response.json()
    if (result.success) {
      success(`✓ Suche funktioniert! ${result.resultCount} Ergebnisse via ${result.source}`)
      await loadWebSearchSettings()
    } else {
      errorToast(`✗ Suche fehlgeschlagen: ${result.error || 'Unbekannter Fehler'}`)
    }
  } catch (error) {
    errorToast('✗ Test fehlgeschlagen: ' + error.message)
  } finally {
    testingSearch.value = false
  }
}

async function testCustomSearxng() {
  if (!webSearchSettings.value.customSearxngInstance) return
  testingSearch.value = true
  try {
    await saveWebSearchSettings()
    const response = await secureFetch('/api/search/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'test' })
    })
    const result = await response.json()
    if (result.success) {
      success(`✓ SearXNG funktioniert! ${result.resultCount} Ergebnisse`)
    } else {
      errorToast(`✗ SearXNG nicht erreichbar: ${result.error || 'Unbekannter Fehler'}`)
    }
  } catch (error) {
    errorToast('✗ Test fehlgeschlagen: ' + error.message)
  } finally {
    testingSearch.value = false
  }
}

function addSearxngInstance() {
  webSearchSettings.value.searxngInstances.push('')
}

function removeSearxngInstance(index) {
  webSearchSettings.value.searxngInstances.splice(index, 1)
}

async function resetSearxngInstances() {
  const confirmed = await confirm({
    title: 'Instanzen zurücksetzen?',
    message: 'Fallback-Instanzen auf Standard zurücksetzen?',
    type: 'warning',
    confirmText: 'Zurücksetzen'
  })
  if (confirmed) {
    webSearchSettings.value.searxngInstances = [...defaultSearxngInstances]
    success('Instanzen zurückgesetzt')
  }
}
</script>

<style scoped>
/* Custom Scrollbar */
.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(156, 163, 175, 0.3);
  border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(156, 163, 175, 0.5);
}

/* Modal Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active > div,
.modal-leave-active > div {
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.modal-enter-from > div {
  transform: scale(0.9) translateY(-20px);
}

.modal-leave-to > div {
  transform: scale(0.9) translateY(20px);
}

/* Font Size Slider - High Contrast */
.font-size-slider {
  background: linear-gradient(to right, #3b82f6, #60a5fa);
  border: 2px solid #60a5fa;
}

:deep(.dark) .font-size-slider,
.dark .font-size-slider {
  background: linear-gradient(to right, #1e40af, #3b82f6);
  border: 2px solid #60a5fa;
}

/* Slider Thumb - WebKit (Chrome, Safari) */
.font-size-slider::-webkit-slider-thumb {
  appearance: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ffffff 0%, #e5e7eb 100%);
  border: 3px solid #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), 0 0 0 2px rgba(59, 130, 246, 0.3);
  transition: all 0.2s ease;
}

.font-size-slider::-webkit-slider-thumb:hover {
  transform: scale(1.15);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 0 4px rgba(59, 130, 246, 0.4);
}

/* Slider Thumb - Firefox */
.font-size-slider::-moz-range-thumb {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ffffff 0%, #e5e7eb 100%);
  border: 3px solid #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), 0 0 0 2px rgba(59, 130, 246, 0.3);
}

.font-size-slider::-moz-range-thumb:hover {
  transform: scale(1.15);
}

/* Slider Track - Firefox */
.font-size-slider::-moz-range-track {
  background: linear-gradient(to right, #3b82f6, #60a5fa);
  border-radius: 8px;
  height: 12px;
}
</style>
