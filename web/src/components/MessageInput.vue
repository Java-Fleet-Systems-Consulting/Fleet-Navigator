<template>
  <div :class="[
    'message-input-container relative',
    props.heroMode ? 'p-2 sm:p-4' : 'px-2 sm:px-4 py-3'
  ]">

    <div class="max-w-6xl mx-auto relative">
      <!-- Uploaded Files Display -->
      <TransitionGroup name="file" tag="div" class="mb-2 flex flex-wrap gap-2 max-h-24 overflow-y-auto custom-scrollbar">
        <div
          v-for="(file, index) in uploadedFiles"
          :key="index"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-gray-700/50 border border-gray-600/50 text-sm group"
        >
          <component :is="getFileIcon(file.type)" class="w-4 h-4 text-fleet-orange-400 flex-shrink-0" />
          <span class="max-w-[120px] truncate text-gray-300">{{ file.name }}</span>
          <button @click="removeFile(index)" class="p-0.5 rounded text-gray-500 hover:text-red-400 transition-colors">
            <XMarkIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </TransitionGroup>

      <!-- Error/Warning Messages -->
      <Transition name="fade">
        <div v-if="hasImages && !isVisionModel && !settingsStore.getSetting('autoSelectVisionModel')"
             class="mb-2 p-2 rounded-lg bg-yellow-900/30 border border-yellow-700/50 text-yellow-300 text-sm flex items-center gap-2">
          <ExclamationTriangleIcon class="w-4 h-4 flex-shrink-0" />
          <span>{{ t('messageInput.visionRecommended') }}</span>
        </div>
      </Transition>
      <Transition name="fade">
        <div v-if="errorMessage" class="mb-2 p-2 rounded-lg bg-red-900/30 border border-red-700/50 text-red-300 text-sm flex items-center gap-2">
          <XCircleIcon class="w-4 h-4 flex-shrink-0" />
          <span>{{ errorMessage }}</span>
        </div>
      </Transition>

      <!-- Main Input Tile - Seamless Style (transparent to match chat background) -->
      <div class="input-tile rounded-2xl bg-transparent border border-gray-300/30 dark:border-gray-700/50 overflow-hidden">
        <!-- Textarea Area -->
        <div class="p-5 pb-3">
          <textarea
            v-model="inputText"
            @keydown.enter.exact.prevent="handleSend"
            @keydown.shift.enter="handleNewLine"
            @input="adjustHeight"
            :placeholder="props.heroMode ? t('messageInput.heroPlaceholder') : t('messageInput.placeholder')"
            rows="1"
            class="
              w-full bg-transparent
              text-gray-800 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500
              focus:outline-none
              resize-none
              text-base
            "
            :style="{ minHeight: props.heroMode ? '60px' : '24px', maxHeight: '200px' }"
            :disabled="chatStore.isLoading"
            ref="textareaRef"
          ></textarea>
        </div>

        <!-- Bottom Bar with Icons and Buttons -->
        <div class="px-3 pb-3 flex items-center justify-between">
          <!-- Left Side Icons -->
          <div class="flex items-center gap-1">
            <!-- File Upload -->
            <button
              @click="triggerFileInput"
              class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
              :disabled="chatStore.isLoading"
              :title="t('messageInput.attachFile')"
            >
              <PlusIcon class="w-5 h-5" />
            </button>
            <input
              ref="fileInput"
              type="file"
              @change="handleFileSelect"
              accept=".pdf,.txt,.md,.html,.json,.xml,.csv,.png,.jpg,.jpeg,.webp,.bmp,.gif,.tiff,.tif"
              multiple
              class="hidden"
            />

            <!-- Web Search Toggle -->
            <button
              @click="webSearchEnabled = !webSearchEnabled"
              :class="[
                'p-2 rounded-lg transition-all',
                webSearchEnabled
                  ? 'text-blue-400 bg-blue-500/20 hover:bg-blue-500/30'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'
              ]"
              :disabled="chatStore.isLoading"
              :title="webSearchEnabled ? t('messageInput.webSearchActive') : t('messageInput.webSearchEnable')"
            >
              <GlobeAltIcon class="w-5 h-5" />
            </button>

            <!-- Voice Input (Microphone) -->
            <button
              @click="toggleRecording"
              :class="[
                'p-2 rounded-lg transition-all',
                isRecording
                  ? 'text-red-400 bg-red-500/20 hover:bg-red-500/30 animate-pulse'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-700/50'
              ]"
              :disabled="chatStore.isLoading || isTranscribing"
              :title="isRecording ? t('messageInput.stopRecording') : t('messageInput.startRecording')"
            >
              <MicrophoneIcon class="w-5 h-5" />
            </button>

            <!-- Hardware Status (kompakt, klickbar → öffnet SystemMonitor) -->
            <button
              v-if="hardwareStats"
              @click="toggleSystemMonitor"
              class="hidden sm:flex items-center gap-1.5 px-2 py-1 rounded-lg bg-gray-200/30 dark:bg-gray-700/30 hover:bg-gray-300/50 dark:hover:bg-gray-600/50 text-xs transition-all cursor-pointer"
              :title="`CPU: ${hardwareStats.cpu?.toFixed(1) || 0}% | GPU: ${hardwareStats.gpuTemp?.toFixed(0) || '-'}°C | RAM: ${hardwareStats.ram?.toFixed(1) || 0}% — Klicken für Details`"
            >
              <!-- CPU -->
              <span class="flex items-center gap-0.5" :class="getCpuColor(hardwareStats.cpu)">
                <CpuChipIcon class="w-3.5 h-3.5" />
                <span>{{ hardwareStats.cpu?.toFixed(0) || 0 }}%</span>
              </span>
              <!-- GPU Temp (wenn vorhanden) -->
              <span v-if="hardwareStats.gpuTemp" class="flex items-center gap-0.5" :class="getTempColor(hardwareStats.gpuTemp)">
                <FireIcon class="w-3.5 h-3.5" />
                <span>{{ hardwareStats.gpuTemp?.toFixed(0) }}°</span>
              </span>
            </button>

            <!-- Model Manager Button -->
            <button
              @click="openModelManager"
              class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
              :title="t('messageInput.openModelManager')"
            >
              <Square3Stack3DIcon class="w-5 h-5" />
            </button>

            <!-- Settings Button -->
            <button
              @click="openSettings"
              class="p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
              :title="t('messageInput.openSettings')"
            >
              <Cog6ToothIcon class="w-5 h-5" />
            </button>
          </div>

          <!-- Right Side - Expert/Model Selector & Send -->
          <div class="flex items-center gap-2">
            <!-- Expert/Model Dropdown -->
            <div class="hidden sm:block relative" ref="dropdownContainer">
              <button
                @click="toggleExpertDropdown"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-gray-600 dark:text-gray-300 text-sm hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
                :disabled="chatStore.isLoading"
              >
                <span v-if="chatStore.selectedExpertId" class="text-purple-400">🎓</span>
                <CpuChipIcon v-else class="w-4 h-4 text-gray-500" />
                <span class="truncate max-w-[120px]">{{ displayModelName }}</span>
                <ChevronDownIcon class="w-4 h-4" :class="showExpertDropdown ? 'rotate-180' : ''" />
              </button>

              <!-- Dropdown Menu - fixed positioning to avoid clipping -->
              <Transition name="dropdown">
                <div
                  v-if="showExpertDropdown"
                  class="fixed right-[calc(50%-8rem)] bottom-28 w-64 max-h-80 overflow-y-auto bg-gray-800 border border-gray-700 rounded-xl shadow-xl z-[9999]"
                  style="transform: translateX(50%);"
                >
                  <!-- Experten Section -->
                  <div v-if="experts && experts.length > 0" class="p-2">
                    <div class="text-xs text-gray-500 px-2 py-1 font-semibold">🎓 Experten</div>
                    <button
                      v-for="expert in experts"
                      :key="expert.id"
                      @click="selectExpert(expert)"
                      class="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left hover:bg-gray-700/70 transition-all"
                      :class="chatStore.selectedExpertId === expert.id ? 'bg-purple-500/20 text-purple-300' : 'text-gray-300'"
                    >
                      <img
                        v-if="expert.avatarUrl"
                        :src="expert.avatarUrl"
                        class="w-6 h-6 rounded-md object-cover"
                        alt=""
                      />
                      <div v-else class="w-6 h-6 rounded-md bg-gray-600 flex items-center justify-center text-xs">
                        {{ expert.name?.charAt(0) || '?' }}
                      </div>
                      <div class="flex-1 min-w-0">
                        <div class="text-sm font-medium truncate">{{ expert.name }}</div>
                        <div class="text-xs text-gray-500 truncate">{{ expert.role }}</div>
                      </div>
                      <CheckIcon v-if="chatStore.selectedExpertId === expert.id" class="w-4 h-4 text-purple-400" />
                    </button>
                  </div>

                  <!-- Hinweis wenn keine Experten -->
                  <div v-if="!experts?.length" class="p-3 text-center text-gray-400 text-sm">
                    {{ t('messageInput.noExperts') }}<br>
                    <span class="text-xs">{{ t('messageInput.createInManager') }}</span>
                  </div>
                </div>
              </Transition>
            </div>

            <!-- Theme Dropdown (kompakt) - fixed position für Sichtbarkeit -->
            <div class="hidden sm:block relative" ref="themeDropdownContainer">
              <button
                @click="showThemeDropdown = !showThemeDropdown"
                class="flex items-center gap-1 px-2 py-1.5 rounded-lg text-gray-500 dark:text-gray-400 text-xs hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
                :title="t('messageInput.themeSelect')"
              >
                <SwatchIcon class="w-4 h-4" />
                <span class="hidden md:inline truncate max-w-[60px]">{{ currentThemeLabel }}</span>
                <ChevronDownIcon class="w-3 h-3" :class="showThemeDropdown ? 'rotate-180' : ''" />
              </button>

              <!-- Theme Dropdown Menu - nur 3 Basis-Themes (hell/dunkel via Toggle) -->
              <Teleport to="body">
                <Transition name="dropdown">
                  <div
                    v-if="showThemeDropdown"
                    class="fixed w-36 bg-gray-800 border border-gray-700 rounded-xl shadow-2xl z-[99999]"
                    :style="themeDropdownStyle"
                  >
                    <div class="p-1.5">
                      <button @click="setThemeBase('tech')" class="w-full flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-gray-700 text-left text-sm" :class="currentThemeBase === 'tech' ? 'bg-cyan-900/30 text-cyan-300' : 'text-gray-300'">
                        <div class="w-4 h-4 rounded-full bg-gradient-to-br from-[#00D9FF] to-[#00FF88]" style="box-shadow: 0 0 6px rgba(0, 217, 255, 0.5);"></div>
                        <span>Tech</span>
                      </button>
                      <button @click="setThemeBase('crazy')" class="w-full flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-gray-700 text-left text-sm" :class="currentThemeBase === 'crazy' ? 'bg-pink-900/30 text-pink-300' : 'text-gray-300'">
                        <div class="w-4 h-4 rounded-full bg-gradient-to-br from-[#FF0D57] to-[#6A0dad]"></div>
                        <span>Crazy</span>
                      </button>
                      <button @click="setThemeBase('lawyer')" class="w-full flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-gray-700 text-left text-sm" :class="currentThemeBase === 'lawyer' ? 'bg-blue-900/30 text-blue-300' : 'text-gray-300'">
                        <div class="w-4 h-4 rounded-full bg-gradient-to-br from-gray-400 to-gray-700 border border-blue-500"></div>
                        <span>Anwalt</span>
                      </button>
                    </div>
                  </div>
                </Transition>
              </Teleport>
            </div>

            <!-- Hell/Dunkel Toggle - wechselt zwischen dark/light Variante des aktuellen Themes -->
            <button
              @click="toggleThemeVariant"
              class="hidden sm:flex p-2 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-200/50 dark:hover:bg-gray-700/50 transition-all"
              :title="isLightTheme ? t('messageInput.darkMode') : t('messageInput.lightMode')"
            >
              <SunIcon v-if="!isLightTheme" class="w-4 h-4 text-amber-400" />
              <MoonIcon v-else class="w-4 h-4 text-indigo-400" />
            </button>

            <!-- Stop Button -->
            <button
              v-if="chatStore.isLoading"
              @click="handleStop"
              class="p-2.5 rounded-xl bg-red-500 hover:bg-red-400 text-white transition-all"
              :title="t('messageInput.stop')"
            >
              <StopIcon class="w-5 h-5" />
            </button>

            <!-- Send Button -->
            <button
              v-else
              @click="handleSend"
              :disabled="!inputText.trim()"
              :class="[
                'send-button p-2.5 rounded-xl transition-all',
                inputText.trim()
                  ? 'bg-fleet-orange-500 hover:bg-fleet-orange-400 text-white'
                  : 'bg-gray-700 text-gray-500 cursor-not-allowed'
              ]"
              :title="t('messageInput.send')"
            >
              <ArrowUpIcon class="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, inject, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  PlusIcon,
  XMarkIcon,
  StopIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
  DocumentTextIcon,
  DocumentIcon,
  PhotoIcon,
  GlobeAltIcon,
  Cog6ToothIcon,
  ChevronDownIcon,
  ArrowUpIcon,
  CheckIcon,
  CpuChipIcon,
  FireIcon,
  SwatchIcon,
  SunIcon,
  MoonIcon,
  Square3Stack3DIcon,
  MicrophoneIcon
} from '@heroicons/vue/24/outline'
import { useChatStore } from '../stores/chatStore'
import { useSettingsStore } from '../stores/settingsStore'
import { storeToRefs } from 'pinia'
import { useToast } from '../composables/useToast'
import api from '../services/api'

const { t } = useI18n()
const { success, error: errorToast } = useToast()
const emit = defineEmits(['send'])
const props = defineProps({
  heroMode: {
    type: Boolean,
    default: false
  }
})
const chatStore = useChatStore()
const settingsStore = useSettingsStore()

// Destructure reactive refs from store
const { experts, models } = storeToRefs(chatStore)

const inputText = ref('')
const textareaRef = ref(null)
const fileInput = ref(null)
const uploadedFiles = ref([])
const isUploading = ref(false)
const uploadProgress = ref('')
const errorMessage = ref('')
const webSearchEnabled = ref(false)

// Voice Recording State
const isRecording = ref(false)
const isTranscribing = ref(false)
const wasVoiceInput = ref(false)  // Flag für Auto-TTS nach Spracheingabe
let mediaRecorder = null
let audioChunks = []

// Silence Detection für automatisches Stoppen
let audioContext = null
let analyser = null
let silenceTimeout = null
const SILENCE_THRESHOLD = 15      // Lautstärke-Schwellwert (0-255)
const SILENCE_DURATION = 500      // 0.5 Sekunden Stille zum Auto-Stopp
const MAX_RECORDING_TIME = 60000  // Max 60 Sekunden Aufnahme

// Hardware Stats (kompakt für Input-Bereich)
const hardwareStats = ref(null)
let hardwareInterval = null

// Expert/Model Dropdown
const showExpertDropdown = ref(false)
const dropdownContainer = ref(null)

// Theme Dropdown
const showThemeDropdown = ref(false)
const themeDropdownContainer = ref(null)

// Dark Mode (von MainLayout injected)
const darkMode = inject('darkMode', ref(true))

// Get showAbortModal from App.vue
const showAbortModal = inject('showAbortModal')
const openSettings = inject('openSettings', () => {})
const toggleSystemMonitor = inject('toggleSystemMonitor', () => {})
const openModelManager = inject('openModelManager', () => {})

// Available models (use storeToRefs 'models' directly)

// Computed properties
const hasImages = computed(() => uploadedFiles.value.some(f => f.type === 'image'))
const isVisionModel = computed(() => settingsStore.isVisionModel(chatStore.selectedModel))

// Theme computed
const currentTheme = computed(() => settingsStore.settings.uiTheme || 'tech-dark')
const currentThemeLabel = computed(() => {
  const theme = currentTheme.value
  if (theme === 'tech-dark' || theme === 'default' || !theme) return 'Tech Dark'
  if (theme === 'tech-light') return 'Tech Hell'
  if (theme === 'crazy-light') return 'Crazy Hell'
  if (theme === 'crazy-dark') return 'Crazy'
  if (theme === 'lawyer-light') return 'Anwalt'
  if (theme === 'lawyer-dark') return 'Anwalt'
  return 'Theme'
})

// Prüft ob aktuelles Theme eine "light" Variante ist
const isLightTheme = computed(() => {
  const theme = currentTheme.value
  return theme?.includes('light')
})

// Basis-Theme ohne dark/light Suffix (für Dropdown-Auswahl)
const currentThemeBase = computed(() => {
  const theme = currentTheme.value || 'tech-dark'
  if (theme.startsWith('tech')) return 'tech'
  if (theme.startsWith('crazy')) return 'crazy'
  if (theme.startsWith('lawyer')) return 'lawyer'
  return 'tech'
})

// Dropdown Position berechnen (über dem Button)
const themeDropdownStyle = computed(() => {
  if (!themeDropdownContainer.value) return {}
  const rect = themeDropdownContainer.value.getBoundingClientRect()
  return {
    bottom: `${window.innerHeight - rect.top + 8}px`,
    right: `${window.innerWidth - rect.right}px`
  }
})
// Display expert name or model name
const displayModelName = computed(() => {
  // If expert is selected, show expert name (ohne Emoji, das ist jetzt im Template)
  if (chatStore.selectedExpertId) {
    const expert = chatStore.getExpertById(chatStore.selectedExpertId)
    if (expert) {
      const name = expert.name // z.B. "Roland Navarro"
      if (name.length > 15) {
        return name.substring(0, 12) + '...'
      }
      return name
    }
  }

  // Fallback to model name
  const model = chatStore.selectedModel || 'Kein Model'
  if (model.length > 15) {
    return model.substring(0, 12) + '...'
  }
  return model
})

// Get file icon component
function getFileIcon(type) {
  if (type === 'image') return PhotoIcon
  if (type === 'pdf') return DocumentIcon
  return DocumentTextIcon
}

// Get model name from model object or string
function getModelName(model) {
  if (typeof model === 'string') return model
  if (model && model.name) return model.name
  return String(model)
}

// Expert/Model Dropdown Functions
function toggleExpertDropdown() {
  showExpertDropdown.value = !showExpertDropdown.value
}

function selectExpert(expert) {
  chatStore.selectExpert(expert)
  showExpertDropdown.value = false
  // Update current chat's expert if chat exists
  if (chatStore.currentChat) {
    chatStore.currentChat.expertId = expert.id
    chatStore.currentChat.expertName = expert.name
  }
}

function selectModel(model) {
  chatStore.setSelectedModel(model)
  showExpertDropdown.value = false
}

// Theme Funktionen
function setTheme(theme) {
  settingsStore.settings.uiTheme = theme
  settingsStore.saveUiThemeToBackend(theme)
  showThemeDropdown.value = false
}

// Setzt Basis-Theme und behält hell/dunkel bei
function setThemeBase(base) {
  const suffix = isLightTheme.value ? '-light' : '-dark'
  setTheme(base + suffix)
}

// Wechselt zwischen dark/light Variante des aktuellen Themes
function toggleThemeVariant() {
  const theme = currentTheme.value || 'tech-dark'
  let newTheme

  if (theme.includes('light')) {
    // Von light zu dark wechseln
    newTheme = theme.replace('-light', '-dark')
  } else if (theme.includes('dark')) {
    // Von dark zu light wechseln
    newTheme = theme.replace('-dark', '-light')
  } else {
    // Fallback: tech-dark zu tech-light
    newTheme = 'tech-light'
  }

  setTheme(newTheme)
}

// Close dropdown when clicking outside
function handleClickOutside(event) {
  if (dropdownContainer.value && !dropdownContainer.value.contains(event.target)) {
    showExpertDropdown.value = false
  }
  if (themeDropdownContainer.value && !themeDropdownContainer.value.contains(event.target)) {
    showThemeDropdown.value = false
  }
}

// Hardware Stats laden
async function loadHardwareStats() {
  try {
    const response = await api.getSystemStats()
    if (response) {
      hardwareStats.value = {
        cpu: response.cpu?.usage_percent || 0,
        ram: response.memory?.used_percent || 0,
        gpuTemp: response.gpu?.[0]?.temperature || null
      }
    }
  } catch (error) {
    // Silent fail - Hardware-Stats sind optional
  }
}

// CPU Farbe basierend auf Auslastung
function getCpuColor(usage) {
  if (!usage) return 'text-gray-400'
  if (usage < 50) return 'text-green-500'
  if (usage < 80) return 'text-yellow-500'
  return 'text-red-500'
}

// Temperatur Farbe
function getTempColor(temp) {
  if (!temp) return 'text-gray-400'
  if (temp < 60) return 'text-green-500'
  if (temp < 80) return 'text-yellow-500'
  return 'text-red-500'
}

// Lifecycle hooks for click-outside handler
onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  // Hardware-Stats laden und alle 5 Sekunden aktualisieren
  loadHardwareStats()
  hardwareInterval = setInterval(loadHardwareStats, 5000)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  if (hardwareInterval) {
    clearInterval(hardwareInterval)
  }
})

// File Upload Functions
function triggerFileInput() {
  fileInput.value?.click()
}

async function handleFileSelect(event) {
  const files = Array.from(event.target.files)
  await processFiles(files)
  event.target.value = ''
}

async function processFiles(files) {
  errorMessage.value = ''

  for (const file of files) {
    const allowedTypes = [
      'application/pdf',
      'text/plain', 'text/markdown', 'text/html', 'text/csv',
      'application/json', 'application/xml', 'text/xml',
      'image/png', 'image/jpeg', 'image/webp', 'image/bmp', 'image/gif', 'image/tiff'
    ]
    const allowedExtensions = [
      '.pdf', '.txt', '.md', '.html', '.htm', '.json', '.xml', '.csv',
      '.png', '.jpg', '.jpeg', '.webp', '.bmp', '.gif', '.tiff', '.tif'
    ]

    const isValidType = allowedTypes.includes(file.type) ||
                       allowedExtensions.some(ext => file.name.toLowerCase().endsWith(ext))

    if (!isValidType) {
      errorMessage.value = t('fileUpload.unsupported', { name: file.name })
      errorToast(t('messageInput.typeNotSupported'))
      continue
    }

    if (file.size > 50 * 1024 * 1024) {
      errorMessage.value = t('fileUpload.tooLarge', { name: file.name })
      errorToast(t('messageInput.fileTooLarge'))
      continue
    }

    await uploadFile(file)
  }
}

async function uploadFile(file) {
  isUploading.value = true
  uploadProgress.value = file.name

  try {
    const response = await api.uploadFile(file)

    if (response.success) {
      if (response.type === 'scanned-pdf' && response.pageImages && response.pageImages.length > 0) {
        for (let i = 0; i < response.pageImages.length; i++) {
          const pageFile = {
            name: `${response.filename} (S.${i + 1})`,
            type: 'image',
            textContent: null,
            base64Content: response.pageImages[i],
            size: response.pageImages[i].length
          }
          uploadedFiles.value.push(pageFile)
        }
        success(t('messageInput.pagesUploaded', { name: file.name, count: response.pageImages.length }))
      } else {
        const uploadedFile = {
          name: response.filename,
          type: response.type,
          textContent: response.textContent || null,
          base64Content: response.base64Content || null,
          size: response.size
        }
        uploadedFiles.value.push(uploadedFile)
        success(t('messageInput.fileUploaded', { name: file.name }))
      }
    } else {
      errorMessage.value = response.error || t('messageInput.uploadFailed')
      errorToast(t('messageInput.uploadFailed'))
    }
  } catch (error) {
    console.error('Upload error:', error)
    errorMessage.value = t('fileUpload.error', { message: error.message })
    errorToast(t('messageInput.uploadError'))
  } finally {
    isUploading.value = false
    uploadProgress.value = ''
  }
}

function removeFile(index) {
  uploadedFiles.value.splice(index, 1)
}

// Send Message Functions
function handleSend() {
  console.log('🚀 handleSend called, wasVoiceInput:', wasVoiceInput.value)
  if (!inputText.value.trim() || chatStore.isLoading) return

  const messageData = {
    text: inputText.value,
    files: uploadedFiles.value,
    webSearchEnabled: webSearchEnabled.value,
    wasVoiceInput: wasVoiceInput.value  // Flag für Auto-TTS
  }
  console.log('🚀 Emitting send with messageData:', messageData)

  emit('send', messageData)
  inputText.value = ''
  uploadedFiles.value = []
  errorMessage.value = ''
  wasVoiceInput.value = false  // Reset nach Senden

  if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
  }
}

function handleNewLine() {
  adjustHeight()
}

async function handleStop() {
  console.log('Stop button clicked')
  const result = await chatStore.abortCurrentRequest()

  if (result) {
    showAbortModal.value = true
    setTimeout(() => {
      showAbortModal.value = false
    }, 3000)
  }
}

function adjustHeight() {
  if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
    const newHeight = Math.min(textareaRef.value.scrollHeight, 200)
    textareaRef.value.style.height = newHeight + 'px'
  }
}

// Voice Recording Functions
async function toggleRecording() {
  if (isRecording.value) {
    stopRecording()
  } else {
    await startRecording()
  }
}

async function startRecording() {
  try {
    // Check if browser supports MediaRecorder
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
      errorToast(t('messageInput.noAudioSupport'))
      return
    }

    // Prüfe zuerst den Berechtigungsstatus (für bessere Fehlermeldung)
    if (navigator.permissions) {
      try {
        const permStatus = await navigator.permissions.query({ name: 'microphone' })
        console.log('🎤 Mikrofon-Berechtigung Status:', permStatus.state)
        if (permStatus.state === 'denied') {
          errorToast(t('messageInput.micBlocked'))
          return
        }
      } catch (permErr) {
        console.log('🎤 Permission API nicht verfügbar:', permErr)
      }
    }

    console.log('🎤 Fordere Mikrofon-Zugriff an...')
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    console.log('🎤 Mikrofon-Zugriff erhalten!')

    // Determine best supported format
    let mimeType = 'audio/webm'
    if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
      mimeType = 'audio/webm;codecs=opus'
    } else if (MediaRecorder.isTypeSupported('audio/ogg;codecs=opus')) {
      mimeType = 'audio/ogg;codecs=opus'
    } else if (MediaRecorder.isTypeSupported('audio/mp4')) {
      mimeType = 'audio/mp4'
    }

    mediaRecorder = new MediaRecorder(stream, { mimeType })
    audioChunks = []

    // Setup Silence Detection
    audioContext = new (window.AudioContext || window.webkitAudioContext)()
    analyser = audioContext.createAnalyser()
    const source = audioContext.createMediaStreamSource(stream)
    source.connect(analyser)
    analyser.fftSize = 256

    const dataArray = new Uint8Array(analyser.frequencyBinCount)
    let lastSoundTime = null  // null = noch nicht gesprochen
    let hasSpoken = false     // Hat der Benutzer angefangen zu sprechen?
    let maxRecordingTimer = null

    // Silence detection loop
    const checkSilence = () => {
      if (!isRecording.value) return

      analyser.getByteFrequencyData(dataArray)
      const average = dataArray.reduce((a, b) => a + b) / dataArray.length

      if (average > SILENCE_THRESHOLD) {
        // Sound detected
        if (!hasSpoken) {
          console.log('Sprache erkannt - Stille-Erkennung aktiviert')
          hasSpoken = true
        }
        lastSoundTime = Date.now()
      } else if (hasSpoken && lastSoundTime) {
        // Silence after speech - check if duration exceeded
        if (Date.now() - lastSoundTime > SILENCE_DURATION) {
          console.log('Auto-stop: Stille nach Sprache erkannt')
          stopRecording()
          return
        }
      }

      silenceTimeout = requestAnimationFrame(checkSilence)
    }

    mediaRecorder.ondataavailable = (event) => {
      if (event.data.size > 0) {
        audioChunks.push(event.data)
      }
    }

    mediaRecorder.onstop = async () => {
      console.log('Recording stopped, audioChunks:', audioChunks.length)

      // Cleanup silence detection
      if (silenceTimeout) {
        cancelAnimationFrame(silenceTimeout)
        silenceTimeout = null
      }
      if (maxRecordingTimer) {
        clearTimeout(maxRecordingTimer)
      }
      if (audioContext) {
        audioContext.close()
        audioContext = null
      }

      // Stop all tracks
      stream.getTracks().forEach(track => track.stop())

      if (audioChunks.length > 0) {
        const audioBlob = new Blob(audioChunks, { type: mimeType })
        console.log('Audio blob size:', audioBlob.size, 'bytes')
        await transcribeRecording(audioBlob)
      } else {
        console.log('No audio chunks collected!')
        errorToast(t('messageInput.noAudioData'))
      }
    }

    mediaRecorder.onerror = (event) => {
      console.error('MediaRecorder error:', event.error)
      errorToast(t('messageInput.recordingError', { message: event.error.message }))
      isRecording.value = false
    }

    mediaRecorder.start(100) // Collect data every 100ms
    isRecording.value = true
    success(t('messageInput.recordingStarted'))

    // Start silence detection
    checkSilence()

    // Max recording time safety
    maxRecordingTimer = setTimeout(() => {
      if (isRecording.value) {
        console.log('Auto-stop: Max Aufnahmezeit erreicht')
        stopRecording()
      }
    }, MAX_RECORDING_TIME)

  } catch (err) {
    console.error('🎤 Fehler bei Aufnahme:', err.name, err.message, err)
    if (err.name === 'NotAllowedError') {
      // Im Inkognito oder wenn vorher verweigert
      errorToast(t('messageInput.micDenied'))
    } else if (err.name === 'NotFoundError') {
      errorToast(t('messageInput.noMicFound'))
    } else if (err.name === 'NotReadableError') {
      errorToast(t('messageInput.micInUse'))
    } else if (err.name === 'OverconstrainedError') {
      errorToast(t('messageInput.recordingError', { message: 'OverconstrainedError' }))
    } else {
      errorToast(t('messageInput.recordingError', { message: err.message || err.name }))
    }
  }
}

function stopRecording() {
  // Cleanup silence detection
  if (silenceTimeout) {
    cancelAnimationFrame(silenceTimeout)
    silenceTimeout = null
  }

  if (mediaRecorder && mediaRecorder.state !== 'inactive') {
    mediaRecorder.stop()
    isRecording.value = false
  }
}

async function transcribeRecording(audioBlob) {
  isTranscribing.value = true

  try {
    // Determine format from blob type
    let format = 'webm'
    if (audioBlob.type.includes('ogg')) format = 'ogg'
    else if (audioBlob.type.includes('mp4')) format = 'mp4'
    else if (audioBlob.type.includes('wav')) format = 'wav'

    const result = await api.transcribeAudio(audioBlob, format)
    console.log('STT Result:', result)

    if (result.text) {
      // Append transcribed text to input
      const newText = inputText.value ? inputText.value + ' ' + result.text : result.text
      console.log('Setting inputText to:', newText)
      inputText.value = newText
      console.log('inputText.value is now:', inputText.value)

      // Flag setzen für Auto-TTS der Antwort
      wasVoiceInput.value = true

      success(t('messageInput.recognized', { text: result.text }))

      // Auto-Send nach kurzer Verzögerung (damit User den Text sieht)
      // Direkt den Text übergeben statt auf inputText.value zu vertrauen
      const textToSend = newText
      setTimeout(() => {
        adjustHeight()
        console.log('🎤 Auto-Send nach Spracheingabe, Text:', textToSend)
        console.log('🎤 inputText.value:', inputText.value)
        console.log('🎤 chatStore.isLoading:', chatStore.isLoading)

        // Sicherstellen dass der Text noch da ist
        if (!inputText.value.trim()) {
          console.log('🎤 inputText war leer, setze neu:', textToSend)
          inputText.value = textToSend
        }

        // Jetzt senden
        if (inputText.value.trim() && !chatStore.isLoading) {
          handleSend()
        } else {
          console.log('🎤 Senden nicht möglich - Text leer oder bereits ladend')
        }
      }, 800)  // 800ms Verzögerung für bessere Sichtbarkeit
    } else {
      console.log('No text in result:', result)
      errorToast(result.error || t('messageInput.noSpeechRecognized'))
    }
  } catch (err) {
    console.error('Transcription error:', err)
    errorToast(t('messageInput.transcriptionError'))
  } finally {
    isTranscribing.value = false
  }
}
</script>

<style scoped>
.input-tile {
  /* Kein Shadow - seamless mit Chat-Hintergrund */
}

/* File Transition */
.file-enter-active {
  animation: file-in 0.2s ease-out;
}
.file-leave-active {
  animation: file-out 0.15s ease-in;
}

@keyframes file-in {
  from { opacity: 0; transform: scale(0.9); }
  to { opacity: 1; transform: scale(1); }
}

@keyframes file-out {
  from { opacity: 1; transform: scale(1); }
  to { opacity: 0; transform: scale(0.9); }
}

/* Fade Transition */
.fade-enter-active, .fade-leave-active {
  transition: all 0.2s ease;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}

/* Custom Scrollbar */
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
  height: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(156, 163, 175, 0.3);
  border-radius: 2px;
}

/* Textarea no scrollbar */
textarea::-webkit-scrollbar {
  display: none;
}
textarea {
  scrollbar-width: none;
}

/* Dropdown Transition */
.dropdown-enter-active {
  animation: dropdown-in 0.15s ease-out;
}
.dropdown-leave-active {
  animation: dropdown-out 0.1s ease-in;
}

@keyframes dropdown-in {
  from { opacity: 0; transform: translateY(8px) scale(0.95); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@keyframes dropdown-out {
  from { opacity: 1; transform: translateY(0) scale(1); }
  to { opacity: 0; transform: translateY(8px) scale(0.95); }
}
</style>
