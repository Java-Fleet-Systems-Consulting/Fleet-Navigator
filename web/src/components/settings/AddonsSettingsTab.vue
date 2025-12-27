<template>
  <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
      <PuzzlePieceIcon class="w-5 h-5 text-indigo-500" />
      {{ $t('settings.addons.title') || 'Erweiterungen' }}
    </h3>
    <p class="text-sm text-gray-600 dark:text-gray-400 mb-6">
      {{ $t('settings.addons.description') || 'Optionale Komponenten für erweiterte Funktionen' }}
    </p>

    <!-- Tesseract OCR -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 mb-4">
      <div class="flex items-start gap-4">
        <!-- Icon -->
        <div class="p-3 rounded-xl bg-indigo-100 dark:bg-indigo-900/30">
          <DocumentMagnifyingGlassIcon class="w-8 h-8 text-indigo-600 dark:text-indigo-400" />
        </div>

        <!-- Content -->
        <div class="flex-1">
          <div class="flex items-center gap-2 mb-1">
            <h4 class="font-semibold text-gray-900 dark:text-white">Tesseract OCR</h4>
            <span
              v-if="tesseractStatus.installed"
              class="px-2 py-0.5 text-xs bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400 rounded-full"
            >
              {{ $t('common.installed') || 'Installiert' }}
            </span>
            <span
              v-else
              class="px-2 py-0.5 text-xs bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400 rounded-full"
            >
              {{ $t('common.notInstalled') || 'Nicht installiert' }}
            </span>
          </div>
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-3">
            {{ $t('settings.addons.tesseract.description') || 'Ermöglicht Text-Extraktion aus Bildern und gescannten Dokumenten. Ideal für Verträge, Rechnungen und andere Dokumente.' }}
          </p>

          <!-- Installierte Sprachen -->
          <div v-if="tesseractStatus.installed && tesseractStatus.languages?.length > 0" class="mb-3">
            <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
              {{ $t('settings.addons.tesseract.languages') || 'Sprachen' }}:
            </span>
            <div class="flex flex-wrap gap-1 mt-1">
              <span
                v-for="lang in tesseractStatus.languages"
                :key="lang"
                class="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400 rounded"
              >
                {{ lang === 'deu' ? 'Deutsch' : lang === 'eng' ? 'English' : lang === 'tur' ? 'Türkçe' : lang }}
              </span>
            </div>
          </div>

          <!-- Download Progress -->
          <div v-if="tesseractDownloading" class="mb-3">
            <div class="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
              <span>{{ tesseractDownloadMessage }}</span>
              <span>{{ tesseractDownloadProgress }}%</span>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                class="bg-indigo-500 h-2 rounded-full transition-all duration-300"
                :style="{ width: tesseractDownloadProgress + '%' }"
              ></div>
            </div>
          </div>

          <!-- Action Button -->
          <button
            v-if="!tesseractStatus.installed && !tesseractDownloading"
            @click="$emit('download-tesseract')"
            class="px-4 py-2 bg-indigo-500 hover:bg-indigo-600 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <ArrowDownTrayIcon class="w-4 h-4" />
            {{ $t('settings.addons.tesseract.download') || 'Tesseract installieren' }}
          </button>
          <button
            v-else-if="tesseractDownloading"
            disabled
            class="px-4 py-2 bg-gray-400 text-white rounded-lg text-sm font-medium cursor-not-allowed flex items-center gap-2"
          >
            <ArrowPathIcon class="w-4 h-4 animate-spin" />
            {{ $t('common.downloading') || 'Wird heruntergeladen...' }}
          </button>
          <div v-else class="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
            <CheckCircleIcon class="w-5 h-5" />
            {{ $t('settings.addons.tesseract.ready') || 'Tesseract ist einsatzbereit' }}
          </div>
        </div>
      </div>
    </div>

    <!-- PostgreSQL Migration -->
    <div class="bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 p-4 mb-4">
      <PostgreSQLMigration @status-change="onPostgresStatusChange" />
    </div>

    <!-- Info Box -->
    <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
      <div class="flex items-start gap-3">
        <InformationCircleIcon class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
        <div class="text-sm text-blue-800 dark:text-blue-200">
          <p>{{ $t('settings.addons.info') || 'Erweiterungen werden in ~/.fleet-navigator/ gespeichert und können jederzeit installiert werden.' }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import {
  PuzzlePieceIcon,
  DocumentMagnifyingGlassIcon,
  ArrowDownTrayIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  InformationCircleIcon
} from '@heroicons/vue/24/outline'
import PostgreSQLMigration from './PostgreSQLMigration.vue'

const props = defineProps({
  tesseractStatus: {
    type: Object,
    default: () => ({
      installed: false,
      binaryPath: '',
      languages: [],
      dataDir: ''
    })
  },
  tesseractDownloading: {
    type: Boolean,
    default: false
  },
  tesseractDownloadProgress: {
    type: Number,
    default: 0
  },
  tesseractDownloadMessage: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['download-tesseract', 'postgres-status-change'])

function onPostgresStatusChange(connected) {
  emit('postgres-status-change', connected)
}
</script>
