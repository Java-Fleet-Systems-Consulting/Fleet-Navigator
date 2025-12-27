<template>
  <div class="space-y-6">
    <!-- System Prompts Section -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            <DocumentTextIcon class="w-5 h-5 text-blue-500" />
            {{ $t('settings.prompts.management') }}
          </h3>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {{ $t('settings.prompts.managementDesc') }}
          </p>
        </div>
        <button
          @click="$emit('create-prompt')"
          class="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded-lg transition-colors flex items-center gap-2"
        >
          <DocumentDuplicateIcon class="w-4 h-4" />
          {{ $t('settings.prompts.create') }}
        </button>
      </div>

      <!-- Prompts List -->
      <div v-if="systemPrompts.length === 0" class="text-center py-12 text-gray-500 dark:text-gray-400">
        <DocumentTextIcon class="w-16 h-16 mx-auto mb-3 opacity-20" />
        <p class="font-medium">{{ $t('settings.prompts.noPrompts') }}</p>
        <p class="text-xs mt-2">{{ $t('settings.prompts.createFirstHint') }}</p>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="prompt in systemPrompts"
          :key="prompt.id"
          class="border border-gray-200 dark:border-gray-700 rounded-lg p-3 hover:border-blue-400 dark:hover:border-blue-600 transition-colors bg-white dark:bg-gray-800"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <h4 class="font-semibold text-sm text-gray-900 dark:text-white">
                  {{ prompt.name }}
                </h4>
                <span v-if="prompt.isDefault" class="px-1.5 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200 text-xs rounded">
                  {{ $t('settings.prompts.default') }}
                </span>
              </div>
              <p class="text-xs text-gray-600 dark:text-gray-400 line-clamp-2">
                {{ prompt.content }}
              </p>
            </div>
            <div class="flex items-center gap-1">
              <!-- Aktivieren Button -->
              <button
                @click="$emit('activate-prompt', prompt)"
                class="p-1.5 rounded transition-colors"
                :class="prompt.isDefault
                  ? 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30'
                  : 'text-gray-400 dark:text-gray-500 hover:text-green-600 dark:hover:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20'"
                :title="prompt.isDefault ? $t('settings.prompts.activePrompt') : $t('settings.prompts.activateAsDefault')"
              >
                <CheckCircleIcon class="w-4 h-4" />
              </button>
              <button
                @click="$emit('edit-prompt', prompt)"
                class="p-1.5 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
                :title="$t('common.edit')"
              >
                <WrenchScrewdriverIcon class="w-4 h-4" />
              </button>
              <button
                @click="$emit('delete-prompt', prompt.id)"
                class="p-1.5 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                :title="$t('common.delete')"
              >
                <TrashIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- LLM Sampling Parameter Section -->
    <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <AdjustmentsHorizontalIcon class="w-5 h-5 text-orange-500" />
        {{ $t('settings.customModels.samplingTitle') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">
        {{ $t('settings.customModels.samplingDesc') }}
      </p>

      <SimpleSamplingParams v-model="localSamplingParams" @update:modelValue="onSamplingParamsChange" />
    </section>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import {
  DocumentTextIcon,
  DocumentDuplicateIcon,
  CheckCircleIcon,
  WrenchScrewdriverIcon,
  TrashIcon,
  AdjustmentsHorizontalIcon
} from '@heroicons/vue/24/outline'
import SimpleSamplingParams from '../SimpleSamplingParams.vue'

const props = defineProps({
  systemPrompts: {
    type: Array,
    default: () => []
  },
  samplingParams: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits([
  'create-prompt',
  'edit-prompt',
  'delete-prompt',
  'activate-prompt',
  'update:samplingParams'
])

// Local copy
const localSamplingParams = ref({ ...props.samplingParams })

// Sync with parent
watch(() => props.samplingParams, (newVal) => {
  localSamplingParams.value = { ...newVal }
}, { deep: true })

function onSamplingParamsChange(value) {
  emit('update:samplingParams', value)
}
</script>
