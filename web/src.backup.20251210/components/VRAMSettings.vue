<template>
  <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
    <div class="flex items-start justify-between mb-4">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          GPU VRAM Management
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Konfiguriere wie GPU-Speicher beim Laden von Modellen verwaltet wird
        </p>
      </div>
      <button
        @click="loadVRAMSettings"
        class="px-3 py-1 text-sm bg-blue-100 dark:bg-blue-800 text-blue-700 dark:text-blue-100 rounded hover:bg-blue-200 dark:hover:bg-blue-700"
        :disabled="loading"
      >
        Aktualisieren
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="flex justify-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
    </div>

    <div v-else>
      <!-- Current VRAM Status -->
      <div class="mb-6 p-4 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 border border-green-200 dark:border-green-800 rounded-lg">
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <span class="text-2xl">GPU</span>
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ vramInfo.gpuName || 'Keine GPU erkannt' }}
              </p>
              <p v-if="vramInfo.available" class="text-xs text-gray-600 dark:text-gray-400">
                {{ vramInfo.usedMB }} MB / {{ vramInfo.totalMB }} MB verwendet ({{ vramInfo.percentUsed }}%)
              </p>
            </div>
          </div>
          <div v-if="vramInfo.available" class="text-right">
            <p class="text-lg font-bold text-green-600 dark:text-green-400">
              {{ vramInfo.freeMB }} MB frei
            </p>
          </div>
        </div>

        <!-- VRAM Bar -->
        <div v-if="vramInfo.available" class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-4 overflow-hidden">
          <div
            class="h-full rounded-full transition-all duration-300"
            :class="getVRAMBarColor(vramInfo.percentUsed)"
            :style="{ width: `${vramInfo.percentUsed}%` }"
          ></div>
        </div>
        <p v-if="vramInfo.available" class="text-xs text-gray-500 dark:text-gray-400 mt-2 text-center">
          {{ getVRAMStatusText(vramInfo.percentUsed) }}
        </p>

        <!-- No GPU Warning -->
        <div v-if="!vramInfo.available" class="mt-2 p-3 bg-yellow-100 dark:bg-yellow-900/30 border border-yellow-300 dark:border-yellow-700 rounded-lg">
          <p class="text-sm text-yellow-800 dark:text-yellow-200">
            Keine NVIDIA GPU erkannt oder nvidia-smi nicht verfügbar.
            VRAM-Management-Funktionen sind deaktiviert.
          </p>
        </div>
      </div>

      <!-- VRAM Strategy Selection -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
          VRAM-Strategie
        </label>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <button
            v-for="strategy in availableStrategies"
            :key="strategy.id"
            @click="selectStrategy(strategy.id)"
            class="p-4 rounded-lg border-2 transition-all duration-200 text-left"
            :class="selectedStrategy === strategy.id
              ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/40 shadow-lg'
              : 'border-gray-300 dark:border-gray-600 hover:border-blue-400'"
          >
            <div class="flex items-center justify-between mb-2">
              <span class="font-semibold text-gray-900 dark:text-white">
                {{ strategy.name }}
              </span>
              <span v-if="strategy.recommended" class="text-xs px-2 py-0.5 bg-green-100 dark:bg-green-900/40 text-green-800 dark:text-green-200 rounded-full">
                Empfohlen
              </span>
            </div>
            <p class="text-xs text-gray-600 dark:text-gray-400">
              {{ strategy.description }}
            </p>
          </button>
        </div>
      </div>

      <!-- VRAM Reserve -->
      <div class="mb-6">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          VRAM-Reserve (MB)
          <span class="text-xs text-gray-500 dark:text-gray-400 ml-2">
            Speicher der immer frei bleiben soll
          </span>
        </label>
        <div class="flex items-center gap-4">
          <input
            v-model.number="reserveMB"
            type="range"
            min="0"
            max="2048"
            step="128"
            class="flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
          />
          <input
            v-model.number="reserveMB"
            type="number"
            min="0"
            max="4096"
            class="w-24 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-center"
          />
          <span class="text-sm text-gray-500 dark:text-gray-400">MB</span>
        </div>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
          Reservierter Speicher für System und andere Anwendungen. Empfohlen: 512-1024 MB
        </p>
      </div>

      <!-- Actions -->
      <div class="flex flex-col sm:flex-row gap-3">
        <button
          @click="saveSettings"
          :disabled="saving"
          class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <span v-if="saving" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></span>
          <span v-else>Speichern</span>
        </button>

        <button
          @click="clearVRAM"
          :disabled="clearing || !vramInfo.available"
          class="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <span v-if="clearing" class="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></span>
          <span v-else>VRAM Leeren</span>
        </button>
      </div>

      <!-- Save Status -->
      <div v-if="saveStatus" class="mt-4">
        <div
          class="px-4 py-2 rounded-lg text-sm"
          :class="saveStatus.success
            ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
            : 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-200'"
        >
          {{ saveStatus.message }}
        </div>
      </div>

      <!-- Strategy Info Box -->
      <div class="mt-6 p-4 bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700 rounded-lg">
        <h4 class="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
          Strategien im Detail
        </h4>
        <ul class="space-y-2 text-xs text-gray-600 dark:text-gray-400">
          <li class="flex items-start gap-2">
            <span class="font-medium text-blue-600 dark:text-blue-400">Smart Swap:</span>
            <span>Prueft ob genug VRAM frei ist. Loescht nur wenn noetig - schnellste Option.</span>
          </li>
          <li class="flex items-start gap-2">
            <span class="font-medium text-green-600 dark:text-green-400">Always Clear:</span>
            <span>Loescht immer den VRAM vor jedem Modell-Wechsel - sicherste Option.</span>
          </li>
          <li class="flex items-start gap-2">
            <span class="font-medium text-purple-600 dark:text-purple-400">Smart Offload:</span>
            <span>Berechnet automatisch wie viele Layer auf die GPU passen. Ermoeglicht grosse Modelle (70B+).</span>
          </li>
          <li class="flex items-start gap-2">
            <span class="font-medium text-gray-600 dark:text-gray-400">Manual:</span>
            <span>Keine automatische Verwaltung - fuer erfahrene Benutzer.</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'

// State
const loading = ref(false)
const saving = ref(false)
const clearing = ref(false)
const saveStatus = ref(null)

const vramInfo = ref({
  totalMB: 0,
  usedMB: 0,
  freeMB: 0,
  percentUsed: 0,
  gpuName: '',
  available: false
})

const availableStrategies = ref([])
const selectedStrategy = ref('smart_swap')
const reserveMB = ref(512)

// Methods
async function loadVRAMSettings() {
  loading.value = true
  try {
    const response = await fetch('/api/llamaserver/vram')
    if (!response.ok) throw new Error('Failed to load VRAM settings')

    const data = await response.json()
    vramInfo.value = data.currentVram
    availableStrategies.value = data.availableStrategies
    selectedStrategy.value = data.strategy
    reserveMB.value = data.reserveMB
  } catch (error) {
    console.error('Failed to load VRAM settings:', error)
    saveStatus.value = { success: false, message: 'Fehler beim Laden der VRAM-Einstellungen' }
  } finally {
    loading.value = false
  }
}

function selectStrategy(strategyId) {
  selectedStrategy.value = strategyId
}

async function saveSettings() {
  saving.value = true
  saveStatus.value = null

  try {
    const response = await fetch('/api/llamaserver/vram', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        strategy: selectedStrategy.value,
        reserveMB: reserveMB.value
      })
    })

    if (!response.ok) throw new Error('Failed to save settings')

    const data = await response.json()
    if (data.success) {
      vramInfo.value = data.settings.currentVram
      saveStatus.value = { success: true, message: 'VRAM-Einstellungen erfolgreich gespeichert!' }
    } else {
      throw new Error(data.error || 'Unknown error')
    }
  } catch (error) {
    console.error('Failed to save VRAM settings:', error)
    saveStatus.value = { success: false, message: 'Fehler beim Speichern: ' + error.message }
  } finally {
    saving.value = false
    setTimeout(() => { saveStatus.value = null }, 5000)
  }
}

async function clearVRAM() {
  if (!confirm('VRAM leeren? Dies beendet alle laufenden llama-server Prozesse.')) {
    return
  }

  clearing.value = true
  saveStatus.value = null

  try {
    const response = await fetch('/api/llamaserver/vram/clear', {
      method: 'POST'
    })

    if (!response.ok) throw new Error('Failed to clear VRAM')

    const data = await response.json()
    if (data.success) {
      vramInfo.value = data.vram
      saveStatus.value = { success: true, message: 'VRAM erfolgreich geleert!' }
    } else {
      throw new Error(data.error || 'Unknown error')
    }
  } catch (error) {
    console.error('Failed to clear VRAM:', error)
    saveStatus.value = { success: false, message: 'Fehler beim Leeren: ' + error.message }
  } finally {
    clearing.value = false
    setTimeout(() => { saveStatus.value = null }, 5000)
  }
}

function getVRAMBarColor(percent) {
  if (percent < 50) return 'bg-green-500'
  if (percent < 75) return 'bg-yellow-500'
  if (percent < 90) return 'bg-orange-500'
  return 'bg-red-500'
}

function getVRAMStatusText(percent) {
  if (percent < 50) return 'Viel Speicher verfuegbar - optimale Leistung'
  if (percent < 75) return 'Guter Speicherstand - Modell geladen'
  if (percent < 90) return 'Wenig Speicher frei - grosse Modelle koennen Probleme haben'
  return 'Kritisch wenig Speicher - VRAM leeren empfohlen'
}

// Initialize
onMounted(() => {
  loadVRAMSettings()
})
</script>
