<template>
  <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6">
    <div class="flex items-center justify-between mb-4">
      <div>
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          🐘 PostgreSQL Migration
        </h2>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Migration von SQLite zu PostgreSQL für erweiterte Features (RAG, Vektor-Suche)
        </p>
      </div>
      <span
        :class="[
          'px-3 py-1 rounded-full text-xs font-semibold',
          status.connected
            ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200'
            : 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200'
        ]"
      >
        {{ status.connected ? '✓ Verbunden' : '○ SQLite aktiv' }}
      </span>
    </div>

    <!-- Info Box: PostgreSQL Requirement -->
    <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mb-4">
      <div class="flex items-start gap-3">
        <span class="text-2xl">ℹ️</span>
        <div>
          <h4 class="font-semibold text-blue-800 dark:text-blue-200 mb-1">
            Voraussetzung: PostgreSQL mit pgvector
          </h4>
          <p class="text-sm text-blue-700 dark:text-blue-300">
            Für RAG (Retrieval-Augmented Generation) und Vektor-Suche muss eine PostgreSQL-Datenbank
            mit der <strong>pgvector</strong>-Extension installiert sein.
          </p>
          <ul class="text-xs text-blue-600 dark:text-blue-400 mt-2 space-y-1">
            <li>• PostgreSQL 14+ empfohlen</li>
            <li>• pgvector Extension installiert (<code class="bg-blue-100 dark:bg-blue-900/50 px-1 rounded">CREATE EXTENSION vector;</code>)</li>
            <li>• Datenbank für Fleet Navigator erstellt</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- PostgreSQL Connection Settings -->
    <div class="space-y-4" :class="{ 'opacity-50': status.migrating }">
      <div class="grid grid-cols-2 gap-4">
        <!-- Host -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Host
          </label>
          <input
            v-model="config.host"
            type="text"
            placeholder="localhost"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
        <!-- Port -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Port
          </label>
          <input
            v-model="config.port"
            type="number"
            placeholder="5432"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <!-- Database -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Datenbank
          </label>
          <input
            v-model="config.database"
            type="text"
            placeholder="fleet_navigator"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
        <!-- Schema -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Schema
          </label>
          <input
            v-model="config.schema"
            type="text"
            placeholder="public"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
      </div>

      <div class="grid grid-cols-2 gap-4">
        <!-- Username -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Benutzername
          </label>
          <input
            v-model="config.username"
            type="text"
            placeholder="postgres"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
        <!-- Password -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Passwort
          </label>
          <input
            v-model="config.password"
            type="password"
            placeholder="••••••••"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
        </div>
      </div>

      <!-- SSL Mode -->
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          SSL-Modus
        </label>
        <select
          v-model="config.sslMode"
          class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        >
          <option value="disable">Deaktiviert (lokal)</option>
          <option value="require">Erforderlich</option>
          <option value="verify-ca">CA verifizieren</option>
          <option value="verify-full">Vollständig verifizieren</option>
        </select>
      </div>
    </div>

    <!-- Action Buttons -->
    <div class="flex gap-3 mt-6">
      <button
        @click="testConnection"
        :disabled="status.testing || status.migrating"
        class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
      >
        <span v-if="status.testing" class="animate-spin">⏳</span>
        <span v-else>🔌</span>
        {{ status.testing ? 'Teste...' : 'Verbindung testen' }}
      </button>
      <button
        @click="startMigration"
        :disabled="!status.canMigrate || status.migrating"
        class="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
      >
        <span v-if="status.migrating" class="animate-spin">⏳</span>
        <span v-else>🚀</span>
        {{ status.migrating ? 'Migriere...' : 'Migration starten' }}
      </button>
    </div>

    <!-- Migration Status -->
    <div v-if="status.lastTest" class="mt-4 p-4 rounded-lg" :class="status.lastTest.success ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800' : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'">
      <div class="flex items-center gap-2">
        <span>{{ status.lastTest.success ? '✅' : '❌' }}</span>
        <span :class="status.lastTest.success ? 'text-green-800 dark:text-green-200' : 'text-red-800 dark:text-red-200'">
          {{ status.lastTest.message }}
        </span>
      </div>
      <div v-if="status.lastTest.success && status.lastTest.pgvector" class="mt-2 text-xs text-green-600 dark:text-green-400">
        ✓ pgvector Extension verfügbar (v{{ status.lastTest.pgvectorVersion }})
      </div>
      <div v-else-if="status.lastTest.success && !status.lastTest.pgvector" class="mt-2 text-xs text-yellow-600 dark:text-yellow-400">
        ⚠️ pgvector Extension nicht gefunden - RAG-Features eingeschränkt
      </div>
    </div>

    <!-- Warning: No PostgreSQL = No RAG -->
    <div v-if="!status.connected" class="mt-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
      <div class="flex items-start gap-3">
        <span class="text-xl">⚠️</span>
        <div>
          <h4 class="font-semibold text-yellow-800 dark:text-yellow-200">
            RAG-Features nicht verfügbar
          </h4>
          <p class="text-sm text-yellow-700 dark:text-yellow-300 mt-1">
            Ohne PostgreSQL-Migration sind die RAG-Quellen in den Experten-Einstellungen
            <strong>ausgegraut</strong>. Migrieren Sie zu PostgreSQL, um Vektor-Suche und
            dokumentenbasiertes Wissen für Ihre Experten zu aktivieren.
          </p>
        </div>
      </div>
    </div>

    <!-- Vector Database Settings (nur wenn verbunden und pgvector vorhanden) -->
    <div v-if="status.connected && status.lastTest?.pgvector" class="mt-6 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg p-6">
      <div class="flex items-center gap-3 mb-4">
        <span class="text-2xl">🧠</span>
        <div>
          <h3 class="text-lg font-semibold text-purple-900 dark:text-purple-100">
            Vektor-Datenbank Einstellungen
          </h3>
          <p class="text-sm text-purple-700 dark:text-purple-300">
            Konfiguriere das Embedding-Modell für RAG und semantische Suche
          </p>
        </div>
      </div>

      <!-- Embedding Model Selection -->
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-purple-800 dark:text-purple-200 mb-2">
            Embedding-Modell
          </label>
          <select
            v-model="embeddingConfig.model"
            @change="saveEmbeddingConfig"
            class="w-full px-4 py-2 border border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          >
            <option value="nomic-embed-text">nomic-embed-text (768 dim) - Empfohlen</option>
            <option value="mxbai-embed-large">mxbai-embed-large (1024 dim) - Hohe Qualität</option>
            <option value="all-minilm">all-minilm (384 dim) - Schnell & Kompakt</option>
            <option value="snowflake-arctic-embed">snowflake-arctic-embed (1024 dim) - State-of-Art</option>
          </select>
          <p class="text-xs text-purple-600 dark:text-purple-400 mt-1">
            Das Modell wird für die Vektorisierung von Dokumenten und Suchanfragen verwendet.
          </p>
        </div>

        <!-- Chunk Settings -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-purple-800 dark:text-purple-200 mb-2">
              Chunk-Größe (Tokens)
            </label>
            <input
              v-model.number="embeddingConfig.chunkSize"
              type="number"
              min="128"
              max="2048"
              step="128"
              class="w-full px-4 py-2 border border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            >
          </div>
          <div>
            <label class="block text-sm font-medium text-purple-800 dark:text-purple-200 mb-2">
              Chunk-Überlappung (Tokens)
            </label>
            <input
              v-model.number="embeddingConfig.chunkOverlap"
              type="number"
              min="0"
              max="512"
              step="32"
              class="w-full px-4 py-2 border border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            >
          </div>
        </div>

        <!-- Top-K Results -->
        <div>
          <label class="block text-sm font-medium text-purple-800 dark:text-purple-200 mb-2">
            Top-K Ergebnisse bei Suche
          </label>
          <input
            v-model.number="embeddingConfig.topK"
            type="number"
            min="1"
            max="20"
            class="w-full px-4 py-2 border border-purple-300 dark:border-purple-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent"
          >
          <p class="text-xs text-purple-600 dark:text-purple-400 mt-1">
            Anzahl der relevantesten Dokument-Chunks, die bei einer Suche zurückgegeben werden.
          </p>
        </div>

        <!-- Save Button -->
        <button
          @click="saveEmbeddingConfig"
          :disabled="savingEmbedding"
          class="w-full px-4 py-2 bg-purple-600 hover:bg-purple-700 text-white rounded-lg transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
        >
          <span v-if="savingEmbedding" class="animate-spin">⏳</span>
          <span v-else>💾</span>
          {{ savingEmbedding ? 'Speichere...' : 'Embedding-Einstellungen speichern' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { secureFetch } from '../../utils/secureFetch'

const emit = defineEmits(['status-change'])

// PostgreSQL Configuration
const config = ref({
  host: 'localhost',
  port: 5432,
  database: 'fleet_navigator',
  schema: 'public',
  username: 'postgres',
  password: '',
  sslMode: 'disable'
})

// Status
const status = ref({
  connected: false,
  testing: false,
  migrating: false,
  canMigrate: false,
  lastTest: null
})

// Embedding/Vector DB Configuration
const embeddingConfig = ref({
  model: 'nomic-embed-text',
  chunkSize: 512,
  chunkOverlap: 64,
  topK: 5
})
const savingEmbedding = ref(false)

onMounted(async () => {
  await loadConfig()
  await checkStatus()
  await loadEmbeddingConfig()
})

async function loadConfig() {
  try {
    const response = await secureFetch('/api/database/postgres/config')
    if (response.ok) {
      const data = await response.json()
      if (data.host) {
        config.value = { ...config.value, ...data }
      }
    }
  } catch (error) {
    console.debug('No existing PostgreSQL config:', error)
  }
}

async function checkStatus() {
  try {
    const response = await secureFetch('/api/database/status')
    if (response.ok) {
      const data = await response.json()
      status.value.connected = data.database === 'postgres'
      emit('status-change', status.value.connected)
    }
  } catch (error) {
    console.debug('Failed to check database status:', error)
  }
}

async function testConnection() {
  status.value.testing = true
  status.value.lastTest = null

  try {
    const response = await secureFetch('/api/database/postgres/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config.value)
    })

    const data = await response.json()

    status.value.lastTest = {
      success: data.success,
      message: data.message || (data.success ? 'Verbindung erfolgreich!' : 'Verbindung fehlgeschlagen'),
      pgvector: data.pgvector || false,
      pgvectorVersion: data.pgvectorVersion || null
    }

    status.value.canMigrate = data.success

  } catch (error) {
    status.value.lastTest = {
      success: false,
      message: `Verbindungsfehler: ${error.message}`
    }
    status.value.canMigrate = false
  } finally {
    status.value.testing = false
  }
}

async function startMigration() {
  status.value.migrating = true

  try {
    const response = await secureFetch('/api/database/postgres/migrate', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config.value)
    })

    const data = await response.json()

    if (data.success) {
      status.value.connected = true
      status.value.lastTest = {
        success: true,
        message: 'Migration erfolgreich! Datenbank wurde auf PostgreSQL umgestellt.',
        pgvector: data.pgvector || false,
        pgvectorVersion: data.pgvectorVersion || null
      }
      emit('status-change', true)
    } else {
      status.value.lastTest = {
        success: false,
        message: data.error || 'Migration fehlgeschlagen'
      }
    }

  } catch (error) {
    status.value.lastTest = {
      success: false,
      message: `Migrationsfehler: ${error.message}`
    }
  } finally {
    status.value.migrating = false
  }
}

// Embedding Configuration Functions
async function loadEmbeddingConfig() {
  try {
    const response = await secureFetch('/api/embedding/config')
    if (response.ok) {
      const data = await response.json()
      if (data.model) {
        embeddingConfig.value = { ...embeddingConfig.value, ...data }
      }
    }
  } catch (error) {
    console.debug('No existing embedding config:', error)
  }
}

async function saveEmbeddingConfig() {
  savingEmbedding.value = true
  try {
    const response = await secureFetch('/api/embedding/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(embeddingConfig.value)
    })

    if (response.ok) {
      console.log('Embedding config saved successfully')
    } else {
      console.error('Failed to save embedding config')
    }
  } catch (error) {
    console.error('Error saving embedding config:', error)
  } finally {
    savingEmbedding.value = false
  }
}

// Expose status for parent component
defineExpose({
  isConnected: () => status.value.connected
})
</script>
