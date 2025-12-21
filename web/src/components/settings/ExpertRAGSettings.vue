<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h3 class="text-md font-semibold text-gray-900 dark:text-white flex items-center gap-2">
        <span>📚</span>
        RAG-Quellen (Vektor-Datenbank)
      </h3>
      <span
        v-if="!postgresConnected"
        class="px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200 rounded text-xs"
      >
        PostgreSQL erforderlich
      </span>
    </div>

    <!-- Disabled Overlay wenn kein PostgreSQL -->
    <div
      :class="{ 'opacity-50 pointer-events-none': !postgresConnected }"
      class="space-y-4"
    >
      <!-- RAG aktivieren Toggle -->
      <div class="flex items-center justify-between p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
        <div>
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            RAG aktivieren
          </label>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Ermöglicht dem Experten Zugriff auf dokumentenbasiertes Wissen
          </p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            v-model="ragConfig.enabled"
            :disabled="!postgresConnected"
            class="sr-only peer"
            @change="emitUpdate"
          >
          <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-purple-300 dark:peer-focus:ring-purple-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-purple-600"></div>
        </label>
      </div>

      <!-- RAG Collection Name -->
      <div v-if="ragConfig.enabled">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Collection-Name
        </label>
        <input
          v-model="ragConfig.collectionName"
          type="text"
          :placeholder="`expert_${expertId}_docs`"
          :disabled="!postgresConnected"
          class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50"
          @change="emitUpdate"
        >
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Name der Vektor-Collection in pgvector
        </p>
      </div>

      <!-- Embedding Model -->
      <div v-if="ragConfig.enabled">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Embedding-Modell
        </label>
        <select
          v-model="ragConfig.embeddingModel"
          :disabled="!postgresConnected"
          class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50"
          @change="emitUpdate"
        >
          <option value="nomic-embed-text">nomic-embed-text (768 dim) - Empfohlen</option>
          <option value="mxbai-embed-large">mxbai-embed-large (1024 dim)</option>
          <option value="all-minilm">all-minilm (384 dim) - Schnell</option>
          <option value="snowflake-arctic-embed">snowflake-arctic-embed (1024 dim)</option>
        </select>
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Modell für die Vektor-Embeddings (muss in Ollama installiert sein)
        </p>
      </div>

      <!-- Chunk Size -->
      <div v-if="ragConfig.enabled" class="grid grid-cols-2 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Chunk-Größe
          </label>
          <input
            v-model.number="ragConfig.chunkSize"
            type="number"
            min="100"
            max="2000"
            :disabled="!postgresConnected"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50"
            @change="emitUpdate"
          >
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Zeichen pro Chunk (500-1000 empfohlen)
          </p>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Chunk-Überlappung
          </label>
          <input
            v-model.number="ragConfig.chunkOverlap"
            type="number"
            min="0"
            max="500"
            :disabled="!postgresConnected"
            class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50"
            @change="emitUpdate"
          >
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Überlappung zwischen Chunks (50-200 empfohlen)
          </p>
        </div>
      </div>

      <!-- Top-K Results -->
      <div v-if="ragConfig.enabled">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Anzahl Ergebnisse (Top-K)
        </label>
        <input
          v-model.number="ragConfig.topK"
          type="number"
          min="1"
          max="20"
          :disabled="!postgresConnected"
          class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 focus:border-transparent disabled:opacity-50"
          @change="emitUpdate"
        >
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Wie viele relevante Dokument-Chunks dem Kontext hinzugefügt werden (3-5 empfohlen)
        </p>
      </div>

      <!-- Similarity Threshold -->
      <div v-if="ragConfig.enabled">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Mindest-Ähnlichkeit: {{ (ragConfig.similarityThreshold * 100).toFixed(0) }}%
        </label>
        <input
          v-model.number="ragConfig.similarityThreshold"
          type="range"
          min="0"
          max="1"
          step="0.05"
          :disabled="!postgresConnected"
          class="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700 disabled:opacity-50"
          @change="emitUpdate"
        >
        <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          Nur Ergebnisse mit mindestens dieser Ähnlichkeit werden verwendet (0.7 = 70% empfohlen)
        </p>
      </div>

      <!-- Document Sources -->
      <div v-if="ragConfig.enabled" class="p-4 bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800 rounded-lg">
        <div class="flex items-center justify-between mb-3">
          <h4 class="text-sm font-medium text-purple-800 dark:text-purple-200">
            📄 Dokument-Quellen
          </h4>
          <button
            @click="$emit('open-document-manager')"
            :disabled="!postgresConnected"
            class="px-3 py-1 bg-purple-600 hover:bg-purple-700 text-white rounded text-xs transition-colors disabled:opacity-50"
          >
            Dokumente verwalten
          </button>
        </div>
        <div v-if="ragConfig.documentCount > 0" class="text-sm text-purple-700 dark:text-purple-300">
          {{ ragConfig.documentCount }} Dokumente indexiert
          <span class="text-xs text-purple-600 dark:text-purple-400 ml-2">
            ({{ ragConfig.chunkCount }} Chunks)
          </span>
        </div>
        <div v-else class="text-sm text-purple-600 dark:text-purple-400">
          Keine Dokumente indexiert. Fügen Sie Dokumente hinzu, um RAG zu nutzen.
        </div>
      </div>
    </div>

    <!-- PostgreSQL nicht verbunden Hinweis -->
    <div v-if="!postgresConnected" class="p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
      <div class="flex items-start gap-3">
        <span class="text-lg">🔒</span>
        <div>
          <h4 class="font-semibold text-yellow-800 dark:text-yellow-200 text-sm">
            PostgreSQL-Migration erforderlich
          </h4>
          <p class="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
            RAG-Features benötigen PostgreSQL mit pgvector. Migrieren Sie in den
            <strong>Einstellungen → PostgreSQL Migration</strong>, um diese Features zu aktivieren.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { secureFetch } from '../../utils/secureFetch'

const props = defineProps({
  expertId: {
    type: [Number, String],
    required: true
  },
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'open-document-manager'])

// PostgreSQL Connection Status
const postgresConnected = ref(false)

// RAG Configuration
const ragConfig = ref({
  enabled: false,
  collectionName: '',
  embeddingModel: 'nomic-embed-text',
  chunkSize: 500,
  chunkOverlap: 100,
  topK: 5,
  similarityThreshold: 0.7,
  documentCount: 0,
  chunkCount: 0
})

onMounted(async () => {
  await checkPostgresStatus()
  if (props.modelValue) {
    ragConfig.value = { ...ragConfig.value, ...props.modelValue }
  }
  if (!ragConfig.value.collectionName) {
    ragConfig.value.collectionName = `expert_${props.expertId}_docs`
  }
})

watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    ragConfig.value = { ...ragConfig.value, ...newVal }
  }
}, { deep: true })

async function checkPostgresStatus() {
  try {
    const response = await secureFetch('/api/database/status')
    if (response.ok) {
      const data = await response.json()
      postgresConnected.value = data.database === 'postgres'
    }
  } catch (error) {
    console.debug('Failed to check database status:', error)
    postgresConnected.value = false
  }
}

function emitUpdate() {
  emit('update:modelValue', { ...ragConfig.value })
}

// Expose postgres status for parent
defineExpose({
  isPostgresConnected: () => postgresConnected.value
})
</script>
