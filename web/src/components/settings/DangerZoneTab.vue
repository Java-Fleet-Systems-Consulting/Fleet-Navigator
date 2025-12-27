<template>
  <section class="danger-zone-section bg-gradient-to-br from-red-50 to-red-100 dark:from-red-900/30 dark:to-red-800/30 p-6 rounded-xl border-2 border-red-300 dark:border-red-700 shadow-lg">
    <div class="flex items-start gap-3 mb-6">
      <ShieldExclamationIcon class="w-8 h-8 text-red-600 dark:text-red-400 flex-shrink-0" />
      <div>
        <h3 class="text-xl font-bold text-red-900 dark:text-red-100 mb-2">
          ⚠️ {{ $t('settings.danger.title') }}
        </h3>
        <p class="text-sm text-red-800 dark:text-red-200" v-html="$t('settings.danger.warningPermanent')">
        </p>
      </div>
    </div>

    <!-- Selective Data Reset -->
    <div class="bg-white/80 dark:bg-gray-900/80 p-5 rounded-xl border-2 border-red-400 dark:border-red-600">
      <div class="flex items-start gap-3 mb-4">
        <TrashIcon class="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0 mt-1" />
        <div class="flex-1">
          <h4 class="text-lg font-bold text-gray-900 dark:text-white mb-2">
            {{ $t('settings.danger.selectiveDelete') }}
          </h4>
          <p class="text-sm text-gray-700 dark:text-gray-300 mb-4">
            {{ $t('settings.danger.selectDataPrompt') }}
          </p>

          <!-- Checkboxes for selective deletion -->
          <div class="space-y-3 mb-4">
            <!-- Chats & Messages -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.chats"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.chats') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.chatsDesc') }}</div>
              </div>
            </label>

            <!-- Projects & Files -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.projects"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.projects') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.projectsDesc') }}</div>
              </div>
            </label>

            <!-- Custom Models -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors border-2 border-orange-300 dark:border-orange-700">
              <input
                type="checkbox"
                v-model="localResetSelection.customModels"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
                  {{ $t('settings.danger.customModels') }}
                  <span class="text-xs bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-200 px-2 py-0.5 rounded">{{ $t('settings.danger.optional') }}</span>
                </div>
                <div class="text-xs text-gray-600 dark:text-gray-400">
                  {{ $t('settings.danger.customModelsDesc') }}
                </div>
              </div>
            </label>

            <!-- Settings -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.settings"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.settingsConfig') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.settingsConfigDesc') }}</div>
              </div>
            </label>

            <!-- Personal Info -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.personalInfo"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.personalInfo') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.personalInfoDesc') }}</div>
              </div>
            </label>

            <!-- Templates -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.templates"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.templates') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.templatesDesc') }}</div>
              </div>
            </label>

            <!-- Statistics -->
            <label class="flex items-start gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 cursor-pointer transition-colors">
              <input
                type="checkbox"
                v-model="localResetSelection.stats"
                class="mt-1 w-4 h-4 text-red-600 rounded focus:ring-red-500"
              />
              <div class="flex-1">
                <div class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.danger.stats') }}</div>
                <div class="text-xs text-gray-600 dark:text-gray-400">{{ $t('settings.danger.statsDesc') }}</div>
              </div>
            </label>
          </div>

          <div class="p-3 bg-yellow-100 dark:bg-yellow-900/30 border border-yellow-400 dark:border-yellow-700 rounded-lg mb-4">
            <p class="text-xs text-yellow-900 dark:text-yellow-200 flex items-start gap-2">
              <ExclamationTriangleIcon class="w-4 h-4 flex-shrink-0 mt-0.5" />
              <span>
                <strong>{{ $t('common.note') }}:</strong> {{ $t('settings.danger.noteAppReload') }}
              </span>
            </p>
          </div>
        </div>
      </div>

      <button
        @click="handleResetAll"
        :disabled="resetting || !hasAnySelection"
        :title="hasAnySelection ? $t('settings.danger.deleteSelectedTitle') : $t('settings.danger.selectCategoryFirst')"
        class="w-full px-6 py-3 rounded-xl bg-gradient-to-r from-red-600 to-red-700 hover:from-red-700 hover:to-red-800 text-white font-bold shadow-lg hover:shadow-xl transition-all transform hover:scale-105 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
      >
        <TrashIcon v-if="!resetting" class="w-5 h-5" />
        <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
        {{ resetting ? $t('settings.danger.deleting') : (hasAnySelection ? $t('settings.danger.deleteSelected') : $t('settings.danger.noSelection')) }}
      </button>
    </div>
  </section>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import {
  ShieldExclamationIcon,
  TrashIcon,
  ArrowPathIcon,
  ExclamationTriangleIcon
} from '@heroicons/vue/24/outline'

const props = defineProps({
  resetSelection: {
    type: Object,
    default: () => ({
      chats: true,
      projects: true,
      customModels: false,
      settings: true,
      personalInfo: true,
      templates: true,
      stats: true
    })
  },
  resetting: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:resetSelection', 'reset-all'])

// Local copy
const localResetSelection = ref({ ...props.resetSelection })

// Sync with parent
watch(() => props.resetSelection, (newVal) => {
  localResetSelection.value = { ...newVal }
}, { deep: true })

watch(localResetSelection, (newVal) => {
  emit('update:resetSelection', newVal)
}, { deep: true })

// Computed
const hasAnySelection = computed(() => {
  return Object.values(localResetSelection.value).some(v => v === true)
})

// Methods
function handleResetAll() {
  emit('reset-all', localResetSelection.value)
}
</script>
