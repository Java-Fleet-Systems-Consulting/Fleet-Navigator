<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="fixed inset-0 z-[99999] flex items-center justify-center">
        <!-- Backdrop -->
        <div class="absolute inset-0 bg-black/60 backdrop-blur-sm" @click="$emit('close')"></div>

        <!-- Dialog -->
        <div class="relative bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-md mx-4 overflow-hidden">
          <!-- Header -->
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-fleet-orange-500 to-fleet-orange-600 flex items-center justify-center">
                <InformationCircleIcon class="w-6 h-6 text-white" />
              </div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('infoDialog.title') }}</h2>
            </div>
            <button @click="$emit('close')" class="p-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors">
              <XMarkIcon class="w-5 h-5 text-gray-500" />
            </button>
          </div>

          <!-- Content -->
          <div class="p-6 space-y-4">
            <!-- Logo & Name -->
            <div class="flex items-center gap-4 pb-4 border-b border-gray-200 dark:border-gray-700">
              <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-fleet-orange-500 to-fleet-orange-600 flex items-center justify-center shadow-lg">
                <span class="text-3xl font-bold text-white">FN</span>
              </div>
              <div>
                <h3 class="text-xl font-bold text-gray-900 dark:text-white">Fleet Navigator</h3>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('infoDialog.subtitle') }}</p>
              </div>
            </div>

            <!-- Version Info -->
            <div class="space-y-3">
              <div class="flex justify-between items-center">
                <span class="text-gray-600 dark:text-gray-400">{{ t('infoDialog.version') }}</span>
                <span class="font-mono text-gray-900 dark:text-white bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                  {{ versionInfo.version || '...' }}
                </span>
              </div>
              <div class="flex justify-between items-center">
                <span class="text-gray-600 dark:text-gray-400">{{ t('infoDialog.buildTime') }}</span>
                <span class="font-mono text-sm text-gray-900 dark:text-white">
                  {{ formatBuildTime(versionInfo.buildTime) }}
                </span>
              </div>
            </div>

            <!-- Update Status -->
            <div v-if="updateStatus" class="mt-4 p-3 rounded-lg" :class="updateStatusClass">
              <div class="flex items-center gap-2">
                <CheckCircleIcon v-if="!updateStatus.updateAvailable && !updateStatus.error" class="w-5 h-5 text-green-500" />
                <ArrowDownTrayIcon v-else-if="updateStatus.updateAvailable" class="w-5 h-5 text-blue-500" />
                <ExclamationCircleIcon v-else class="w-5 h-5 text-yellow-500" />
                <span class="text-sm">{{ updateStatus.message }}</span>
              </div>
              <div v-if="updateStatus.updateAvailable" class="mt-2 flex items-center gap-2">
                <span class="text-xs text-gray-500">{{ t('infoDialog.newVersion') }}: {{ updateStatus.latestVersion }}</span>
                <a
                  v-if="updateStatus.releaseURL"
                  :href="updateStatus.releaseURL"
                  target="_blank"
                  class="text-xs text-blue-500 hover:underline"
                >
                  {{ t('infoDialog.viewRelease') }}
                </a>
              </div>
            </div>

            <!-- Check Update Button -->
            <button
              @click="checkForUpdate"
              :disabled="isChecking"
              class="w-full mt-4 px-4 py-3 rounded-xl font-medium transition-all flex items-center justify-center gap-2"
              :class="isChecking
                ? 'bg-gray-200 dark:bg-gray-700 text-gray-500 cursor-not-allowed'
                : 'bg-fleet-orange-500 hover:bg-fleet-orange-600 text-white'"
            >
              <ArrowPathIcon v-if="isChecking" class="w-5 h-5 animate-spin" />
              <ArrowPathIcon v-else class="w-5 h-5" />
              {{ isChecking ? t('infoDialog.checking') : t('infoDialog.checkUpdate') }}
            </button>
          </div>

          <!-- Footer -->
          <div class="px-6 py-4 bg-gray-50 dark:bg-gray-900/50 text-center text-xs text-gray-500 dark:text-gray-400">
            <p>&copy; 2024-2025 JavaFleet Systems Consulting</p>
            <p class="mt-1">{{ t('infoDialog.madeWith') }}</p>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  InformationCircleIcon,
  XMarkIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  ArrowDownTrayIcon,
  ExclamationCircleIcon
} from '@heroicons/vue/24/outline'
import api from '../services/api'

const { t } = useI18n()

defineProps({
  show: {
    type: Boolean,
    default: false
  }
})

defineEmits(['close'])

const versionInfo = ref({
  version: '',
  buildTime: ''
})
const updateStatus = ref(null)
const isChecking = ref(false)

const updateStatusClass = computed(() => {
  if (!updateStatus.value) return ''
  if (updateStatus.value.error) return 'bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800'
  if (updateStatus.value.updateAvailable) return 'bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800'
  return 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
})

function formatBuildTime(buildTime) {
  if (!buildTime || buildTime === 'development') return 'Development Build'
  // Try to format as date
  try {
    const date = new Date(buildTime)
    if (!isNaN(date.getTime())) {
      return date.toLocaleString()
    }
  } catch (e) {
    // Ignore
  }
  return buildTime
}

async function loadVersionInfo() {
  try {
    const response = await api.getSystemVersion()
    versionInfo.value = response
  } catch (err) {
    console.error('Failed to load version info:', err)
  }
}

async function checkForUpdate() {
  isChecking.value = true
  updateStatus.value = null

  try {
    const response = await api.checkForUpdate()
    updateStatus.value = response
  } catch (err) {
    console.error('Update check failed:', err)
    updateStatus.value = {
      updateAvailable: false,
      error: true,
      message: t('infoDialog.checkFailed')
    }
  } finally {
    isChecking.value = false
  }
}

onMounted(() => {
  loadVersionInfo()
})
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: all 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from > div:last-child,
.modal-leave-to > div:last-child {
  transform: scale(0.95);
}
</style>
