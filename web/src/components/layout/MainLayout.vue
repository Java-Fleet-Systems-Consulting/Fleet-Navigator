<template>
  <div class="h-screen flex flex-col overflow-hidden bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-gray-100" :class="rootClasses">
    <!-- Top Bar (Full Width) - kann ausgeblendet werden -->
    <TopBar
      v-if="settingsStore.settings.showTopBar"
      @toggle-monitor="showMonitor = !showMonitor"
      @toggle-model-manager="showModelManager = !showModelManager"
      @toggle-settings="showSettings = !showSettings"
      @open-settings-tab="openSettingsOnTab"
      @toggle-theme="toggleDarkMode"
      :dark-mode="darkMode"
    />

    <!-- Content Area: Sidebar + Main -->
    <div class="flex flex-1 min-h-0 overflow-hidden">
      <!-- Sidebar -->
      <Sidebar
        @select-project="handleSelectProject"
        @new-chat="handleNewChat"
        @open-settings="showSettings = true"
      />

      <!-- Main Content (Router View) -->
      <div class="flex-1 flex flex-col overflow-hidden">
        <router-view
          v-slot="{ Component }"
          :selected-project="selectedProject"
          :project-chats="projectChats"
          @close-project="selectedProject = null"
          @refresh-project="refreshProject"
        >
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </div>

    <!-- System Monitor (collapsible slide-in panel) -->
    <Transition name="slide">
      <SystemMonitor v-if="showMonitor" @close="showMonitor = false" />
    </Transition>

    <!-- Model Manager Modal -->
    <Transition name="fade">
      <ModelManager v-if="showModelManager" @close="showModelManager = false" />
    </Transition>

    <!-- Settings Modal -->
    <Transition name="fade">
      <SettingsModal v-if="showSettings" :is-open="showSettings" :initial-tab="settingsInitialTab" @close="showSettings = false; settingsInitialTab = null" />
    </Transition>

    <!-- Info Dialog -->
    <InfoDialog :show="showInfoDialog" @close="showInfoDialog = false" />

    <!-- Abort Confirmation Modal -->
    <Transition name="fade">
      <div v-if="showAbortModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md shadow-xl">
          <h3 class="text-lg font-bold mb-2 text-gray-900 dark:text-white">✋ Anfrage abgebrochen</h3>
          <p class="text-gray-600 dark:text-gray-300 mb-4">
            Die AI-Anfrage wurde erfolgreich abgebrochen.
          </p>
          <button
            @click="showAbortModal = false"
            class="w-full px-4 py-2 bg-fleet-orange-500 hover:bg-fleet-orange-600 text-white rounded-lg transition-colors"
          >
            OK
          </button>
        </div>
      </div>
    </Transition>

    <!-- Toast Notifications -->
    <ToastContainer />

    <!-- Setup Wizard Modal (shown on first run without models) -->
    <SetupWizardModal
      :is-visible="showSetupWizard"
      @complete="handleSetupComplete"
      @close="showSetupWizard = false"
    />

    <!-- AI Startup Overlay (shown while llama-server starts) -->
    <AiStartupOverlay />
  </div>
</template>

<script setup>
import { ref, provide, onMounted, watch, computed } from 'vue'
import { useRouter } from 'vue-router'
import Sidebar from '../Sidebar.vue'
import TopBar from '../TopBar.vue'
import SystemMonitor from '../SystemMonitor.vue'
import ModelManager from '../ModelManager.vue'
import SettingsModal from '../SettingsModal.vue'
import SetupWizardModal from '../SetupWizardModal.vue'
import ToastContainer from '../ToastContainer.vue'
import AiStartupOverlay from '../AiStartupOverlay.vue'
import InfoDialog from '../InfoDialog.vue'
import { useChatStore } from '../../stores/chatStore'
import { useSettingsStore } from '../../stores/settingsStore'
import api from '../../services/api'
import axios from 'axios'

const chatStore = useChatStore()
const settingsStore = useSettingsStore()
const router = useRouter()
const showMonitor = ref(false)
const showModelManager = ref(false)
const showSettings = ref(false)
const showAbortModal = ref(false)
const showSetupWizard = ref(false)
const showInfoDialog = ref(false)
const selectedProject = ref(null)
const settingsInitialTab = ref(null)

// Get chats that belong to the selected project
const projectChats = computed(() => {
  if (!selectedProject.value) return []
  return (chatStore.chats || []).filter(chat => chat.projectId === selectedProject.value.id)
})

// Dark Mode IMMER als Default
const darkMode = ref(true)

// Computed class for root element (dark mode + ui theme)
const rootClasses = computed(() => {
  const classes = []
  if (darkMode.value) {
    classes.push('dark')
  }
  // Apply UI theme class (e.g., 'theme-lawyer' for lawyer style)
  const uiTheme = settingsStore.settings.uiTheme
  if (uiTheme && uiTheme !== 'default') {
    classes.push(`theme-${uiTheme}`)
  }
  return classes
})

onMounted(async () => {
  darkMode.value = true
  localStorage.setItem('darkMode', 'true')
  // Dark-Mode Klasse auf <html> setzen für Teleport-Komponenten (ConfirmDialog etc.)
  applyDarkModeToDocument(true)
  // Theme-Klasse auf <html> setzen für Teleport-Komponenten (ConfirmDialog etc.)
  applyThemeToDocument(settingsStore.settings.uiTheme)
  await chatStore.loadModels()
  // Sync vision settings with backend (H2 DB = Source of Truth)
  await settingsStore.syncVisionSettingsWithBackend()

  // Schriftgröße beim Start anwenden
  applyFontSize(settingsStore.settings.fontSize || 'medium')

  // Check if setup is needed (no models available)
  await checkSetupStatus()
})

// Dark-Mode und Theme auf document.documentElement setzen (wichtig für Teleport-Komponenten!)
function applyDarkModeToDocument(isDark) {
  const root = document.documentElement
  if (isDark) {
    root.classList.add('dark')
  } else {
    root.classList.remove('dark')
  }
}

// Theme-Klasse auf document.documentElement setzen (wichtig für Teleport-Komponenten wie ConfirmDialog!)
function applyThemeToDocument(theme) {
  const root = document.documentElement
  // Alle vorherigen Theme-Klassen entfernen
  root.classList.forEach(cls => {
    if (cls.startsWith('theme-')) {
      root.classList.remove(cls)
    }
  })
  // Neue Theme-Klasse hinzufügen
  if (theme && theme !== 'default') {
    root.classList.add(`theme-${theme}`)
  }
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

async function checkSetupStatus() {
  try {
    const response = await axios.get('/api/system/setup-status')
    if (response.data.needsSetup) {
      console.log('Setup needed - showing wizard')
      showSetupWizard.value = true
    }
  } catch (err) {
    console.error('Could not check setup status:', err)
  }
}

function handleSetupComplete() {
  showSetupWizard.value = false
  // Reload models after setup
  chatStore.loadModels()
}

// Toggle function
function toggleDarkMode() {
  darkMode.value = !darkMode.value
  localStorage.setItem('darkMode', JSON.stringify(darkMode.value))
  // Auch auf <html> Element aktualisieren für Teleport-Komponenten
  applyDarkModeToDocument(darkMode.value)
}

function openSettingsOnTab(tabName) {
  settingsInitialTab.value = tabName
  showSettings.value = true
}

// Watch for chat changes - close project view if chat is not part of current project
watch(() => chatStore.currentChat, (newChat) => {
  if (newChat && selectedProject.value) {
    if (!newChat.projectId || newChat.projectId !== selectedProject.value.id) {
      selectedProject.value = null
    }
  }
})

// Watch for font size changes
watch(() => settingsStore.settings.fontSize, (newSize) => {
  if (newSize) {
    applyFontSize(newSize)
  }
})

// Watch for UI theme changes (wichtig für Teleport-Komponenten!)
watch(() => settingsStore.settings.uiTheme, (newTheme) => {
  applyThemeToDocument(newTheme)
})

// Handle project selection
async function handleSelectProject(project) {
  try {
    const fullProject = await api.getProject(project.id)
    selectedProject.value = fullProject
  } catch (err) {
    console.error('Failed to load project:', err)
    selectedProject.value = project
  }
}

// Handle new chat - close project view and start new chat
function handleNewChat() {
  selectedProject.value = null
  chatStore.startNewChat()
}

// Refresh project after file changes
async function refreshProject() {
  if (!selectedProject.value) return
  try {
    const fullProject = await api.getProject(selectedProject.value.id)
    selectedProject.value = fullProject
  } catch (err) {
    console.error('Failed to refresh project:', err)
  }
}

// Provide to child components
provide('darkMode', darkMode)
provide('showAbortModal', showAbortModal)
provide('selectedProject', selectedProject)
provide('projectChats', projectChats)
provide('openSettings', () => { showSettings.value = true })
provide('toggleSystemMonitor', () => { showMonitor.value = !showMonitor.value })
provide('openModelManager', () => { showModelManager.value = true })
provide('openMates', () => {
  // Fleet Mates in neuem Tab öffnen
  const route = router.resolve({ name: 'fleet-mates' })
  window.open(route.href, '_blank')
})
provide('openInfo', () => { showInfoDialog.value = true })
</script>

<style>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.3s;
}
.fade-enter-from, .fade-leave-to {
  opacity: 0;
}
</style>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease;
}

.slide-enter-from {
  transform: translateX(100%);
}

.slide-leave-to {
  transform: translateX(100%);
}
</style>
