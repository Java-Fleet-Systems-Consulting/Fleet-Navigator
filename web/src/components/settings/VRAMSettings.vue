<template>
  <div class="vram-settings">
    <!-- Header -->
    <div class="flex items-center justify-between mb-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
        GPU / VRAM Einstellungen
      </h3>
      <button
        @click="refreshAll"
        :disabled="loading"
        class="px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors flex items-center gap-2"
      >
        <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Aktualisieren
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading && !gpuInfo" class="flex items-center justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
      <span class="ml-3 text-gray-600 dark:text-gray-400">Lade GPU-Informationen...</span>
    </div>

    <!-- No GPU Available -->
    <div v-else-if="!gpuInfo?.available" class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
      <div class="flex items-center gap-3">
        <svg class="w-6 h-6 text-yellow-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <p class="font-medium text-yellow-800 dark:text-yellow-200">Keine NVIDIA GPU erkannt</p>
          <p class="text-sm text-yellow-600 dark:text-yellow-400">GPU-Beschleunigung ist nicht verfügbar. Modelle laufen auf der CPU.</p>
        </div>
      </div>
    </div>

    <!-- GPU Info Available -->
    <div v-else class="space-y-6">
      <!-- GPU Info Card -->
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <div class="flex items-center gap-3 mb-3">
          <div class="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
            <svg class="w-6 h-6 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
            </svg>
          </div>
          <div>
            <p class="font-medium text-gray-900 dark:text-white">{{ gpuInfo.gpuName }}</p>
            <p class="text-sm text-gray-500 dark:text-gray-400">GPU erkannt und bereit</p>
          </div>
        </div>

        <!-- VRAM Bar -->
        <div class="mt-4">
          <div class="flex justify-between text-sm mb-1">
            <span class="text-gray-600 dark:text-gray-400">VRAM Nutzung</span>
            <span class="font-medium" :class="vramColorClass">
              {{ formatMB(gpuInfo.usedMB) }} / {{ formatMB(gpuInfo.totalMB) }} ({{ gpuInfo.percentUsed }}%)
            </span>
          </div>
          <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
            <div
              class="h-3 rounded-full transition-all duration-300"
              :class="vramBarColorClass"
              :style="{ width: `${gpuInfo.percentUsed}%` }"
            ></div>
          </div>
          <div class="flex justify-between text-xs text-gray-500 dark:text-gray-400 mt-1">
            <span>Frei: {{ formatMB(gpuInfo.freeMB) }}</span>
            <button
              @click="clearVRAM"
              :disabled="clearing"
              class="text-red-500 hover:text-red-600 dark:text-red-400 dark:hover:text-red-300 underline"
            >
              {{ clearing ? 'Leere VRAM...' : 'VRAM leeren' }}
            </button>
          </div>
        </div>
      </div>

      <!-- GPU Layers Configuration -->
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <h4 class="font-medium text-gray-900 dark:text-white mb-3">GPU-Layer Konfiguration</h4>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Bestimmt wie viele Modell-Layer auf der GPU laufen. Mehr Layer = schneller, aber mehr VRAM.
        </p>

        <!-- Auto/Manual Toggle -->
        <div class="space-y-3">
          <label class="flex items-start gap-3 cursor-pointer">
            <input
              type="radio"
              v-model="layerMode"
              value="auto"
              class="mt-1 w-4 h-4 text-blue-600"
            />
            <div>
              <span class="font-medium text-gray-900 dark:text-white">Automatisch (empfohlen)</span>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                Berechnet optimale Layer basierend auf verfügbarem VRAM
              </p>
            </div>
          </label>

          <label class="flex items-start gap-3 cursor-pointer">
            <input
              type="radio"
              v-model="layerMode"
              value="manual"
              class="mt-1 w-4 h-4 text-blue-600"
            />
            <div class="flex-1">
              <span class="font-medium text-gray-900 dark:text-white">Manuell</span>
              <div v-if="layerMode === 'manual'" class="mt-2 flex items-center gap-3">
                <input
                  type="number"
                  v-model.number="manualLayers"
                  min="0"
                  max="999"
                  class="w-24 px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
                <span class="text-sm text-gray-500 dark:text-gray-400">Layer</span>
              </div>
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                99 = alle Layer auf GPU, 0 = nur CPU
              </p>
            </div>
          </label>
        </div>

        <!-- Current Setting Display -->
        <div class="mt-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
          <p class="text-sm text-blue-800 dark:text-blue-200">
            <strong>Aktuelle Einstellung:</strong> {{ llamaConfig?.gpuLayers ?? '?' }} Layer
            <span v-if="layerMode === 'auto'" class="text-blue-600 dark:text-blue-400">(Automatisch)</span>
          </p>
        </div>

        <!-- Save Button -->
        <button
          @click="saveLayerConfig"
          :disabled="saving"
          class="mt-4 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50"
        >
          {{ saving ? 'Speichere...' : 'Layer-Einstellung speichern' }}
        </button>
      </div>

      <!-- VRAM Strategy -->
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <h4 class="font-medium text-gray-900 dark:text-white mb-3">VRAM-Strategie</h4>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Wie soll der Navigator mit VRAM umgehen wenn ein Modell geladen wird?
        </p>

        <select
          v-model="selectedStrategy"
          @change="saveVRAMStrategy"
          class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
        >
          <option
            v-for="strategy in vramSettings?.availableStrategies"
            :key="strategy.id"
            :value="strategy.id"
          >
            {{ strategy.name }} - {{ strategy.description }}
          </option>
        </select>

        <!-- Strategy Descriptions -->
        <div class="mt-3 text-sm text-gray-600 dark:text-gray-400">
          <div v-if="selectedStrategy === 'smart_swap'" class="flex items-start gap-2">
            <span class="text-green-500">&#10003;</span>
            <span>VRAM wird nur geleert wenn nötig. Schneller Modell-Wechsel wenn genug Platz.</span>
          </div>
          <div v-else-if="selectedStrategy === 'always_clear'" class="flex items-start gap-2">
            <span class="text-blue-500">&#10003;</span>
            <span>VRAM wird vor jedem Modell-Laden geleert. Sicherste Option.</span>
          </div>
          <div v-else-if="selectedStrategy === 'smart_offload'" class="flex items-start gap-2">
            <span class="text-yellow-500">&#10003;</span>
            <span>Reduziert automatisch GPU-Layer wenn VRAM knapp wird.</span>
          </div>
          <div v-else-if="selectedStrategy === 'manual'" class="flex items-start gap-2">
            <span class="text-gray-500">&#10003;</span>
            <span>Keine automatische VRAM-Verwaltung. Du kontrollierst alles.</span>
          </div>
        </div>
      </div>

      <!-- VRAM Reserve -->
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <h4 class="font-medium text-gray-900 dark:text-white mb-3">VRAM Reserve</h4>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          VRAM das für System und Desktop reserviert wird (nicht für Modelle verwendet).
        </p>

        <div class="flex items-center gap-3">
          <input
            type="number"
            v-model.number="reserveMB"
            min="0"
            max="4096"
            step="128"
            class="w-32 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          />
          <span class="text-gray-600 dark:text-gray-400">MB</span>
          <button
            @click="saveVRAMReserve"
            :disabled="saving"
            class="px-4 py-2 bg-gray-200 dark:bg-gray-600 hover:bg-gray-300 dark:hover:bg-gray-500 rounded-lg transition-colors"
          >
            Speichern
          </button>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
          Empfohlen: 512 MB für Desktop, 1024 MB wenn du nebenbei andere GPU-Anwendungen nutzt.
        </p>
      </div>

      <!-- Memory Optimizations (mmap/mlock) -->
      <div class="bg-gray-50 dark:bg-gray-800 rounded-lg p-4">
        <h4 class="font-medium text-gray-900 dark:text-white mb-3">Speicher-Optimierungen</h4>
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Optionen für schnelleres Modell-Laden und bessere Speicherverwaltung.
        </p>

        <div class="space-y-4">
          <!-- mmap Toggle -->
          <label class="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              v-model="useMmap"
              @change="saveMmapSetting"
              class="mt-1 w-5 h-5 text-blue-600 rounded"
            />
            <div>
              <span class="font-medium text-gray-900 dark:text-white">Memory-Mapped I/O (mmap)</span>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                Modell wird direkt in den Speicher eingeblendet. Schnelleres Laden, besonders bei wiederholtem Start.
              </p>
              <span class="text-xs text-green-600 dark:text-green-400">Empfohlen</span>
            </div>
          </label>

          <!-- mlock Toggle -->
          <label class="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              v-model="useMlock"
              @change="saveMlockSetting"
              class="mt-1 w-5 h-5 text-blue-600 rounded"
            />
            <div>
              <span class="font-medium text-gray-900 dark:text-white">Memory-Lock (mlock)</span>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                Verhindert, dass das Modell auf die Festplatte ausgelagert wird (kein Swap). Hält das Modell im RAM.
              </p>
              <div class="mt-1 flex items-center gap-2">
                <span class="text-xs text-yellow-600 dark:text-yellow-400">⚠️ Benötigt erhöhte Berechtigungen</span>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Falls Fehler auftreten: <code class="bg-gray-200 dark:bg-gray-700 px-1 rounded">ulimit -l unlimited</code> oder systemd LimitMEMLOCK=infinity
              </p>
            </div>
          </label>
        </div>
      </div>

      <!-- Save Status -->
      <div v-if="saveStatus" class="p-3 rounded-lg" :class="saveStatus.success ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200' : 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200'">
        {{ saveStatus.message }}
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../../services/api'

// State
const loading = ref(true)
const saving = ref(false)
const clearing = ref(false)
const gpuInfo = ref(null)
const llamaConfig = ref(null)
const vramSettings = ref(null)
const saveStatus = ref(null)

// Layer Config
const layerMode = ref('auto')
const manualLayers = ref(99)

// VRAM Settings
const selectedStrategy = ref('smart_swap')
const reserveMB = ref(512)
const useMmap = ref(true)
const useMlock = ref(false)

// Computed
const vramColorClass = computed(() => {
  if (!gpuInfo.value) return 'text-gray-600'
  const percent = gpuInfo.value.percentUsed
  if (percent >= 90) return 'text-red-600 dark:text-red-400'
  if (percent >= 70) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-green-600 dark:text-green-400'
})

const vramBarColorClass = computed(() => {
  if (!gpuInfo.value) return 'bg-gray-400'
  const percent = gpuInfo.value.percentUsed
  if (percent >= 90) return 'bg-red-500'
  if (percent >= 70) return 'bg-yellow-500'
  return 'bg-green-500'
})

// Methods
function formatMB(mb) {
  if (mb >= 1024) {
    return `${(mb / 1024).toFixed(1)} GB`
  }
  return `${mb} MB`
}

async function loadVRAMInfo() {
  try {
    gpuInfo.value = await api.getVRAMInfo()
  } catch (error) {
    console.error('Fehler beim Laden der VRAM-Info:', error)
    gpuInfo.value = { available: false }
  }
}

async function loadLlamaConfig() {
  try {
    llamaConfig.value = await api.getLlamaServerConfig()
    // Determine if auto or manual based on current value
    if (llamaConfig.value?.gpuLayers === 99 || llamaConfig.value?.gpuLayers === -1) {
      layerMode.value = 'auto'
    } else {
      layerMode.value = 'manual'
      manualLayers.value = llamaConfig.value?.gpuLayers ?? 99
    }
  } catch (error) {
    console.error('Fehler beim Laden der LlamaServer-Config:', error)
  }
}

async function loadVRAMSettings() {
  try {
    vramSettings.value = await api.getVRAMSettings()
    selectedStrategy.value = vramSettings.value?.strategy || 'smart_swap'
    reserveMB.value = vramSettings.value?.reserveMB || 512
    useMmap.value = vramSettings.value?.useMmap ?? true
    useMlock.value = vramSettings.value?.useMlock ?? false
  } catch (error) {
    console.error('Fehler beim Laden der VRAM-Settings:', error)
  }
}

async function refreshAll() {
  loading.value = true
  await Promise.all([
    loadVRAMInfo(),
    loadLlamaConfig(),
    loadVRAMSettings()
  ])
  loading.value = false
}

async function saveLayerConfig() {
  saving.value = true
  try {
    const layers = layerMode.value === 'auto' ? 99 : manualLayers.value
    await api.updateLlamaServerConfig({ gpuLayers: layers })
    await loadLlamaConfig()
    showSaveStatus(true, 'GPU-Layer Einstellung gespeichert!')
  } catch (error) {
    console.error('Fehler beim Speichern:', error)
    showSaveStatus(false, 'Fehler beim Speichern: ' + error.message)
  } finally {
    saving.value = false
  }
}

async function saveVRAMStrategy() {
  saving.value = true
  try {
    await api.updateVRAMSettings({ strategy: selectedStrategy.value })
    showSaveStatus(true, 'VRAM-Strategie gespeichert!')
  } catch (error) {
    console.error('Fehler beim Speichern:', error)
    showSaveStatus(false, 'Fehler beim Speichern: ' + error.message)
  } finally {
    saving.value = false
  }
}

async function saveVRAMReserve() {
  saving.value = true
  try {
    await api.updateVRAMSettings({ reserveMB: reserveMB.value })
    showSaveStatus(true, 'VRAM-Reserve gespeichert!')
  } catch (error) {
    console.error('Fehler beim Speichern:', error)
    showSaveStatus(false, 'Fehler beim Speichern: ' + error.message)
  } finally {
    saving.value = false
  }
}

async function saveMmapSetting() {
  saving.value = true
  try {
    await api.updateVRAMSettings({ useMmap: useMmap.value })
    showSaveStatus(true, `mmap ${useMmap.value ? 'aktiviert' : 'deaktiviert'}!`)
  } catch (error) {
    console.error('Fehler beim Speichern:', error)
    showSaveStatus(false, 'Fehler beim Speichern: ' + error.message)
  } finally {
    saving.value = false
  }
}

async function saveMlockSetting() {
  saving.value = true
  try {
    await api.updateVRAMSettings({ useMlock: useMlock.value })
    showSaveStatus(true, `mlock ${useMlock.value ? 'aktiviert' : 'deaktiviert'}!`)
  } catch (error) {
    console.error('Fehler beim Speichern:', error)
    showSaveStatus(false, 'Fehler beim Speichern: ' + error.message)
  } finally {
    saving.value = false
  }
}

async function clearVRAM() {
  if (!confirm('VRAM leeren? Dies beendet alle laufenden llama-server Prozesse.')) {
    return
  }
  clearing.value = true
  try {
    await api.clearVRAM()
    await loadVRAMInfo()
    showSaveStatus(true, 'VRAM wurde geleert!')
  } catch (error) {
    console.error('Fehler beim Leeren:', error)
    showSaveStatus(false, 'Fehler: ' + error.message)
  } finally {
    clearing.value = false
  }
}

function showSaveStatus(success, message) {
  saveStatus.value = { success, message }
  setTimeout(() => {
    saveStatus.value = null
  }, 3000)
}

// Lifecycle
onMounted(() => {
  refreshAll()
})
</script>

<style scoped>
.vram-settings {
  @apply text-gray-900 dark:text-gray-100;
}
</style>
