<template>
  <div class="space-y-6">
    <!-- Header mit Status -->
    <section class="bg-gradient-to-br from-emerald-50 to-green-100 dark:from-emerald-900/30 dark:to-green-800/30 p-5 rounded-xl border border-emerald-200/50 dark:border-emerald-700/50 shadow-sm">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
          <ChartBarIcon class="w-5 h-5 text-emerald-500" />
          Observer Status
        </h3>
        <div class="flex items-center gap-2">
          <span
            class="px-3 py-1 rounded-full text-xs font-medium"
            :class="status.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'"
          >
            {{ status.enabled ? 'Aktiv' : 'Inaktiv' }}
          </span>
          <span
            v-if="status.running"
            class="px-3 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 animate-pulse"
          >
            Sammelt...
          </span>
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="bg-white/60 dark:bg-gray-800/60 rounded-lg p-3 text-center">
          <div class="text-2xl font-bold text-emerald-600 dark:text-emerald-400">{{ stats.totalValues || 0 }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-400">Datenpunkte</div>
        </div>
        <div class="bg-white/60 dark:bg-gray-800/60 rounded-lg p-3 text-center">
          <div class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ stats.indicatorCount || 0 }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-400">Indikatoren</div>
        </div>
        <div class="bg-white/60 dark:bg-gray-800/60 rounded-lg p-3 text-center">
          <div class="text-2xl font-bold text-purple-600 dark:text-purple-400">{{ stats.sourceCount || 0 }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-400">Quellen</div>
        </div>
        <div class="bg-white/60 dark:bg-gray-800/60 rounded-lg p-3 text-center">
          <div class="text-2xl font-bold text-orange-600 dark:text-orange-400">{{ stats.totalRuns || 0 }}</div>
          <div class="text-xs text-gray-500 dark:text-gray-400">Sammelläufe</div>
        </div>
      </div>

      <!-- Letzter Lauf Info -->
      <div v-if="stats.lastRunAt" class="mt-4 text-sm text-gray-600 dark:text-gray-400">
        Letzter Lauf: {{ formatDate(stats.lastRunAt) }}
        <span :class="stats.lastRunStatus === 'COMPLETED' ? 'text-emerald-600' : 'text-orange-600'">
          ({{ stats.lastRunStatus }})
        </span>
      </div>
    </section>

    <!-- Aktivierung -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <Cog6ToothIcon class="w-5 h-5 text-gray-500" />
        Konfiguration
      </h3>

      <!-- Enable/Disable Toggle -->
      <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <PlayIcon class="w-4 h-4 text-emerald-500" />
              Observer aktivieren
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Automatische tägliche Sammlung von Finanz- und Wirtschaftsdaten
            </p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.enabled" @change="saveConfig" class="sr-only peer">
            <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-emerald-300 dark:peer-focus:ring-emerald-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-emerald-500"></div>
          </label>
        </div>
      </div>

      <!-- Sammelzeit -->
      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
          <ClockIcon class="w-4 h-4" />
          Tägliche Sammelzeit
        </label>
        <input
          type="time"
          v-model="config.dailyCollectionTime"
          @change="saveConfig"
          class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
        />
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Daten werden täglich um diese Uhrzeit gesammelt (nur Werktage)
        </p>
      </div>

      <!-- Auto-Backfill Toggle -->
      <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
              <ArrowPathIcon class="w-4 h-4 text-blue-500" />
              Auto-Backfill
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
              Fehlende historische Daten automatisch nachholen
            </p>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.autoBackfill" @change="saveConfig" class="sr-only peer">
            <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-blue-500"></div>
          </label>
        </div>
      </div>

      <!-- Max Backfill Days -->
      <div v-if="config.autoBackfill" class="mb-4">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Maximale Backfill-Tage
        </label>
        <input
          type="number"
          v-model.number="config.maxBackfillDays"
          @change="saveConfig"
          min="30"
          max="3650"
          class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-emerald-500 focus:border-transparent"
        />
      </div>
    </section>

    <!-- Transparenz / Prompt -->
    <section class="bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-blue-900/30 dark:to-indigo-800/30 p-5 rounded-xl border border-blue-200/50 dark:border-blue-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <DocumentTextIcon class="w-5 h-5 text-blue-500" />
        Transparenz - Observer-Prompt
      </h3>
      <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
        Dieser Prompt beschreibt die Aufgabe und das Verhalten des Observers. Er kann vom Finanzberater-Experten gelesen werden.
      </p>
      <textarea
        v-model="config.prompt"
        @change="saveConfig"
        rows="8"
        class="w-full px-4 py-3 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-blue-500 focus:border-transparent font-mono text-sm"
        placeholder="Beschreibung des Observer-Verhaltens..."
      />
    </section>

    <!-- Datenquellen Status -->
    <section class="bg-gradient-to-br from-purple-50 to-violet-100 dark:from-purple-900/30 dark:to-violet-800/30 p-5 rounded-xl border border-purple-200/50 dark:border-purple-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <ServerIcon class="w-5 h-5 text-purple-500" />
        Datenquellen
      </h3>

      <div class="space-y-2">
        <div
          v-for="(available, source) in status.collectorStatus"
          :key="source"
          class="flex items-center justify-between p-3 bg-white/60 dark:bg-gray-800/60 rounded-lg"
        >
          <div class="flex items-center gap-3">
            <div
              class="w-3 h-3 rounded-full"
              :class="available ? 'bg-emerald-500' : 'bg-red-500'"
            />
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ getSourceName(source) }}</span>
          </div>
          <span class="text-xs text-gray-500">{{ available ? 'Erreichbar' : 'Nicht erreichbar' }}</span>
        </div>
      </div>
    </section>

    <!-- Aktionen -->
    <section class="bg-gradient-to-br from-orange-50 to-amber-100 dark:from-orange-900/30 dark:to-amber-800/30 p-5 rounded-xl border border-orange-200/50 dark:border-orange-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <BoltIcon class="w-5 h-5 text-orange-500" />
        Aktionen
      </h3>

      <div class="flex flex-wrap gap-3">
        <button
          @click="runNow"
          :disabled="status.running || loading"
          class="px-4 py-2 rounded-lg bg-emerald-500 text-white font-medium hover:bg-emerald-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center gap-2"
        >
          <PlayIcon class="w-4 h-4" />
          Jetzt sammeln
        </button>

        <button
          @click="showBackfillDialog = true"
          :disabled="status.running || loading"
          class="px-4 py-2 rounded-lg bg-blue-500 text-white font-medium hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center gap-2"
        >
          <ArrowPathIcon class="w-4 h-4" />
          Backfill starten
        </button>

        <button
          @click="exportData"
          :disabled="loading"
          class="px-4 py-2 rounded-lg bg-purple-500 text-white font-medium hover:bg-purple-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all flex items-center gap-2"
        >
          <ArrowDownTrayIcon class="w-4 h-4" />
          Exportieren (SQL)
        </button>
      </div>
    </section>

    <!-- Backfill Dialog -->
    <Transition name="modal">
      <div v-if="showBackfillDialog" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showBackfillDialog = false">
        <div class="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-xl">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Backfill starten</h3>

          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Anzahl Tage
            </label>
            <input
              type="number"
              v-model.number="backfillDays"
              min="7"
              max="365"
              class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            />
            <p class="text-xs text-gray-500 mt-1">Historische Daten der letzten X Tage nachholen</p>
          </div>

          <div class="flex justify-end gap-3">
            <button
              @click="showBackfillDialog = false"
              class="px-4 py-2 rounded-lg bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
            >
              Abbrechen
            </button>
            <button
              @click="startBackfill"
              class="px-4 py-2 rounded-lg bg-blue-500 text-white font-medium hover:bg-blue-600"
            >
              Starten
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  ChartBarIcon,
  Cog6ToothIcon,
  PlayIcon,
  ClockIcon,
  ArrowPathIcon,
  DocumentTextIcon,
  ServerIcon,
  BoltIcon,
  ArrowDownTrayIcon
} from '@heroicons/vue/24/outline'

// API Base URL
const API_BASE = import.meta.env.VITE_API_URL || ''

// State
const loading = ref(false)
const status = ref({
  enabled: false,
  running: false,
  collectorStatus: {}
})
const stats = ref({})
const config = ref({
  enabled: false,
  strategy: 'CONSERVATIVE',
  dailyCollectionTime: '06:00',
  autoBackfill: true,
  maxBackfillDays: 365,
  prompt: '',
  activeSources: [],
  activeIndicators: []
})

const showBackfillDialog = ref(false)
const backfillDays = ref(30)

// Source names mapping
const sourceNames = {
  'ECB': 'Europäische Zentralbank',
  'BUNDESBANK': 'Deutsche Bundesbank',
  'ESTR': 'Euro Short-Term Rate'
}

function getSourceName(code) {
  return sourceNames[code] || code
}

function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// Load status and config
async function loadStatus() {
  try {
    const response = await fetch(`${API_BASE}/api/observer/status`)
    if (response.ok) {
      const data = await response.json()
      status.value = data
      stats.value = data.stats || {}
    }
  } catch (error) {
    console.error('Observer status laden fehlgeschlagen:', error)
  }
}

async function loadConfig() {
  try {
    const response = await fetch(`${API_BASE}/api/observer/config`)
    if (response.ok) {
      config.value = await response.json()
    }
  } catch (error) {
    console.error('Observer config laden fehlgeschlagen:', error)
  }
}

// Save config
async function saveConfig() {
  try {
    const response = await fetch(`${API_BASE}/api/observer/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config.value)
    })
    if (response.ok) {
      await loadStatus()
    }
  } catch (error) {
    console.error('Observer config speichern fehlgeschlagen:', error)
  }
}

// Run now
async function runNow() {
  loading.value = true
  try {
    const response = await fetch(`${API_BASE}/api/observer/run`, {
      method: 'POST'
    })
    if (response.ok) {
      const result = await response.json()
      alert(`Sammellauf abgeschlossen: ${result.totalRecords} Datenpunkte gesammelt`)
      await loadStatus()
    } else {
      const error = await response.text()
      alert('Fehler: ' + error)
    }
  } catch (error) {
    console.error('Sammellauf fehlgeschlagen:', error)
    alert('Fehler beim Sammeln: ' + error.message)
  } finally {
    loading.value = false
  }
}

// Start backfill
async function startBackfill() {
  showBackfillDialog.value = false
  loading.value = true
  try {
    const response = await fetch(`${API_BASE}/api/observer/backfill`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ days: backfillDays.value })
    })
    if (response.ok) {
      const result = await response.json()
      alert(`Backfill abgeschlossen: ${result.totalRecords} Datenpunkte gesammelt`)
      await loadStatus()
    } else {
      const error = await response.text()
      alert('Fehler: ' + error)
    }
  } catch (error) {
    console.error('Backfill fehlgeschlagen:', error)
    alert('Fehler beim Backfill: ' + error.message)
  } finally {
    loading.value = false
  }
}

// Export data
async function exportData() {
  try {
    window.open(`${API_BASE}/api/observer/export?format=sql`, '_blank')
  } catch (error) {
    console.error('Export fehlgeschlagen:', error)
  }
}

// Initialize
onMounted(async () => {
  await Promise.all([loadStatus(), loadConfig()])
})
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
