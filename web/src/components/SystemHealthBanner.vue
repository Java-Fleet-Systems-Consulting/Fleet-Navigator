<template>
  <div v-if="!health.healthy && health.errors?.length > 0"
       class="bg-red-50 dark:bg-red-900/20 border-l-4 border-red-500 p-4 mb-4">
    <div class="flex items-start">
      <div class="flex-shrink-0">
        <svg class="h-5 w-5 text-red-500" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
      </div>
      <div class="ml-3 flex-1">
        <h3 class="text-sm font-medium text-red-800 dark:text-red-200">
          {{ t('systemHealth.notOperational') }}
        </h3>
        <div class="mt-2 text-sm text-red-700 dark:text-red-300">
          <ul class="list-disc pl-5 space-y-1">
            <li v-for="(error, index) in health.errors" :key="index">
              {{ error }}
            </li>
          </ul>
        </div>

        <!-- Model Installation Instructions -->
        <div v-if="!health.hasModels" class="mt-3 p-3 bg-white dark:bg-gray-800 rounded-md">
          <p class="font-semibold text-gray-900 dark:text-gray-100 mb-2">
            🤖 {{ t('systemHealth.modelInstallation') }}
          </p>
          <div class="text-sm text-gray-700 dark:text-gray-300 space-y-2">
            <p>{{ t('systemHealth.openModelManager') }}</p>
            <p class="font-semibold">{{ t('systemHealth.recommended') }}</p>
            <ul class="list-disc pl-5 space-y-1">
              <li><strong>Qwen 2.5 (3B) - Instruct</strong> - {{ t('systemHealth.qwenDesc') }}</li>
              <li><strong>Phi-3.5-Mini</strong> - {{ t('systemHealth.phiDesc') }}</li>
            </ul>
          </div>
        </div>

        <button
          @click="recheckHealth"
          class="mt-3 px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm rounded-md transition-colors"
        >
          🔄 {{ t('systemHealth.checkAgain') }}
        </button>
      </div>
    </div>
  </div>

  <!-- Warnings (non-critical) - auto-hide after 6 seconds -->
  <div v-else-if="health.warnings?.length > 0 && showWarnings"
       class="bg-yellow-50 dark:bg-yellow-900/20 border-l-4 border-yellow-400 p-4 mb-4 transition-opacity duration-500"
       :class="{ 'opacity-0': fadingOut }">
    <div class="flex items-start">
      <div class="flex-shrink-0">
        <svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
          <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
        </svg>
      </div>
      <div class="ml-3">
        <h3 class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
          {{ t('systemHealth.systemWarnings') }}
        </h3>
        <div class="mt-2 text-sm text-yellow-700 dark:text-yellow-300">
          <ul class="list-disc pl-5 space-y-1">
            <li v-for="(warning, index) in health.warnings" :key="index">
              {{ warning }}
            </li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n()

const health = ref({
  healthy: true,
  hasModels: false,
  sufficientMemory: true,
  warnings: [],
  errors: [],
  summary: ''
})

// Auto-hide warnings after 6 seconds
const showWarnings = ref(true)
const fadingOut = ref(false)
let warningHideTimer = null
let healthCheckInterval = null


// Start auto-hide timer for warnings
const startWarningTimer = () => {
  // Clear existing timer
  if (warningHideTimer) {
    clearTimeout(warningHideTimer)
  }

  // Only start timer if there are warnings
  if (health.value.warnings?.length > 0) {
    showWarnings.value = true
    fadingOut.value = false

    // Start fade-out after 5.5 seconds
    warningHideTimer = setTimeout(() => {
      fadingOut.value = true
      // Hide completely after fade animation (500ms)
      setTimeout(() => {
        showWarnings.value = false
      }, 500)
    }, 5500)
  }
}

// Watch for warning changes to reset timer
watch(() => health.value.warnings?.length ?? 0, (newLen, oldLen) => {
  if (newLen > 0 && newLen !== oldLen) {
    startWarningTimer()
  } else if (newLen === 0) {
    showWarnings.value = true
    fadingOut.value = false
  }
})

const checkHealth = async () => {
  try {
    const response = await api.getSystemHealth()
    // Ensure errors and warnings are always arrays
    health.value = {
      ...response,
      errors: response.errors || [],
      warnings: response.warnings || []
    }
  } catch (error) {
    console.error('Failed to check system health:', error)
    health.value.errors = ['Failed to connect to backend']
  }
}

const recheckHealth = async () => {
  await checkHealth()
}

onMounted(() => {
  checkHealth()
  // Start warning timer after initial check
  setTimeout(startWarningTimer, 100)
  // Periodischer Health-Check alle 15 Sekunden
  healthCheckInterval = setInterval(checkHealth, 15000)
})

onUnmounted(() => {
  if (healthCheckInterval) {
    clearInterval(healthCheckInterval)
  }
  if (warningHideTimer) {
    clearTimeout(warningHideTimer)
  }
})
</script>
