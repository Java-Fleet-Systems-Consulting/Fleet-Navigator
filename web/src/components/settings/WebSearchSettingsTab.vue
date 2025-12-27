<template>
  <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
    <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
      <MagnifyingGlassIcon class="w-5 h-5 text-blue-500" />
      {{ $t('settings.webSearch.title') }}
    </h3>

    <!-- Suchzähler Tiles - Brave und SearXNG -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
      <!-- Brave Search API Zähler -->
      <div class="p-4 rounded-xl bg-gradient-to-r from-orange-500/10 to-amber-500/10 border-2 transition-all"
           :class="webSearchSettings.braveConfigured ? 'border-orange-500 dark:border-orange-400 shadow-lg shadow-orange-500/20' : 'border-orange-300/50 dark:border-orange-600/30'">
        <div class="flex items-center gap-3 mb-3">
          <div class="p-2 rounded-lg bg-orange-500/20">
            <StarIcon class="w-6 h-6 text-orange-500" />
          </div>
          <div class="flex-1">
            <h4 class="font-bold text-gray-900 dark:text-white flex items-center gap-2">
              {{ $t('settings.webSearch.braveSearchApi') }}
              <span v-if="webSearchSettings.braveConfigured" class="text-xs px-1.5 py-0.5 bg-orange-500 text-white rounded font-bold animate-pulse">
                {{ $t('settings.webSearch.primary') }}
              </span>
              <span v-else class="text-xs px-1.5 py-0.5 bg-gray-200 dark:bg-gray-700 text-gray-500 dark:text-gray-400 rounded">
                {{ $t('settings.webSearch.notConfigured') }}
              </span>
            </h4>
            <p class="text-xs text-gray-500 dark:text-gray-400">
              {{ webSearchSettings.currentMonth || $t('settings.webSearch.currentMonth') }}
            </p>
          </div>
        </div>
        <div class="text-right mb-2">
          <div class="text-2xl font-bold" :class="searchCountColor">
            {{ webSearchSettings.searchCount || 0 }} / {{ webSearchSettings.searchLimit || 2000 }}
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ webSearchSettings.remainingSearches || 2000 }} {{ $t('settings.webSearch.remaining') }}
          </p>
        </div>
        <!-- Progress Bar -->
        <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div
            class="h-full transition-all duration-500"
            :class="searchCountColor.replace('text-', 'bg-')"
            :style="{ width: searchCountPercent + '%' }"
          ></div>
        </div>
      </div>

      <!-- SearXNG Zähler -->
      <div class="p-4 rounded-xl bg-gradient-to-r from-green-500/10 to-emerald-500/10 border-2 transition-all"
           :class="!webSearchSettings.braveConfigured ? 'border-green-500 dark:border-green-400 shadow-lg shadow-green-500/20' : 'border-green-300/50 dark:border-green-600/30'">
        <div class="flex items-center gap-3 mb-3">
          <div class="p-2 rounded-lg bg-green-500/20">
            <ServerIcon class="w-6 h-6 text-green-500" />
          </div>
          <div class="flex-1">
            <h4 class="font-bold text-gray-900 dark:text-white flex items-center gap-2 flex-wrap">
              {{ $t('settings.webSearch.searxng') }}
              <span v-if="!webSearchSettings.braveConfigured" class="text-xs px-1.5 py-0.5 bg-green-500 text-white rounded font-bold animate-pulse">
                {{ $t('settings.webSearch.primary') }}
              </span>
              <span v-else class="text-xs px-1.5 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded">
                {{ $t('settings.webSearch.fallback') }}
              </span>
            </h4>
            <p class="text-xs text-gray-500 dark:text-gray-400 truncate max-w-[180px]"
               :title="webSearchSettings.customSearxngInstance || $t('settings.webSearch.publicInstances')">
              {{ webSearchSettings.customSearxngInstance || $t('settings.webSearch.publicInstances') }}
            </p>
          </div>
        </div>
        <div class="flex justify-between items-end">
          <div>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('settings.webSearch.thisMonth') }}</p>
            <div class="text-xl font-bold text-green-600 dark:text-green-400">
              {{ webSearchSettings.searxngMonthCount || 0 }}
            </div>
          </div>
          <div class="text-right">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('settings.webSearch.total') }}</p>
            <div class="text-xl font-bold text-green-700 dark:text-green-300">
              {{ webSearchSettings.searxngTotalCount || 0 }}
            </div>
          </div>
        </div>
        <div class="mt-2 text-xs text-green-600 dark:text-green-400 text-center">
          {{ $t('settings.webSearch.noLimit') }}
        </div>
      </div>
    </div>

    <!-- Brave API Key -->
    <div class="mb-6 p-4 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
      <div class="flex items-start gap-3 mb-3">
        <div class="p-2 rounded-lg bg-orange-500/20">
          <StarIcon class="w-5 h-5 text-orange-500" />
        </div>
        <div class="flex-1">
          <h4 class="font-semibold text-gray-900 dark:text-white flex items-center gap-2">
            {{ $t('settings.webSearch.braveSearchApi') }}
            <span v-if="webSearchSettings.braveConfigured" class="text-xs px-2 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-full">
              {{ $t('settings.webSearch.active') }}
            </span>
          </h4>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {{ $t('settings.webSearch.braveApiDescription') }}
          </p>
        </div>
      </div>

      <div class="flex gap-2">
        <input
          v-model="localSettings.braveApiKey"
          type="password"
          :placeholder="$t('settings.webSearch.enterBraveApiKey')"
          class="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-orange-500 focus:border-transparent font-mono text-sm"
        />
        <button
          @click="$emit('test-brave')"
          :disabled="testingSearch"
          class="px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white rounded-xl transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <ArrowPathIcon v-if="testingSearch" class="w-4 h-4 animate-spin" />
          <CheckIcon v-else class="w-4 h-4" />
          {{ $t('settings.webSearch.test') }}
        </button>
      </div>

      <div class="mt-2 flex items-center gap-2">
        <a
          href="https://brave.com/search/api/"
          target="_blank"
          class="text-xs text-blue-600 dark:text-blue-400 hover:underline"
        >
          → {{ $t('settings.webSearch.getApiKey') }}
        </a>
      </div>
    </div>

    <!-- Eigene SearXNG Instanz -->
    <div class="mb-6 p-4 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
      <div class="flex items-start gap-3 mb-3">
        <div class="p-2 rounded-lg bg-green-500/20">
          <ServerIcon class="w-5 h-5 text-green-500" />
        </div>
        <div class="flex-1">
          <h4 class="font-semibold text-gray-900 dark:text-white">
            {{ $t('settings.webSearch.ownSearxngInstance') }}
          </h4>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
            Wird zuerst verwendet (Priorität 1) • Keine Limits • Volle Kontrolle
          </p>
        </div>
      </div>

      <div class="flex gap-2">
        <input
          v-model="localSettings.customSearxngInstance"
          type="url"
          placeholder="https://search.java-fleet.com"
          class="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-green-500 focus:border-transparent text-sm"
        />
        <button
          @click="$emit('test-searxng')"
          :disabled="!localSettings.customSearxngInstance || testingSearch"
          class="px-4 py-2 bg-green-500 hover:bg-green-600 text-white rounded-xl transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <ArrowPathIcon v-if="testingSearch" class="w-4 h-4 animate-spin" />
          <CheckIcon v-else class="w-4 h-4" />
          Test
        </button>
      </div>
    </div>

    <!-- Erweiterte Such-Features -->
    <div class="mb-6 p-4 rounded-xl bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700">
      <h4 class="font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
        <AdjustmentsHorizontalIcon class="w-5 h-5 text-purple-500" />
        Erweiterte Such-Features
      </h4>

      <div class="space-y-4">
        <!-- Query-Optimierung -->
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Query-Optimierung</label>
            <p class="text-xs text-gray-500 dark:text-gray-400">LLM optimiert Suchanfragen für bessere Ergebnisse</p>
          </div>
          <ToggleSwitch v-model="localSettings.queryOptimizationEnabled" color="purple" />
        </div>

        <!-- Content-Scraping -->
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Vollständige Inhalte</label>
            <p class="text-xs text-gray-500 dark:text-gray-400">Lädt Webseiten-Inhalte statt nur Snippets</p>
          </div>
          <ToggleSwitch v-model="localSettings.contentScrapingEnabled" color="blue" />
        </div>

        <!-- Re-Ranking -->
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Re-Ranking</label>
            <p class="text-xs text-gray-500 dark:text-gray-400">Sortiert Ergebnisse nach Relevanz</p>
          </div>
          <ToggleSwitch v-model="localSettings.reRankingEnabled" color="green" />
        </div>

        <!-- Multi-Query -->
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Multi-Query</label>
            <p class="text-xs text-gray-500 dark:text-gray-400">Parallele Suchen mit Query-Variationen (mehr API-Calls)</p>
          </div>
          <ToggleSwitch v-model="localSettings.multiQueryEnabled" color="amber" />
        </div>
      </div>

      <!-- Animation Selector -->
      <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex-1">
            <label class="font-medium text-gray-700 dark:text-gray-200 text-sm flex items-center gap-2">
              🎨 Lade-Animation
            </label>
            <p class="text-xs text-gray-500 dark:text-gray-400">Animation während der Web-Suche</p>
          </div>
          <select
            v-model="localSettings.webSearchAnimation"
            @change="onSettingsChange"
            class="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          >
            <option v-for="opt in animationOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>
      </div>

      <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
        <p class="text-xs text-gray-500 dark:text-gray-400">
          💾 15 Min Cache • 🌐 Sprach-Erkennung automatisch • ⏱️ Zeitfilter verfügbar
        </p>
      </div>
    </div>

    <!-- Info Box -->
    <div class="p-3 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700/50">
      <div class="flex items-start gap-2">
        <InformationCircleIcon class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
        <div class="text-xs text-blue-800 dark:text-blue-200">
          <strong>So funktioniert's:</strong><br>
          Wenn die Web-Suche aktiviert ist (Checkbox im Chat), werden Suchergebnisse
          als Kontext an das KI-Modell übergeben. Das Modell kann dann aktuelle
          Informationen aus dem Web in seine Antwort einbeziehen (RAG).
        </div>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import {
  MagnifyingGlassIcon,
  StarIcon,
  ServerIcon,
  ArrowPathIcon,
  CheckIcon,
  AdjustmentsHorizontalIcon,
  InformationCircleIcon
} from '@heroicons/vue/24/outline'
import ToggleSwitch from '../ToggleSwitch.vue'

const props = defineProps({
  webSearchSettings: {
    type: Object,
    required: true
  },
  testingSearch: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits([
  'update:webSearchSettings',
  'test-brave',
  'test-searxng',
  'save-settings'
])

// Animation options
const animationOptions = [
  { value: 'data-wave', label: '🌊 Data Wave' },
  { value: 'orbit', label: '🌐 Orbiting Network' },
  { value: 'radar', label: '📡 Radar Scan' },
  { value: 'constellation', label: '✨ Constellation' }
]

// Local copy
const localSettings = ref({ ...props.webSearchSettings })

// Sync with parent
watch(() => props.webSearchSettings, (newVal) => {
  localSettings.value = { ...newVal }
}, { deep: true })

watch(localSettings, (newVal) => {
  emit('update:webSearchSettings', newVal)
}, { deep: true })

// Computed
const searchCountColor = computed(() => {
  const percent = (props.webSearchSettings.searchCount / props.webSearchSettings.searchLimit) * 100
  if (percent >= 90) return 'text-red-500'
  if (percent >= 70) return 'text-yellow-500'
  return 'text-green-500'
})

const searchCountPercent = computed(() => {
  return Math.min(100, (props.webSearchSettings.searchCount / props.webSearchSettings.searchLimit) * 100)
})

function onSettingsChange() {
  emit('save-settings')
}
</script>
