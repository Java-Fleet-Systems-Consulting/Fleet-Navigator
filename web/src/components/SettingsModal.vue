<template>
  <Transition name="modal">
    <div v-if="isOpen" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4" @click.self="close">
      <div class="bg-white/95 dark:bg-gray-800/95 backdrop-blur-xl rounded-2xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden border border-gray-200/50 dark:border-gray-700/50">
        <!-- Header with Gradient -->
        <div class="sticky top-0 bg-gradient-to-r from-fleet-orange-500/10 to-orange-500/10 dark:from-fleet-orange-500/20 dark:to-orange-500/20 backdrop-blur-sm border-b border-gray-200/50 dark:border-gray-700/50 px-6 py-4 flex justify-between items-center z-10">
          <div class="flex items-center gap-3">
            <div class="p-2 rounded-xl bg-gradient-to-br from-fleet-orange-500 to-orange-600 shadow-lg">
              <Cog6ToothIcon class="w-6 h-6 text-white" />
            </div>
            <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ $t('settings.title') }}</h2>
          </div>
          <button
            @click="close"
            class="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all transform hover:scale-110"
          >
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <!-- Tab Navigation -->
        <div class="flex flex-wrap border-b border-gray-200 dark:border-gray-700 px-6 bg-gray-50/50 dark:bg-gray-900/50 gap-1">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            @click="activeTab = tab.id"
            class="flex items-center gap-2 px-3 py-2 text-sm font-medium transition-all relative whitespace-nowrap rounded-lg"
            :class="activeTab === tab.id
              ? 'text-fleet-orange-600 dark:text-fleet-orange-400'
              : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200'"
          >
            <component :is="tab.icon" class="w-4 h-4" />
            {{ tab.label }}
            <div
              v-if="activeTab === tab.id"
              class="absolute bottom-0 left-0 right-0 h-0.5 bg-fleet-orange-500"
            />
          </button>
        </div>

        <!-- Content with Custom Scrollbar -->
        <div class="overflow-y-auto p-6 space-y-6 custom-scrollbar" style="max-height: calc(90vh - 220px);">

          <!-- TAB: General Settings -->
          <div v-if="activeTab === 'general'">
            <GeneralSettingsTab
              :settings="settings"
              :voice-download-info="voiceDownloadInfo"
              :is-downloading-voices="isDownloadingVoices"
              @update:settings="settings = $event"
              @language-change="onLanguageChange"
              @download-voices="downloadVoicesForLanguage"
            />
          </div>

          <!-- TAB: Fleet Mates -->
          <div v-if="activeTab === 'mates'">
            <MatesSettingsTab
              :trusted-mates="trustedMates"
              :pending-pairing-requests="pendingPairingRequests"
              :processing-pairing="processingPairing"
              :removing-mate-id="removingMateId"
              :forgetting-mates="forgettingMates"
              :mate-models="mateModels"
              :fast-models="fastModels"
              :available-models="availableModels"
              @approve-pairing="approvePairingRequest"
              @reject-pairing="rejectPairingRequest"
              @remove-mate="removeTrustedMate"
              @forget-all-mates="forgetAllMates"
              @save-email-model="saveEmailModel"
              @save-document-model="saveDocumentModel"
              @save-log-analysis-model="saveLogAnalysisModel"
              @save-coder-model="saveCoderModel"
            />
          </div>

          <!-- TAB: LLM Provider -->
          <div v-if="activeTab === 'providers'">
            <ProviderSettings />
          </div>

          <!-- TAB: Custom Modell (System-Prompts + Sampling) -->
          <div v-if="activeTab === 'customModels'">
            <CustomModelsTab
              :system-prompts="systemPrompts"
              :sampling-params="samplingParams"
              @create-prompt="showPromptEditor = true; editingPrompt = null; resetPromptForm()"
              @edit-prompt="editSystemPrompt"
              @delete-prompt="deleteSystemPrompt"
              @activate-prompt="activateSystemPrompt"
              @update:sampling-params="samplingParams = $event"
            />
          </div>

          <!-- TAB: Personal Info -->
          <div v-if="activeTab === 'personal'">
            <PersonalInfoTab ref="personalInfoTabRef" />
          </div>

          <!-- TAB: Observer (Finanz- und Wirtschaftsdaten) -->
          <div v-if="activeTab === 'observer'">
            <ObserverSettings />
          </div>

          <!-- TAB: Agents -->
          <div v-if="activeTab === 'agents'">
            <AgentsSettingsTab
              :settings="settings"
              :model-selection-settings="modelSelectionSettings"
              :web-search-think-first="webSearchThinkFirst"
              :vision-models="visionModels"
              :file-search-folders="fileSearchFolders"
              :file-search-status="fileSearchStatus"
              @update:settings="settings = $event"
              @update:model-selection-settings="modelSelectionSettings = $event"
              @update:web-search-think-first="webSearchThinkFirst = $event"
              @add-folder="addSearchFolder"
              @remove-folder="removeSearchFolder"
              @reindex-folder="reindexFolder"
            />
          </div>

          <!-- TAB: Web Search Settings -->
          <div v-if="activeTab === 'web-search'">
            <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                <MagnifyingGlassIcon class="w-5 h-5 text-blue-500" />
                {{ $t('settings.webSearch.title') }}
              </h3>

              <!-- Suchzähler Tiles - Brave und SearXNG -->
              <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                <!-- Brave Search API Zähler -->
                <div class="p-4 rounded-xl bg-gradient-to-r from-orange-500/10 to-amber-500/10 border-2 transition-all" :class="webSearchSettings.braveConfigured ? 'border-orange-500 dark:border-orange-400 shadow-lg shadow-orange-500/20' : 'border-orange-300/50 dark:border-orange-600/30'">
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
                <div class="p-4 rounded-xl bg-gradient-to-r from-green-500/10 to-emerald-500/10 border-2 transition-all" :class="!webSearchSettings.braveConfigured ? 'border-green-500 dark:border-green-400 shadow-lg shadow-green-500/20' : 'border-green-300/50 dark:border-green-600/30'">
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
                        <span v-if="webSearchSettings.customSearxngInstance" class="text-xs px-1.5 py-0.5 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded">
                          {{ $t('settings.webSearch.ownInstance') }}
                        </span>
                      </h4>
                      <p class="text-xs text-gray-500 dark:text-gray-400 truncate max-w-[180px]" :title="webSearchSettings.customSearxngInstance || $t('settings.webSearch.publicInstances')">
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
                  <!-- Kein Limit Hinweis -->
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
                      <span v-else class="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-500 rounded-full">
                        {{ $t('settings.webSearch.notConfigured') }}
                      </span>
                    </h4>
                    <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {{ $t('settings.webSearch.braveApiDescription') }}
                    </p>
                  </div>
                </div>

                <div class="flex gap-2">
                  <input
                    v-model="webSearchSettings.braveApiKey"
                    type="password"
                    :placeholder="$t('settings.webSearch.enterBraveApiKey')"
                    class="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-orange-500 focus:border-transparent font-mono text-sm"
                  />
                  <button
                    @click="testBraveSearch"
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

              <!-- Eigene SearXNG Instanz (Priorität 1) -->
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
                    v-model="webSearchSettings.customSearxngInstance"
                    type="url"
                    placeholder="https://search.java-fleet.com"
                    class="flex-1 px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-green-500 focus:border-transparent text-sm"
                  />
                  <button
                    @click="testCustomSearxng"
                    :disabled="!webSearchSettings.customSearxngInstance || testingSearch"
                    class="px-4 py-2 bg-green-500 hover:bg-green-600 text-white rounded-xl transition-colors disabled:opacity-50 flex items-center gap-2"
                  >
                    <ArrowPathIcon v-if="testingSearch" class="w-4 h-4 animate-spin" />
                    <CheckIcon v-else class="w-4 h-4" />
                    Test
                  </button>
                </div>
              </div>

              <!-- SearXNG Fallback-Instanzen (editierbar) -->
              <details class="mb-4">
                <summary class="cursor-pointer text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white flex items-center gap-2">
                  <span>▸ Öffentliche Fallback-Instanzen</span>
                  <span class="text-xs bg-gray-200 dark:bg-gray-700 px-2 py-0.5 rounded">
                    {{ webSearchSettings.searxngInstances?.length || 0 }}
                  </span>
                </summary>
                <div class="mt-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
                  <div class="space-y-2 max-h-48 overflow-y-auto custom-scrollbar">
                    <div
                      v-for="(instance, index) in webSearchSettings.searxngInstances"
                      :key="index"
                      class="flex items-center gap-2"
                    >
                      <span class="text-xs text-gray-400 w-5">{{ index + 1 }}.</span>
                      <input
                        v-model="webSearchSettings.searxngInstances[index]"
                        type="url"
                        class="flex-1 px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      />
                      <button
                        @click="removeSearxngInstance(index)"
                        class="p-1.5 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors"
                        title="Entfernen"
                      >
                        <TrashIcon class="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                  <div class="mt-3 flex gap-2">
                    <button
                      @click="addSearxngInstance"
                      class="px-3 py-1.5 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors flex items-center gap-1"
                    >
                      <PlusIcon class="w-4 h-4" />
                      Hinzufügen
                    </button>
                    <button
                      @click="resetSearxngInstances"
                      class="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
                    >
                      Zurücksetzen
                    </button>
                  </div>
                </div>
              </details>

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
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="webSearchSettings.queryOptimizationEnabled" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-purple-300 dark:peer-focus:ring-purple-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-purple-600"></div>
                    </label>
                  </div>

                  <!-- Modell für Query-Optimierung -->
                  <div v-if="webSearchSettings.queryOptimizationEnabled" class="ml-4 pl-4 border-l-2 border-purple-200 dark:border-purple-700">
                    <label class="block text-xs font-medium text-gray-600 dark:text-gray-400 mb-1">Optimierungs-Modell</label>
                    <select
                      v-model="webSearchSettings.queryOptimizationModel"
                      class="w-full px-3 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-white"
                    >
                      <option v-if="smallModels.length === 0" value="">Keine kleinen Modelle gefunden</option>
                      <option v-for="model in smallModels" :key="model" :value="model">
                        {{ model }}
                      </option>
                      <!-- Fallback: Alle Modelle wenn keine kleinen gefunden -->
                      <optgroup v-if="smallModels.length > 0 && availableModels.length > smallModels.length" label="── Alle Modelle ──">
                        <option v-for="model in availableModels" :key="'all-' + model.name" :value="model.name">
                          {{ model.name }}
                        </option>
                      </optgroup>
                    </select>

                    <!-- Effektives Modell Status -->
                    <div class="mt-2 text-xs">
                      <!-- Warnung: Konfiguriertes Modell nicht verfügbar -->
                      <div v-if="webSearchSettings.effectiveOptimizationModel && webSearchSettings.effectiveOptimizationModel !== webSearchSettings.queryOptimizationModel"
                           class="flex items-center gap-1.5 text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 px-2 py-1.5 rounded-lg">
                        <svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z" clip-rule="evenodd"/>
                        </svg>
                        <span>
                          <strong>{{ webSearchSettings.queryOptimizationModel }}</strong> nicht verfügbar.
                          Verwende: <strong class="text-green-600 dark:text-green-400">{{ webSearchSettings.effectiveOptimizationModel }}</strong>
                        </span>
                      </div>
                      <!-- Info: Modell nicht verfügbar, Optimierung deaktiviert -->
                      <div v-else-if="!webSearchSettings.effectiveOptimizationModel"
                           class="flex items-center gap-1.5 text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 px-2 py-1.5 rounded-lg">
                        <svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.28 7.22a.75.75 0 00-1.06 1.06L8.94 10l-1.72 1.72a.75.75 0 101.06 1.06L10 11.06l1.72 1.72a.75.75 0 101.06-1.06L11.06 10l1.72-1.72a.75.75 0 00-1.06-1.06L10 8.94 8.28 7.22z" clip-rule="evenodd"/>
                        </svg>
                        <span>Kein Modell verfügbar - Query-Optimierung deaktiviert</span>
                      </div>
                      <!-- OK: Konfiguriertes Modell wird verwendet -->
                      <div v-else class="flex items-center gap-1.5 text-green-600 dark:text-green-400">
                        <svg class="w-4 h-4 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.857-9.809a.75.75 0 00-1.214-.882l-3.483 4.79-1.88-1.88a.75.75 0 10-1.06 1.061l2.5 2.5a.75.75 0 001.137-.089l4-5.5z" clip-rule="evenodd"/>
                        </svg>
                        <span>Modell aktiv: <strong>{{ webSearchSettings.effectiveOptimizationModel }}</strong></span>
                      </div>
                    </div>

                    <p class="text-xs text-gray-400 mt-1">
                      <span v-if="smallModels.length > 0">{{ smallModels.length }} kleine Modelle (1B-7B) verfügbar</span>
                      <span v-else>Lade Modelle oder installiere ein kleines Modell (z.B. llama3.2:3b)</span>
                    </p>
                  </div>

                  <!-- Content-Scraping -->
                  <div class="flex items-center justify-between">
                    <div class="flex-1">
                      <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Vollständige Inhalte</label>
                      <p class="text-xs text-gray-500 dark:text-gray-400">Lädt Webseiten-Inhalte statt nur Snippets</p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="webSearchSettings.contentScrapingEnabled" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-blue-600"></div>
                    </label>
                  </div>

                  <!-- Re-Ranking -->
                  <div class="flex items-center justify-between">
                    <div class="flex-1">
                      <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Re-Ranking</label>
                      <p class="text-xs text-gray-500 dark:text-gray-400">Sortiert Ergebnisse nach Relevanz</p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="webSearchSettings.reRankingEnabled" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-green-300 dark:peer-focus:ring-green-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-green-600"></div>
                    </label>
                  </div>

                  <!-- Multi-Query -->
                  <div class="flex items-center justify-between">
                    <div class="flex-1">
                      <label class="font-medium text-gray-700 dark:text-gray-200 text-sm">Multi-Query</label>
                      <p class="text-xs text-gray-500 dark:text-gray-400">Parallele Suchen mit Query-Variationen (mehr API-Calls)</p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="webSearchSettings.multiQueryEnabled" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-amber-300 dark:peer-focus:ring-amber-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-amber-600"></div>
                    </label>
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
                      v-model="webSearchSettings.webSearchAnimation"
                      @change="saveWebSearchSettings"
                      class="bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                    >
                      <option v-for="opt in animationOptions" :key="opt.value" :value="opt.value">
                        {{ opt.label }}
                      </option>
                    </select>
                  </div>
                  <!-- Animation Preview -->
                  <div class="mt-3 p-3 rounded-lg bg-gray-900/80 border border-gray-700">
                    <p class="text-xs text-gray-400 mb-2">Vorschau:</p>
                    <div class="flex items-center gap-3">
                      <div class="p-2 rounded-lg bg-gradient-to-br from-cyan-500 via-blue-500 to-purple-600 shadow-lg">
                        <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
                        </svg>
                      </div>
                      <div class="flex-1">
                        <!-- Data Wave Preview -->
                        <div v-if="webSearchSettings.webSearchAnimation === 'data-wave'" class="h-8 relative overflow-hidden rounded-lg bg-gradient-to-r from-cyan-500/10 via-blue-500/15 to-purple-500/10 border border-blue-500/20">
                          <div class="absolute inset-0 flex items-center">
                            <div class="w-full h-0.5 bg-gradient-to-r from-transparent via-blue-500 to-transparent animate-pulse"></div>
                          </div>
                          <div class="absolute inset-0 flex items-center justify-around">
                            <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-ping"></span>
                            <span class="w-2 h-2 rounded-full bg-blue-500 animate-ping" style="animation-delay: 0.3s"></span>
                            <span class="w-1.5 h-1.5 rounded-full bg-purple-400 animate-ping" style="animation-delay: 0.6s"></span>
                          </div>
                        </div>
                        <!-- Orbit Preview -->
                        <div v-else-if="webSearchSettings.webSearchAnimation === 'orbit'" class="h-8 relative overflow-hidden rounded-lg bg-gradient-to-r from-blue-500/10 to-indigo-500/10 border border-indigo-500/20 flex items-center justify-center">
                          <div class="w-6 h-6 rounded-full border-2 border-indigo-400/50 relative">
                            <span class="absolute w-2 h-2 rounded-full bg-indigo-500 animate-spin" style="animation-duration: 1.5s; top: -4px; left: 50%; transform: translateX(-50%);"></span>
                          </div>
                        </div>
                        <!-- Radar Preview -->
                        <div v-else-if="webSearchSettings.webSearchAnimation === 'radar'" class="h-8 relative overflow-hidden rounded-lg bg-gradient-to-r from-green-500/10 to-emerald-500/10 border border-green-500/20 flex items-center justify-center">
                          <div class="w-6 h-6 rounded-full border border-green-500/30 flex items-center justify-center">
                            <div class="w-4 h-4 rounded-full border border-green-400/50 animate-ping"></div>
                          </div>
                        </div>
                        <!-- Constellation Preview -->
                        <div v-else-if="webSearchSettings.webSearchAnimation === 'constellation'" class="h-8 relative overflow-hidden rounded-lg bg-gradient-to-r from-violet-500/10 to-pink-500/10 border border-violet-500/20">
                          <div class="absolute inset-0 flex items-center justify-around">
                            <span class="w-1 h-1 rounded-full bg-violet-400 animate-pulse"></span>
                            <span class="w-1.5 h-1.5 rounded-full bg-pink-400 animate-pulse" style="animation-delay: 0.2s"></span>
                            <span class="w-1 h-1 rounded-full bg-violet-300 animate-pulse" style="animation-delay: 0.4s"></span>
                            <span class="w-2 h-2 rounded-full bg-pink-500 animate-pulse" style="animation-delay: 0.1s"></span>
                            <span class="w-1 h-1 rounded-full bg-violet-400 animate-pulse" style="animation-delay: 0.3s"></span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Info -->
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
          </div>

          <!-- TAB: Voice (STT/TTS) -->
          <div v-if="activeTab === 'voice'">
            <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                <MicrophoneIcon class="w-5 h-5 text-purple-500" />
                {{ $t('settings.voice.title') }}
              </h3>

              <!-- TTS Global Toggle -->
              <div class="mb-4 p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <SpeakerWaveIcon class="w-5 h-5 text-indigo-500" />
                    <div>
                      <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.ttsEnabled') }}</span>
                      <p class="text-xs text-gray-500 dark:text-gray-400">{{ $t('settings.voice.ttsEnabledDesc') }}</p>
                    </div>
                  </div>
                  <button
                    @click="toggleTtsEnabled"
                    :class="[
                      'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                      ttsEnabled ? 'bg-indigo-500' : 'bg-gray-300 dark:bg-gray-600'
                    ]"
                  >
                    <span
                      :class="[
                        'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                        ttsEnabled ? 'translate-x-6' : 'translate-x-1'
                      ]"
                    />
                  </button>
                </div>
              </div>

              <!-- Download Progress (global) -->
              <div v-if="voiceDownloading" class="mb-4 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
                <div class="flex items-center gap-3 mb-2">
                  <div class="animate-spin rounded-full h-5 w-5 border-2 border-blue-500 border-t-transparent"></div>
                  <span class="font-medium text-blue-700 dark:text-blue-300">{{ voiceDownloadStatus }}</span>
                </div>
                <div v-if="voiceDownloadProgress > 0" class="w-full bg-blue-200 dark:bg-blue-800 rounded-full h-2">
                  <div class="bg-blue-500 h-2 rounded-full transition-all" :style="{ width: voiceDownloadProgress + '%' }"></div>
                </div>
                <p v-if="voiceDownloadSpeed" class="text-xs text-blue-600 dark:text-blue-400 mt-1">{{ voiceDownloadSpeed }}</p>
              </div>

              <!-- Whisper STT Section -->
              <div class="mb-6 p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                <div class="flex items-center gap-2 mb-3">
                  <MicrophoneIcon class="w-5 h-5 text-purple-500" />
                  <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.whisperStt') }}</span>
                  <span v-if="voiceModels.whisperBinary" class="ml-auto text-xs px-2 py-0.5 rounded-full bg-green-200 dark:bg-green-800 text-green-800 dark:text-green-200">{{ $t('settings.voice.ready') }}</span>
                  <span v-else class="ml-auto text-xs px-2 py-0.5 rounded-full bg-red-200 dark:bg-red-800 text-red-800 dark:text-red-200">{{ $t('settings.voice.notInstalled') }}</span>
                </div>

                <!-- Hinweis wenn Binary fehlt -->
                <div v-if="!voiceModels.whisperBinary" class="mb-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
                  <p class="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
                    <strong>{{ $t('settings.voice.whisperNotInstalled') }}</strong> {{ $t('settings.voice.whisperInstallHint') }}
                  </p>
                  <button
                    @click="downloadWhisper"
                    :disabled="voiceDownloading"
                    class="w-full px-4 py-2 rounded-lg bg-purple-500 hover:bg-purple-600 text-white font-medium disabled:opacity-50 flex flex-col items-center justify-center gap-1"
                  >
                    <div class="flex items-center gap-2">
                      <div v-if="voiceDownloadComponent === 'whisper'" class="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                      <ArrowDownTrayIcon v-else class="w-5 h-5" />
                      {{ voiceDownloadComponent === 'whisper' ? $t('settings.voice.installing') : $t('settings.voice.installWhisper') }}
                    </div>
                    <!-- Progress Bar nur wenn Whisper heruntergeladen wird -->
                    <div v-if="voiceDownloadComponent === 'whisper' && voiceDownloadProgress > 0" class="w-full mt-1">
                      <div class="w-full bg-purple-300 rounded-full h-1.5">
                        <div class="bg-white h-1.5 rounded-full transition-all" :style="{ width: voiceDownloadProgress + '%' }"></div>
                      </div>
                      <span class="text-xs">{{ Math.round(voiceDownloadProgress) }}%</span>
                    </div>
                  </button>
                  <p class="text-xs text-yellow-600 dark:text-yellow-400 mt-2">
                    {{ $t('settings.voice.whisperDownloadInfo') }}
                  </p>
                </div>

                <!-- Nur anzeigen wenn Binary vorhanden -->
                <template v-if="voiceModels.whisperBinary">
                  <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ $t('settings.voice.selectModelHint') }}</p>

                  <!-- Whisper Models Grid -->
                  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                    <div
                      v-for="model in voiceModels.whisper"
                      :key="model.id"
                      class="p-3 rounded-lg border-2 cursor-pointer transition-all"
                      :class="[
                        voiceModels.currentWhisper === model.id
                          ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20'
                          : model.installed
                            ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/10 hover:border-green-400'
                            : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                      ]"
                      @click="model.installed && selectWhisperModel(model.id)"
                    >
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-medium text-gray-900 dark:text-white">{{ model.name }}</span>
                        <div class="flex items-center gap-1">
                          <span v-if="voiceModels.currentWhisper === model.id" class="text-xs px-1.5 py-0.5 rounded bg-purple-500 text-white">{{ $t('settings.voice.active') }}</span>
                          <span v-else-if="model.installed" class="text-xs px-1.5 py-0.5 rounded bg-green-500 text-white">{{ $t('settings.voice.installed') }}</span>
                        </div>
                      </div>
                      <p class="text-xs text-gray-500 dark:text-gray-400 mb-2">{{ model.description }}</p>
                      <div class="flex items-center justify-between">
                        <span class="text-xs text-gray-400">{{ model.sizeMB }} MB</span>
                        <button
                          v-if="!model.installed"
                          @click.stop="downloadSpecificModel('whisper', model.id)"
                          :disabled="voiceDownloading"
                          class="text-xs px-2 py-1 rounded bg-purple-500 hover:bg-purple-600 text-white disabled:opacity-50"
                        >
                          {{ $t('settings.voice.download') }}
                        </button>
                        <button
                          v-else-if="voiceModels.currentWhisper !== model.id"
                          @click.stop="selectWhisperModel(model.id)"
                          class="text-xs px-2 py-1 rounded bg-gray-200 dark:bg-gray-600 hover:bg-gray-300 dark:hover:bg-gray-500 text-gray-700 dark:text-gray-200"
                        >
                          {{ $t('settings.voice.activate') }}
                        </button>
                      </div>
                    </div>
                  </div>
                </template>

                <!-- Vorschau wenn Binary fehlt -->
                <div v-else class="opacity-50 pointer-events-none">
                  <p class="text-sm text-gray-500 mb-2">{{ $t('settings.voice.availableModels') }}</p>
                  <div class="grid grid-cols-3 md:grid-cols-5 gap-2">
                    <div v-for="model in voiceModels.whisper" :key="model.id" class="p-2 rounded border border-gray-200 dark:border-gray-600 text-xs text-center">
                      {{ model.name }}<br><span class="text-gray-400">{{ model.sizeMB }} MB</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Piper TTS Section -->
              <div class="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800">
                <div class="flex items-center gap-2 mb-3">
                  <SpeakerWaveIcon class="w-5 h-5 text-indigo-500" />
                  <span class="font-semibold text-gray-900 dark:text-white">{{ $t('settings.voice.piperTts') }}</span>
                  <span v-if="voiceModels.piperBinary" class="ml-auto text-xs px-2 py-0.5 rounded-full bg-green-200 dark:bg-green-800 text-green-800 dark:text-green-200">{{ $t('settings.voice.ready') }}</span>
                  <span v-else class="ml-auto text-xs px-2 py-0.5 rounded-full bg-red-200 dark:bg-red-800 text-red-800 dark:text-red-200">{{ $t('settings.voice.notInstalled') }}</span>
                </div>

                <!-- Hinweis wenn Binary fehlt -->
                <div v-if="!voiceModels.piperBinary" class="mb-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
                  <p class="text-sm text-yellow-800 dark:text-yellow-200 mb-2">
                    <strong>{{ $t('settings.voice.piperNotInstalled') }}</strong> {{ $t('settings.voice.piperInstallHint') }}
                  </p>
                  <button
                    @click="downloadPiper"
                    :disabled="voiceDownloading"
                    class="w-full px-4 py-2 rounded-lg bg-indigo-500 hover:bg-indigo-600 text-white font-medium disabled:opacity-50 flex flex-col items-center justify-center gap-1"
                  >
                    <div class="flex items-center gap-2">
                      <div v-if="voiceDownloadComponent === 'piper'" class="animate-spin rounded-full h-5 w-5 border-2 border-white border-t-transparent"></div>
                      <ArrowDownTrayIcon v-else class="w-5 h-5" />
                      {{ voiceDownloadComponent === 'piper' ? $t('settings.voice.downloading') : $t('settings.voice.installPiper') }}
                    </div>
                    <!-- Progress Bar nur wenn Piper heruntergeladen wird -->
                    <div v-if="voiceDownloadComponent === 'piper' && voiceDownloadProgress > 0" class="w-full mt-1">
                      <div class="w-full bg-indigo-300 rounded-full h-1.5">
                        <div class="bg-white h-1.5 rounded-full transition-all" :style="{ width: voiceDownloadProgress + '%' }"></div>
                      </div>
                      <span class="text-xs">{{ Math.round(voiceDownloadProgress) }}%</span>
                    </div>
                  </button>
                  <p class="text-xs text-yellow-600 dark:text-yellow-400 mt-2">
                    {{ $t('settings.voice.piperDownloadInfo') }}
                  </p>
                </div>

                <!-- Nur anzeigen wenn Binary vorhanden -->
                <template v-if="voiceModels.piperBinary">
                  <!-- Language Filter -->
                  <div class="flex gap-2 mb-3">
                    <button
                      @click="piperLanguageFilter = 'de'"
                      class="px-3 py-1 rounded-full text-sm transition-all"
                      :class="piperLanguageFilter === 'de' ? 'bg-indigo-500 text-white' : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-200'"
                    >
                      {{ $t('settings.voice.german') }}
                    </button>
                    <button
                      @click="piperLanguageFilter = 'en'"
                      class="px-3 py-1 rounded-full text-sm transition-all"
                      :class="piperLanguageFilter === 'en' ? 'bg-indigo-500 text-white' : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-200'"
                    >
                      {{ $t('settings.voice.english') }}
                    </button>
                    <button
                      @click="piperLanguageFilter = 'all'"
                      class="px-3 py-1 rounded-full text-sm transition-all"
                      :class="piperLanguageFilter === 'all' ? 'bg-indigo-500 text-white' : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-200'"
                    >
                      {{ $t('settings.voice.all') }}
                    </button>
                  </div>

                  <!-- Piper Voices Grid -->
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-3 max-h-80 overflow-y-auto pr-1">
                    <div
                      v-for="voice in filteredPiperVoices"
                      :key="voice.id"
                      class="p-3 rounded-lg border-2 cursor-pointer transition-all"
                      :class="[
                        voiceModels.currentPiper === voice.id
                          ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                          : voice.installed
                            ? 'border-green-300 dark:border-green-700 bg-green-50 dark:bg-green-900/10 hover:border-green-400'
                            : 'border-gray-200 dark:border-gray-600 hover:border-gray-300 dark:hover:border-gray-500'
                      ]"
                      @click="voice.installed && selectPiperVoice(voice.id)"
                    >
                      <div class="flex items-center justify-between mb-1">
                        <span class="font-medium text-gray-900 dark:text-white">{{ voice.name }}</span>
                        <div class="flex items-center gap-1">
                          <span v-if="voiceModels.currentPiper === voice.id" class="text-xs px-1.5 py-0.5 rounded bg-indigo-500 text-white">{{ $t('settings.voice.active') }}</span>
                          <span v-else-if="voice.installed" class="text-xs px-1.5 py-0.5 rounded bg-green-500 text-white">{{ $t('settings.voice.installed') }}</span>
                        </div>
                      </div>
                      <p class="text-xs text-gray-500 dark:text-gray-400">{{ voice.description }}</p>
                      <div class="flex items-center justify-between mt-2">
                        <div class="flex items-center gap-2 text-xs text-gray-400">
                          <span>{{ voice.language }}</span>
                          <span class="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700">{{ voice.quality }}</span>
                          <span>{{ voice.sizeMB }} MB</span>
                        </div>
                        <button
                          v-if="!voice.installed"
                          @click.stop="downloadSpecificModel('piper', voice.id)"
                          :disabled="voiceDownloading"
                          class="text-xs px-2 py-1 rounded bg-indigo-500 hover:bg-indigo-600 text-white disabled:opacity-50"
                        >
                          {{ $t('settings.voice.download') }}
                        </button>
                        <button
                          v-else-if="voiceModels.currentPiper !== voice.id"
                          @click.stop="selectPiperVoice(voice.id)"
                          class="text-xs px-2 py-1 rounded bg-gray-200 dark:bg-gray-600 hover:bg-gray-300 dark:hover:bg-gray-500 text-gray-700 dark:text-gray-200"
                        >
                          {{ $t('settings.voice.activate') }}
                        </button>
                      </div>
                    </div>
                  </div>
                </template>

                <!-- Vorschau der Stimmen wenn Binary fehlt (ausgegraut) -->
                <div v-else class="opacity-50 pointer-events-none">
                  <p class="text-sm text-gray-500 mb-2">{{ $t('settings.voice.availableVoices') }}</p>
                  <div class="grid grid-cols-2 md:grid-cols-3 gap-2">
                    <div v-for="voice in filteredPiperVoices?.slice(0, 6)" :key="voice.id" class="p-2 rounded border border-gray-200 dark:border-gray-600 text-xs">
                      {{ voice.name }} ({{ voice.language }})
                    </div>
                  </div>
                </div>
              </div>

              <!-- Voice Store - Alle Piper Stimmen durchsuchen -->
              <div class="mt-6 p-4 rounded-lg border border-indigo-200 dark:border-indigo-700 bg-indigo-50/50 dark:bg-indigo-900/10">
                <VoiceStore ref="voiceStoreRef" />
              </div>

              <!-- Info -->
              <div class="mt-4 p-3 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm text-gray-600 dark:text-gray-400">
                <p><strong>{{ $t('common.note') }}:</strong> {{ $t('settings.voice.voiceStorageInfo') }}</p>
                <p class="mt-1">{{ $t('settings.voice.clickToTest') }}</p>
              </div>
            </section>

            <!-- Voice Assistant / Wakeword Section -->
            <section class="mt-6 bg-gradient-to-br from-green-50 to-emerald-100 dark:from-green-900/30 dark:to-emerald-800/30 p-5 rounded-xl border border-green-200/50 dark:border-green-700/50 shadow-sm">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                <span class="text-2xl">🎙️</span>
                Voice Assistant (Ewa)
              </h3>

              <!-- Voice Assistant Enable/Disable -->
              <div class="mb-4 p-4 rounded-xl border border-green-200 dark:border-green-700 bg-white dark:bg-gray-800">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-3">
                    <span class="text-xl">{{ voiceAssistantSettings.enabled ? '🟢' : '⚪' }}</span>
                    <div>
                      <span class="font-semibold text-gray-900 dark:text-white">Voice Assistant aktivieren</span>
                      <p class="text-xs text-gray-500 dark:text-gray-400">Sprachsteuerung per Wake Word</p>
                    </div>
                  </div>
                  <label class="relative inline-flex items-center cursor-pointer">
                    <input type="checkbox" v-model="voiceAssistantSettings.enabled" @change="saveVoiceAssistantSettings" class="sr-only peer">
                    <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-green-300 dark:peer-focus:ring-green-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-green-500"></div>
                  </label>
                </div>
              </div>

              <!-- Wake Word Selection (nur wenn aktiviert) -->
              <div v-if="voiceAssistantSettings.enabled" class="space-y-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    🗣️ Wake Word auswählen
                  </label>
                  <div class="grid grid-cols-3 gap-3">
                    <button
                      @click="voiceAssistantSettings.wakeWord = 'hey_ewa'; saveVoiceAssistantSettings()"
                      :class="[
                        'p-3 rounded-lg border-2 transition-all text-center',
                        voiceAssistantSettings.wakeWord === 'hey_ewa'
                          ? 'border-green-500 bg-green-500/10'
                          : 'border-gray-300 dark:border-gray-600 hover:border-green-400'
                      ]"
                    >
                      <span class="block font-medium text-gray-900 dark:text-white">"Hey Ewa"</span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">Standard</span>
                    </button>
                    <button
                      @click="voiceAssistantSettings.wakeWord = 'ewa'; saveVoiceAssistantSettings()"
                      :class="[
                        'p-3 rounded-lg border-2 transition-all text-center',
                        voiceAssistantSettings.wakeWord === 'ewa'
                          ? 'border-green-500 bg-green-500/10'
                          : 'border-gray-300 dark:border-gray-600 hover:border-green-400'
                      ]"
                    >
                      <span class="block font-medium text-gray-900 dark:text-white">"Ewa"</span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">Kurz</span>
                    </button>
                    <button
                      @click="voiceAssistantSettings.wakeWord = 'custom'; saveVoiceAssistantSettings()"
                      :class="[
                        'p-3 rounded-lg border-2 transition-all text-center',
                        voiceAssistantSettings.wakeWord === 'custom'
                          ? 'border-green-500 bg-green-500/10'
                          : 'border-gray-300 dark:border-gray-600 hover:border-green-400'
                      ]"
                    >
                      <span class="block font-medium text-gray-900 dark:text-white">Eigenes</span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">Benutzerdefiniert</span>
                    </button>
                  </div>
                </div>

                <!-- Custom Wake Word Input -->
                <div v-if="voiceAssistantSettings.wakeWord === 'custom'">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Eigenes Wake Word
                  </label>
                  <input
                    v-model="voiceAssistantSettings.customWakeWord"
                    type="text"
                    placeholder="z.B. 'Computer', 'Jarvis', ..."
                    class="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-green-500 focus:border-transparent"
                    @change="saveVoiceAssistantSettings"
                  >
                </div>

                <!-- Auto-Stop Toggle -->
                <div class="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-white/50 dark:bg-gray-800/50">
                  <div class="flex items-center justify-between">
                    <div>
                      <label class="text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-2">
                        ⏹️ Auto-Stopp nach Antwort
                      </label>
                      <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Beendet Lauschen nach einer vollständigen Antwort
                      </p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="voiceAssistantSettings.autoStop" @change="saveVoiceAssistantSettings" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-green-300 dark:peer-focus:ring-green-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-green-600"></div>
                    </label>
                  </div>
                </div>

                <!-- Ruhezeiten -->
                <div class="p-4 rounded-xl border border-blue-200 dark:border-blue-700 bg-blue-50/50 dark:bg-blue-900/20">
                  <div class="flex items-center justify-between mb-3">
                    <div>
                      <label class="text-sm font-medium text-gray-700 dark:text-gray-300 flex items-center gap-2">
                        🌙 Ruhezeiten
                      </label>
                      <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                        Voice Assistant pausiert während dieser Zeit
                      </p>
                    </div>
                    <label class="relative inline-flex items-center cursor-pointer">
                      <input type="checkbox" v-model="voiceAssistantSettings.quietHoursEnabled" @change="saveVoiceAssistantSettings" class="sr-only peer">
                      <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"></div>
                    </label>
                  </div>
                  <div v-if="voiceAssistantSettings.quietHoursEnabled" class="flex items-center gap-2">
                    <input type="time" v-model="voiceAssistantSettings.quietHoursStart" @change="saveVoiceAssistantSettings" class="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm">
                    <span class="text-gray-500">bis</span>
                    <input type="time" v-model="voiceAssistantSettings.quietHoursEnd" @change="saveVoiceAssistantSettings" class="px-3 py-1.5 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm">
                  </div>
                </div>

                <!-- Anleitung -->
                <div class="p-4 bg-gradient-to-r from-green-50 to-blue-50 dark:from-green-900/20 dark:to-blue-900/20 border border-green-200 dark:border-green-700 rounded-lg">
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">💡 So funktioniert's</h4>
                  <ul class="space-y-1 text-xs text-gray-600 dark:text-gray-400">
                    <li>1. Sage <strong>"{{ getWakeWordDisplay(voiceAssistantSettings.wakeWord) }}"</strong> um Ewa zu aktivieren</li>
                    <li>2. Stelle deine Frage oder gib einen Befehl</li>
                    <li>3. Ewa antwortet per Sprache (TTS)</li>
                    <li>4. Wiederhole oder sage "Stop" zum Beenden</li>
                  </ul>
                </div>
              </div>
            </section>
          </div>

          <!-- TAB: Addons/Erweiterungen -->
          <div v-if="activeTab === 'addons'">
            <AddonsSettingsTab
              :tesseract-status="tesseractStatus"
              :tesseract-downloading="tesseractDownloading"
              :tesseract-download-progress="tesseractDownloadProgress"
              :tesseract-download-message="tesseractDownloadMessage"
              @download-tesseract="downloadTesseract"
              @postgres-status-change="onPostgresStatusChange"
            />
          </div>

          <!-- TAB: Danger Zone -->
          <div v-if="activeTab === 'danger'">
            <DangerZoneTab
              v-model:resetSelection="resetSelection"
              :resetting="resetting"
              @reset-all="handleResetAll"
            />
          </div>

        </div>

        <!-- System Prompt Editor Modal -->
        <Transition name="modal">
          <div v-if="showPromptEditor" class="absolute inset-0 bg-black/70 flex items-center justify-center z-50 p-4" @click.self="showPromptEditor = false">
            <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[80vh] overflow-y-auto">
              <div class="p-5 border-b border-gray-200 dark:border-gray-700">
                <h4 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ editingPrompt ? $t('settings.prompts.editTitle') : $t('settings.prompts.newTitle') }}
                </h4>
              </div>

              <div class="p-5 space-y-4">
                <!-- Name -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {{ $t('settings.prompts.name') }}
                  </label>
                  <input
                    v-model="promptForm.name"
                    type="text"
                    :placeholder="$t('settings.prompts.namePlaceholder')"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  >
                </div>

                <!-- Content -->
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    {{ $t('settings.prompts.promptText') }}
                  </label>
                  <textarea
                    v-model="promptForm.content"
                    rows="8"
                    :placeholder="$t('settings.prompts.promptPlaceholder')"
                    class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-y"
                  ></textarea>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ promptForm.content.length }} {{ $t('settings.prompts.characters') }}
                  </p>
                </div>

                <!-- Is Default -->
                <div class="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ $t('settings.prompts.setAsDefaultLabel') }}
                    </label>
                    <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                      {{ $t('settings.prompts.setAsDefaultHint') }}
                    </p>
                  </div>
                  <input
                    type="checkbox"
                    v-model="promptForm.isDefault"
                    class="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
                  >
                </div>
              </div>

              <div class="p-5 border-t border-gray-200 dark:border-gray-700 flex justify-end gap-3">
                <button
                  @click="showPromptEditor = false"
                  class="px-4 py-2 border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
                >
                  {{ $t('common.cancel') }}
                </button>
                <button
                  @click="saveSystemPrompt"
                  :disabled="!promptForm.name.trim() || !promptForm.content.trim()"
                  class="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  <CheckIcon class="w-4 h-4" />
                  {{ editingPrompt ? $t('settings.prompts.update') : $t('common.create') }}
                </button>
              </div>
            </div>
          </div>
        </Transition>

        <!-- Footer with Gradient -->
        <div class="sticky bottom-0 bg-gray-50/90 dark:bg-gray-900/90 backdrop-blur-sm border-t border-gray-200/50 dark:border-gray-700/50 px-6 py-4 flex justify-between">
          <button
            @click="resetToDefaults"
            class="px-4 py-2 rounded-xl text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-gray-700 transition-all flex items-center gap-2"
          >
            <ArrowPathIcon class="w-4 h-4" />
            {{ $t('common.reset') }}
          </button>
          <div class="flex gap-3">
            <button
              @click="close"
              class="px-5 py-2 rounded-xl border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-all transform hover:scale-105"
            >
              {{ $t('common.cancel') }}
            </button>
            <button
              @click="save"
              :disabled="saving"
              class="px-6 py-2 rounded-xl bg-gradient-to-r from-fleet-orange-500 to-orange-600 hover:from-fleet-orange-400 hover:to-orange-500 text-white font-semibold shadow-lg hover:shadow-xl transition-all transform hover:scale-105 disabled:opacity-50 flex items-center gap-2"
            >
              <CheckIcon v-if="!saving" class="w-5 h-5" />
              <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
              {{ saving ? $t('common.saving') : $t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, watch, onMounted, computed } from 'vue'
import {
  Cog6ToothIcon,
  XMarkIcon,
  GlobeAltIcon,
  LanguageIcon,
  SunIcon,
  CpuChipIcon,
  SparklesIcon,
  ExclamationTriangleIcon,
  CodeBracketIcon,
  BoltIcon,
  InformationCircleIcon,
  AdjustmentsHorizontalIcon,
  DocumentTextIcon,
  FireIcon,
  DocumentDuplicateIcon,
  PhotoIcon,
  EyeIcon,
  LinkIcon,
  WrenchScrewdriverIcon,
  Bars3Icon,
  HashtagIcon,
  BugAntIcon,
  ArrowPathIcon,
  CheckIcon,
  CheckCircleIcon,
  UserIcon,
  TrashIcon,
  ShieldExclamationIcon,
  UsersIcon,
  MagnifyingGlassIcon,
  StarIcon,
  ServerIcon,
  PlusIcon,
  FolderIcon,
  FolderOpenIcon,
  ComputerDesktopIcon,
  EnvelopeIcon,
  DocumentIcon,
  MicrophoneIcon,
  SpeakerWaveIcon,
  ArrowDownTrayIcon,
  ArrowsRightLeftIcon,
  LightBulbIcon,
  ChartBarIcon,
  PuzzlePieceIcon,
  DocumentMagnifyingGlassIcon
} from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '../stores/settingsStore'
import { useChatStore } from '../stores/chatStore'
import PersonalInfoTab from './PersonalInfoTab.vue'
import ProviderSettings from './ProviderSettings.vue'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import { formatDateAbsolute } from '../composables/useFormatters'
import api from '../services/api'
import { secureFetch } from '../utils/secureFetch'
import ToggleSwitch from './ToggleSwitch.vue'
import SimpleSamplingParams from './SimpleSamplingParams.vue'
import VRAMSettings from './settings/VRAMSettings.vue'
import PostgreSQLMigration from './settings/PostgreSQLMigration.vue'
import VoiceStore from './VoiceStore.vue'
import ObserverSettings from './ObserverSettings.vue'
import { filterVisionModels, filterCodeModels } from '../utils/modelFilters'

// New refactored tab components
import GeneralSettingsTab from './settings/GeneralSettingsTab.vue'
import MatesSettingsTab from './settings/MatesSettingsTab.vue'
import AgentsSettingsTab from './settings/AgentsSettingsTab.vue'
import AddonsSettingsTab from './settings/AddonsSettingsTab.vue'
import DangerZoneTab from './settings/DangerZoneTab.vue'
import CustomModelsTab from './settings/CustomModelsTab.vue'
import WebSearchSettingsTab from './settings/WebSearchSettingsTab.vue'
import VoiceSettingsTab from './settings/VoiceSettingsTab.vue'

const { success, error: errorToast } = useToast()
const { confirm, confirmDelete } = useConfirmDialog()

const props = defineProps({
  isOpen: Boolean,
  initialTab: String
})

const emit = defineEmits(['close', 'save'])

const settingsStore = useSettingsStore()
const chatStore = useChatStore()

// PostgreSQL Status
const postgresConnected = ref(false)

function onPostgresStatusChange(connected) {
  postgresConnected.value = connected
  console.log('PostgreSQL Status:', connected ? 'Verbunden' : 'SQLite')
}

// Local copy of settings for editing
const settings = ref({ ...settingsStore.settings })
const saving = ref(false)
const resetting = ref(false)

// Voice Download Dialog für Sprachwechsel
const showVoiceDownloadDialog = ref(false)
const voiceDownloadInfo = ref({ availableVoices: [], installedVoices: [] })
const isDownloadingVoices = ref(false)

// Sampling Parameters
const samplingParams = ref({})

// Reset selection checkboxes
const resetSelection = ref({
  chats: true,
  projects: true,
  customModels: false,  // Custom Models standardmäßig NICHT löschen
  settings: true,
  personalInfo: true,
  templates: true,
  stats: true
})

// Active tab
const activeTab = ref(props.initialTab || 'general')

// Watch for tab changes to reload data when needed
watch(activeTab, async (newTab, oldTab) => {
  if (newTab === 'mates') {
    await loadTrustedMatesCount()
    startPairingPoll()
  } else if (oldTab === 'mates') {
    stopPairingPoll()
  }

  if (newTab === 'voice') {
    await loadVoiceStatus()
    await loadVoiceModels()
    await loadTtsSetting()
  }
})

// Sync cpuOnly with settingsStore (persistent) - für CPU-Only Mode Toggle
watch(() => settings.value.cpuOnly, (newValue) => {
  settingsStore.settings.cpuOnly = newValue
  console.log('🖥️ CPU-Only Mode:', newValue ? 'aktiviert' : 'deaktiviert')
}, { immediate: false })

// Ref to PersonalInfoTab
const personalInfoTabRef = ref(null)

// Fleet Mates Pairing
const trustedMatesCount = ref(0)
const trustedMates = ref([])
const forgettingMates = ref(false)
const removingMateId = ref(null)
const pendingPairingRequests = ref([])
const processingPairing = ref(false)
let pairingPollInterval = null

// Mate type helpers
const getMateTypeIcon = (type) => {
  switch (type) {
    case 'os': return ComputerDesktopIcon
    case 'mail': return EnvelopeIcon
    case 'office': return DocumentIcon
    case 'browser': return GlobeAltIcon
    default: return ComputerDesktopIcon
  }
}

const getMateTypeColor = (type) => {
  switch (type) {
    case 'os': return 'bg-blue-500'
    case 'mail': return 'bg-purple-500'
    case 'office': return 'bg-green-500'
    case 'browser': return 'bg-orange-500'
    default: return 'bg-gray-500'
  }
}

const getMateTypeLabel = (type) => {
  switch (type) {
    case 'os': return 'System-Agent'
    case 'mail': return 'E-Mail-Agent'
    case 'office': return 'Office-Agent'
    case 'browser': return 'Browser-Agent'
    default: return 'Fleet Mate'
  }
}

// File Search (OS Mate RAG)
const fileSearchFolders = ref([])
const fileSearchStatus = ref(null)
const newFolderPath = ref('')

async function loadFileSearchStatus() {
  try {
    const response = await fetch('/api/file-search/status')
    if (response.ok) {
      fileSearchStatus.value = await response.json()
      fileSearchFolders.value = fileSearchStatus.value.searchFolders || []
    }
  } catch (err) {
    console.error('Failed to load file search status:', err)
  }
}

// ========================================
// Tesseract OCR Status
// ========================================

const tesseractStatus = ref({
  installed: false,
  binaryPath: '',
  languages: [],
  dataDir: ''
})
const tesseractDownloading = ref(false)
const tesseractDownloadProgress = ref(0)
const tesseractDownloadMessage = ref('')

async function loadTesseractStatus() {
  try {
    const response = await fetch('/api/setup/tesseract/status')
    if (response.ok) {
      tesseractStatus.value = await response.json()
    }
  } catch (err) {
    console.error('Failed to load Tesseract status:', err)
  }
}

async function downloadTesseract() {
  if (tesseractDownloading.value) return

  tesseractDownloading.value = true
  tesseractDownloadProgress.value = 0
  tesseractDownloadMessage.value = 'Starte Download...'

  try {
    const eventSource = new EventSource('/api/setup/tesseract/download')

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)

        if (data.status === 'complete') {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadProgress.value = 100
          tesseractDownloadMessage.value = 'Installation abgeschlossen!'
          success('Tesseract OCR erfolgreich installiert!')
          // Status neu laden
          loadTesseractStatus()
          return
        }

        if (data.error) {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadMessage.value = 'Fehler: ' + data.error
          errorToast('Tesseract-Installation fehlgeschlagen: ' + data.error)
          return
        }

        tesseractDownloadProgress.value = data.percent || 0
        tesseractDownloadMessage.value = data.message || 'Downloading...'

        if (data.done) {
          eventSource.close()
          tesseractDownloading.value = false
          tesseractDownloadProgress.value = 100
          success('Tesseract OCR erfolgreich installiert!')
          loadTesseractStatus()
        }
      } catch (e) {
        console.error('Error parsing SSE:', e)
      }
    }

    eventSource.onerror = (error) => {
      console.error('SSE Error:', error)
      eventSource.close()
      tesseractDownloading.value = false
      // Prüfe ob Download erfolgreich war
      if (tesseractDownloadProgress.value >= 99) {
        success('Tesseract OCR erfolgreich installiert!')
        loadTesseractStatus()
      } else {
        tesseractDownloadMessage.value = 'Verbindungsfehler'
        errorToast('Download-Verbindung unterbrochen')
      }
    }
  } catch (err) {
    console.error('Failed to download Tesseract:', err)
    tesseractDownloading.value = false
    tesseractDownloadMessage.value = 'Fehler: ' + err.message
    errorToast('Tesseract-Download fehlgeschlagen')
  }
}

// ========================================
// Sprachwechsel mit Voice-Download Dialog
// ========================================

async function onLanguageChange() {
  const newLocale = settings.value.language
  console.log('[Settings] Sprachwechsel zu:', newLocale)

  try {
    // Backend informieren
    const response = await fetch('/api/settings/language', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ locale: newLocale })
    })

    if (response.ok) {
      const result = await response.json()
      console.log('[Settings] Sprache gewechselt:', result)

      // i18n aktualisieren
      const { locale } = useI18n()
      locale.value = newLocale
      localStorage.setItem('fleet-navigator-locale', newLocale)
      document.documentElement.lang = newLocale

      // Prüfen ob Voice-Download nötig
      if (result.needsVoiceDownload && result.availableVoices?.length > 0) {
        voiceDownloadInfo.value = result
        showVoiceDownloadDialog.value = true
      }

      showToast(`Sprache auf ${newLocale.toUpperCase()} gewechselt`, 'success')
    }
  } catch (err) {
    console.error('[Settings] Sprachwechsel fehlgeschlagen:', err)
    showToast('Sprachwechsel fehlgeschlagen', 'error')
  }
}

async function downloadVoicesForLanguage() {
  const locale = settings.value.language
  const voices = voiceDownloadInfo.value.availableVoices || []

  if (voices.length === 0) {
    showVoiceDownloadDialog.value = false
    return
  }

  isDownloadingVoices.value = true

  try {
    // Piper-Stimme herunterladen (mit korrektem Endpoint via SSE)
    const voice = voices[0]
    // Voice-ID Format: de_DE-thorsten-medium oder tr_TR-fahrettin-medium
    const voiceId = voice.id

    console.log('[Settings] Lade Stimme herunter:', voiceId)

    // SSE-basierter Download wie im SetupWizard
    await new Promise((resolve, reject) => {
      const eventSource = new EventSource(
        `/api/setup/download-voice?component=piper&modelId=${encodeURIComponent(voiceId)}&lang=${locale}`
      )

      eventSource.onmessage = (event) => {
        try {
          const progress = JSON.parse(event.data)

          if (progress.message) {
            console.log('[Settings] Voice-Download:', progress.message)
          }

          if (progress.done) {
            eventSource.close()
            if (progress.error) {
              reject(new Error(progress.error))
            } else {
              resolve()
            }
          }
        } catch (e) {
          console.error('[Settings] Progress Parse Fehler:', e)
        }
      }

      eventSource.onerror = (error) => {
        console.error('[Settings] SSE Fehler:', error)
        eventSource.close()
        reject(new Error('Download-Verbindung unterbrochen'))
      }
    })

    showToast(`Stimme "${voice.name}" heruntergeladen!`, 'success')
    showVoiceDownloadDialog.value = false

    // Voice-Status neu laden
    await loadVoiceStatus()
  } catch (err) {
    console.error('[Settings] Voice-Download fehlgeschlagen:', err)
    showToast(`Download fehlgeschlagen: ${err.message}`, 'error')
  } finally {
    isDownloadingVoices.value = false
  }
}

async function addSearchFolder(folderPath = null) {
  const pathToAdd = folderPath || newFolderPath.value
  if (!pathToAdd) return

  try {
    const response = await secureFetch('/api/file-search/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderPath: pathToAdd })
    })

    if (response.ok) {
      newFolderPath.value = ''
      await loadFileSearchStatus()
      success('Ordner hinzugefügt und Indexierung gestartet')
    } else {
      const data = await response.json()
      errorToast(data.error || 'Fehler beim Hinzufügen')
    }
  } catch (err) {
    console.error('Failed to add search folder:', err)
    errorToast('Fehler beim Hinzufügen des Ordners')
  }
}

async function removeSearchFolder(folderId) {
  try {
    await secureFetch(`/api/file-search/folders/${folderId}`, { method: 'DELETE' })
    await loadFileSearchStatus()
    success('Ordner entfernt')
  } catch (err) {
    console.error('Failed to remove folder:', err)
    errorToast('Fehler beim Entfernen')
  }
}

async function reindexFolder(folderId) {
  try {
    await secureFetch(`/api/file-search/folders/${folderId}/reindex`, { method: 'POST' })
    success('Neu-Indexierung gestartet')
    // Refresh status after a delay
    setTimeout(loadFileSearchStatus, 2000)
  } catch (err) {
    console.error('Failed to reindex folder:', err)
    errorToast('Fehler beim Indexieren')
  }
}

// formatDate importiert aus useFormatters.js als formatDateAbsolute

// Tab configuration
const tabs = [
  { id: 'general', label: 'Allgemein', icon: GlobeAltIcon },
  { id: 'mates', label: 'Fleet Mates', icon: UsersIcon },
  { id: 'providers', label: 'LLM Provider', icon: CpuChipIcon },
  { id: 'customModels', label: 'Custom Modell', icon: AdjustmentsHorizontalIcon },
  { id: 'personal', label: 'Persönliche Daten', icon: UserIcon },
  { id: 'observer', label: 'Observer', icon: ChartBarIcon },
  { id: 'agents', label: 'Agents', icon: SparklesIcon },
  { id: 'web-search', label: 'Web-Suche', icon: MagnifyingGlassIcon },
  { id: 'voice', label: 'Sprache', icon: MicrophoneIcon },
  { id: 'addons', label: 'Erweiterungen', icon: PuzzlePieceIcon },
  { id: 'danger', label: 'Danger Zone', icon: ShieldExclamationIcon }
]

// Model selection settings
const modelSelectionSettings = ref({
  enabled: true,
  codeModel: 'qwen2.5-coder:7b',
  fastModel: 'llama3.2:3b',
  visionModel: 'llava:13b',
  defaultModel: 'qwen2.5-coder:7b',
  visionChainingEnabled: true,
  visionChainingSmartSelection: true
})

// Web Search Think First (LLM denkt erst nach, dann Websuche bei Unsicherheit)
const webSearchThinkFirst = ref(true)

// Load Think First setting from localStorage
function loadThinkFirstSetting() {
  try {
    const chainingSettings = JSON.parse(localStorage.getItem('chainingSettings') || '{}')
    webSearchThinkFirst.value = chainingSettings.webSearchThinkFirst ?? true
  } catch (e) {
    console.error('Failed to load Think First setting:', e)
  }
}

// Save Think First setting to localStorage
watch(webSearchThinkFirst, (newValue) => {
  try {
    const chainingSettings = JSON.parse(localStorage.getItem('chainingSettings') || '{}')
    chainingSettings.webSearchThinkFirst = newValue
    localStorage.setItem('chainingSettings', JSON.stringify(chainingSettings))
    console.log('💾 Think First setting saved:', newValue)
  } catch (e) {
    console.error('Failed to save Think First setting:', e)
  }
})

// Load on component mount
loadThinkFirstSetting()

// Fleet Mates Model Settings
const mateModels = ref({
  emailModel: '',
  documentModel: '',
  logAnalysisModel: '',
  coderModel: ''
})

// Web Search Settings (Brave API + SearXNG)
const webSearchSettings = ref({
  braveApiKey: '',
  braveConfigured: false,
  searchCount: 0,
  searchLimit: 2000,
  remainingSearches: 2000,
  currentMonth: '',
  customSearxngInstance: '',  // Eigene Instanz (Priorität 1)
  searxngInstances: [],       // Öffentliche Fallback-Instanzen
  searxngTotalCount: 0,       // Gesamte SearXNG-Suchen
  searxngMonthCount: 0,       // SearXNG-Suchen diesen Monat
  // Feature Flags
  queryOptimizationEnabled: true,
  contentScrapingEnabled: true,
  multiQueryEnabled: false,
  reRankingEnabled: true,
  queryOptimizationModel: 'llama3.2:3b',
  effectiveOptimizationModel: null,  // Das tatsächlich verwendete Modell (nach Fallback)
  // UI Animation
  webSearchAnimation: 'data-wave'  // Animation: data-wave, orbit, radar, constellation
})

// Animation Options für Dropdown
const animationOptions = [
  { value: 'data-wave', label: '🌊 Data Wave', description: 'Fließende Datenwelle' },
  { value: 'orbit', label: '🌐 Orbiting Network', description: 'Kreisende Datenpunkte' },
  { value: 'radar', label: '📡 Radar Scan', description: 'Radar-Scanning-Effekt' },
  { value: 'constellation', label: '✨ Constellation', description: 'Sternbild-Netzwerk' }
]

// Voice Settings (STT/TTS)
const voiceStatus = ref({
  initialized: false,
  whisper: { available: false, binaryFound: false, modelFound: false, model: '' },
  piper: { available: false, binaryFound: false, voiceFound: false, voice: '' }
})
const voiceModels = ref({
  whisper: [],
  piper: [],
  currentWhisper: '',
  currentPiper: '',
  whisperBinary: false,
  piperBinary: false
})
const voiceDownloading = ref(false)
const voiceDownloadComponent = ref('')  // 'whisper' oder 'piper' - welche Komponente wird gerade geladen
const voiceDownloadStatus = ref('')
const voiceDownloadProgress = ref(0)
const voiceDownloadSpeed = ref('')
const piperLanguageFilter = ref('de')
const ttsEnabled = ref(true)

// Voice Assistant Settings
const voiceAssistantSettings = ref({
  enabled: false,
  wakeWord: 'hey_ewa',
  customWakeWord: '',
  autoStop: true,
  quietHoursEnabled: false,
  quietHoursStart: '22:00',
  quietHoursEnd: '07:00'
})

// VoiceStore ref
const voiceStoreRef = ref(null)

// TTS Toggle
async function toggleTtsEnabled() {
  ttsEnabled.value = !ttsEnabled.value
  try {
    await api.updateSettings({ ttsEnabled: ttsEnabled.value })
    // Auch im localStorage für schnellen Zugriff
    localStorage.setItem('ttsEnabled', String(ttsEnabled.value))
  } catch (err) {
    console.error('Failed to save TTS setting:', err)
  }
}

// Load TTS setting on mount
async function loadTtsSetting() {
  // Erst localStorage prüfen für schnellen Start
  const stored = localStorage.getItem('ttsEnabled')
  if (stored !== null) {
    ttsEnabled.value = stored === 'true'
  }
  // Dann vom Backend laden
  try {
    const settings = await api.getVoiceAssistantSettings()
    if (settings?.ttsEnabled !== undefined) {
      ttsEnabled.value = settings.ttsEnabled
      localStorage.setItem('ttsEnabled', String(ttsEnabled.value))
    }
  } catch (err) {
    // Endpoint nicht implementiert - ignorieren, localStorage-Wert behalten
    console.debug('Voice assistant settings not available:', err.message)
  }
}

// Voice Assistant Functions
async function loadVoiceAssistantSettings() {
  try {
    const response = await fetch('/api/voice-assistant/settings')
    if (response.ok) {
      const data = await response.json()
      voiceAssistantSettings.value = {
        enabled: data.enabled ?? false,
        wakeWord: data.wakeWord ?? 'hey_ewa',
        customWakeWord: data.customWakeWord ?? '',
        autoStop: data.autoStop ?? true,
        quietHoursEnabled: data.quietHoursEnabled ?? false,
        quietHoursStart: data.quietHoursStart ?? '22:00',
        quietHoursEnd: data.quietHoursEnd ?? '07:00'
      }
    }
  } catch (err) {
    console.error('Failed to load voice assistant settings:', err)
  }
}

async function saveVoiceAssistantSettings() {
  try {
    await fetch('/api/voice-assistant/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(voiceAssistantSettings.value)
    })
  } catch (err) {
    console.error('Failed to save voice assistant settings:', err)
  }
}

function getWakeWordDisplay(wakeWord) {
  switch (wakeWord) {
    case 'hey_ewa':
      return 'Hey Ewa'
    case 'ewa':
      return 'Ewa'
    case 'custom':
      return voiceAssistantSettings.value.customWakeWord || 'Eigenes Wake Word'
    default:
      return 'Hey Ewa'
  }
}

// Download custom voice from HuggingFace
async function downloadCustomVoice() {
  if (!customVoiceId.value) return

  customVoiceError.value = ''
  customVoiceSuccess.value = ''
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'custom'
  voiceDownloadStatus.value = 'Lade Stimme herunter...'
  voiceDownloadProgress.value = 0

  try {
    // Parse voice ID - expected format: locale-name-quality (e.g., de_DE-mls-medium)
    let voiceId = customVoiceId.value.trim()

    // If it's a URL, extract the voice ID
    if (voiceId.includes('huggingface.co')) {
      const match = voiceId.match(/([a-z]{2}_[A-Z]{2}-[\w]+-[\w]+)/)
      if (match) {
        voiceId = match[1]
      }
    }

    // Validate format
    const voicePattern = /^[a-z]{2}_[A-Z]{2}-[\w]+-[\w]+$/
    if (!voicePattern.test(voiceId)) {
      throw new Error('Ungültiges Format. Erwartet: locale-name-quality (z.B. de_DE-mls-medium)')
    }

    // Use existing download API
    const eventSource = new EventSource(`/api/voice/download-model?component=piper&model=${voiceId}`)

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        console.log('Custom voice download progress:', data)

        if (data.percent) {
          voiceDownloadProgress.value = data.percent
        }
        if (data.speedMBps) {
          voiceDownloadSpeed.value = `${data.speedMBps.toFixed(1)} MB/s`
        }
        voiceDownloadStatus.value = data.status === 'downloading'
          ? `Lade ${data.file || voiceId}... ${Math.round(data.percent || 0)}%`
          : data.status

        if (data.status === 'done' || data.status === 'complete') {
          eventSource.close()
          voiceDownloading.value = false
          voiceDownloadComponent.value = ''
          customVoiceSuccess.value = `Stimme "${voiceId}" erfolgreich installiert!`
          customVoiceId.value = ''
          loadVoiceModels()
        } else if (data.status === 'error') {
          eventSource.close()
          voiceDownloading.value = false
          voiceDownloadComponent.value = ''
          customVoiceError.value = data.error || 'Download fehlgeschlagen'
        }
      } catch (e) {
        console.error('Error parsing SSE data:', e)
      }
    }

    eventSource.onerror = (err) => {
      console.error('SSE error:', err)
      eventSource.close()
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      customVoiceError.value = 'Verbindungsfehler beim Download'
    }

  } catch (err) {
    console.error('Custom voice download failed:', err)
    voiceDownloading.value = false
    voiceDownloadComponent.value = ''
    customVoiceError.value = err.message || 'Download fehlgeschlagen'
  }
}

// Filtered Piper voices based on language
const filteredPiperVoices = computed(() => {
  if (!voiceModels.value.piper) return []
  if (piperLanguageFilter.value === 'all') return voiceModels.value.piper
  return voiceModels.value.piper.filter(v =>
    v.language?.startsWith(piperLanguageFilter.value === 'de' ? 'de' : 'en')
  )
})

// Load Voice Status
async function loadVoiceStatus() {
  try {
    const status = await api.getVoiceStatus()
    voiceStatus.value = status
    console.log('🎤 Voice status loaded:', status)
  } catch (err) {
    console.warn('Failed to load voice status:', err)
  }
}

// Load Voice Models with installed status
async function loadVoiceModels() {
  try {
    const models = await api.getVoiceModels()
    voiceModels.value = models
    console.log('🎤 Voice models loaded:', models)
  } catch (err) {
    console.warn('Failed to load voice models:', err)
  }
}

// Select Whisper Model
async function selectWhisperModel(modelId) {
  try {
    await api.setVoiceConfig({ whisperModel: modelId })
    voiceModels.value.currentWhisper = modelId
    success(`Whisper-Modell "${modelId}" aktiviert`)
    await loadVoiceStatus()
  } catch (err) {
    errorToast('Fehler beim Aktivieren: ' + err.message)
  }
}

// Select Piper Voice
async function selectPiperVoice(voiceId) {
  try {
    await api.setVoiceConfig({ piperVoice: voiceId })
    voiceModels.value.currentPiper = voiceId
    success(`Stimme "${voiceId}" aktiviert`)
    await loadVoiceStatus()
  } catch (err) {
    errorToast('Fehler beim Aktivieren: ' + err.message)
  }
}

// Download specific model
function downloadSpecificModel(component, modelId) {
  voiceDownloading.value = true
  voiceDownloadStatus.value = `Lade ${component === 'whisper' ? 'Whisper-Modell' : 'Piper-Stimme'} "${modelId}"...`
  voiceDownloadProgress.value = 0

  api.downloadVoiceModel(component, modelId, (progress) => {
    voiceDownloadStatus.value = `${progress.component}: ${progress.file || modelId} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadStatus.value = ''
      voiceDownloadSpeed.value = ''
      loadVoiceModels()
      loadVoiceStatus()
      success(`${component === 'whisper' ? 'Modell' : 'Stimme'} erfolgreich installiert!`)
    } else if (progress.status === 'error') {
      voiceDownloading.value = false
      errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
    }
  })
}

// Download Whisper (Binary + Model)
async function downloadWhisper() {
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'whisper'
  voiceDownloadStatus.value = 'Starte Whisper Download...'
  voiceDownloadProgress.value = 0

  // Nur Whisper herunterladen (nicht Piper)
  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `Whisper: ${progress.file || 'binary'} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'error' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      if (progress.status === 'done' || progress.status === 'complete') {
        success('Whisper erfolgreich installiert!')
      } else if (progress.status === 'error') {
        errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
      }
    }
  }, 'whisper')  // <-- Nur Whisper herunterladen
}

// Download Piper (Binary + Voice)
async function downloadPiper() {
  voiceDownloading.value = true
  voiceDownloadComponent.value = 'piper'
  voiceDownloadStatus.value = 'Starte Piper Download...'
  voiceDownloadProgress.value = 0

  // Nur Piper herunterladen (nicht Whisper)
  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `Piper: ${progress.file || 'binary'} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done' || progress.status === 'error' || progress.status === 'complete') {
      voiceDownloading.value = false
      voiceDownloadComponent.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      if (progress.status === 'done' || progress.status === 'complete') {
        success('Piper erfolgreich installiert!')
      } else if (progress.status === 'error') {
        errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
      }
    }
  }, 'piper')  // <-- Nur Piper herunterladen
}

// Download All Voice Models
async function downloadAllVoice() {
  voiceDownloading.value = true
  voiceDownloadStatus.value = 'Starte Downloads...'
  voiceDownloadProgress.value = 0

  const eventSource = api.downloadVoiceModels((progress) => {
    voiceDownloadStatus.value = `${progress.component}: ${progress.file} (${progress.status})`
    voiceDownloadProgress.value = progress.percent || 0
    if (progress.speedMBps) {
      voiceDownloadSpeed.value = `${progress.speedMBps.toFixed(1)} MB/s`
    }

    if (progress.status === 'done') {
      voiceDownloading.value = false
      voiceDownloadStatus.value = ''
      voiceDownloadSpeed.value = ''
      loadVoiceStatus()
      loadVoiceModels()
      success('Voice-Modelle erfolgreich installiert!')
    } else if (progress.status === 'error') {
      voiceDownloading.value = false
      errorToast('Download fehlgeschlagen: ' + (progress.error || 'Unbekannter Fehler'))
    }
  })
}

const defaultSearxngInstances = [
  'https://search.sapti.me',
  'https://searx.tiekoetter.com',
  'https://priv.au',
  'https://search.ononoki.org',
  'https://search.bus-hit.me',
  'https://paulgo.io'
]

const testingSearch = ref(false)

// Computed: Farbe des Zählers basierend auf Verbrauch
const searchCountColor = computed(() => {
  const percent = (webSearchSettings.value.searchCount / webSearchSettings.value.searchLimit) * 100
  if (percent >= 90) return 'text-red-500'
  if (percent >= 70) return 'text-yellow-500'
  return 'text-green-500'
})

// Computed: Prozent für Progress Bar
const searchCountPercent = computed(() => {
  return Math.min(100, (webSearchSettings.value.searchCount / webSearchSettings.value.searchLimit) * 100)
})

// Available models
const availableModels = ref([])

// Fast models (< 10GB, good for Mates)
const fastModels = computed(() => {
  return availableModels.value.filter(m => m.size && m.size < 10 * 1024 * 1024 * 1024)
})

// Format file size
function formatSize(bytes) {
  if (!bytes) return '?'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)}GB`
  const mb = bytes / (1024 * 1024)
  return `${mb.toFixed(0)}MB`
}

// System Prompts Management
const systemPrompts = ref([])
const showPromptEditor = ref(false)
const editingPrompt = ref(null)
const promptForm = ref({
  name: '',
  content: '',
  isDefault: false
})

// Filtered models for specific use cases
const visionModels = computed(() => {
  const allModelNames = availableModels.value.map(m => m.name)
  const filtered = filterVisionModels(allModelNames)
  console.log('Vision Models:', filtered) // Debug
  return filtered
})
const codeModels = computed(() => {
  const allModelNames = availableModels.value.map(m => m.name)
  const filtered = filterCodeModels(allModelNames)
  console.log('Code Models:', filtered) // Debug
  return filtered
})

// Kleine/schnelle Modelle für Query-Optimierung (1B-7B Parameter)
const smallModels = computed(() => {
  const smallPatterns = [
    /llama.*[1-3]b/i,
    /qwen.*[1-3]b/i,
    /phi.*[1-3]/i,
    /gemma.*2b/i,
    /tinyllama/i,
    /smollm/i,
    /mistral.*7b/i,
    /llama.*7b/i,
    /qwen.*7b/i,
  ]

  return availableModels.value
    .map(m => m.name)
    .filter(name => smallPatterns.some(pattern => pattern.test(name)))
    .sort((a, b) => {
      // Sortiere nach Parametergröße (kleinste zuerst)
      const sizeA = parseInt(a.match(/(\d+)b/i)?.[1] || '99')
      const sizeB = parseInt(b.match(/(\d+)b/i)?.[1] || '99')
      return sizeA - sizeB
    })
})

// Check if at least one option is selected
const hasAnySelection = computed(() => {
  return Object.values(resetSelection.value).some(val => val === true)
})

// Load model selection settings and available models on mount
onMounted(async () => {
  // Initialize sampling params from settings store
  samplingParams.value = {
    maxTokens: settingsStore.settings.maxTokens || 512,
    temperature: settingsStore.settings.temperature || 0.7,
    topP: settingsStore.settings.topP || 0.9,
    topK: settingsStore.settings.topK || 40,
    minP: settingsStore.settings.minP || 0.05,
    repeatPenalty: settingsStore.settings.repeatPenalty || 1.18,
    repeatLastN: settingsStore.settings.repeatLastN || 64,
    presencePenalty: settingsStore.settings.presencePenalty || 0.0,
    frequencyPenalty: settingsStore.settings.frequencyPenalty || 0.0,
    mirostatMode: settingsStore.settings.mirostatMode || 0,
    mirostatTau: settingsStore.settings.mirostatTau || 5.0,
    mirostatEta: settingsStore.settings.mirostatEta || 0.1
  }

  // Set initial tab if provided
  if (props.initialTab) {
    activeTab.value = props.initialTab
  }

  await loadModelSelectionSettings()
  await loadAvailableModels()
  await loadMateModels()
  await loadSystemPrompts()
  await loadTrustedMatesCount()
  await loadWebSearchSettings()
  await loadFileSearchStatus()
  await loadVoiceAssistantSettings()
  await loadTesseractStatus()
})

// Schriftgröße setzen und persistieren (stufenlos)
function setFontSize(size) {
  // Konvertiere alte String-Werte zu Zahlen
  if (typeof size === 'string') {
    const sizeMap = { small: 85, medium: 100, large: 115, xlarge: 130 }
    size = sizeMap[size] || 100
  }
  settings.value.fontSize = size
  settingsStore.settings.fontSize = size
  // CSS Custom Property anwenden
  applyFontSize(size)
}

// Schriftgröße auf das root-Element anwenden (stufenlos via CSS Variable)
function applyFontSize(size) {
  // Konvertiere alte String-Werte zu Zahlen
  if (typeof size === 'string') {
    const sizeMap = { small: 85, medium: 100, large: 115, xlarge: 130 }
    size = sizeMap[size] || 100
  }
  const root = document.documentElement
  // Alte Klassen entfernen (für Kompatibilität)
  root.classList.remove('font-size-small', 'font-size-medium', 'font-size-large', 'font-size-xlarge')
  // CSS Custom Property setzen (stufenlos)
  root.style.setProperty('--font-scale', size / 100)
  root.style.fontSize = `${size}%`
}

async function loadModelSelectionSettings() {
  try {
    const loadedSettings = await api.getModelSelectionSettings()
    modelSelectionSettings.value = loadedSettings
  } catch (error) {
    console.error('Failed to load model selection settings:', error)
  }
}

async function loadAvailableModels() {
  try {
    const response = await api.getAvailableModels()
    // API gibt {current_model, models: [...]} zurück
    // Konvertiere zu Array von {name, size} Objekten
    const modelList = response.models || response || []
    if (Array.isArray(modelList)) {
      availableModels.value = modelList.map(m => {
        // Wenn es bereits ein Objekt mit name ist
        if (typeof m === 'object' && m.name) {
          return m
        }
        // Wenn es nur ein String ist
        return { name: m, size: 0 }
      })
    } else {
      availableModels.value = []
    }
    console.log('📦 Loaded', availableModels.value.length, 'models')
  } catch (error) {
    console.error('Failed to load available models:', error)
    availableModels.value = []
  }
}

// Fleet Mates Model Functions
async function loadMateModels() {
  try {
    const [emailRes, docRes, logRes, coderRes] = await Promise.all([
      fetch('/api/settings/email-model'),
      fetch('/api/settings/document-model'),
      fetch('/api/settings/log-analysis-model'),
      fetch('/api/settings/coder-model')
    ])

    mateModels.value.emailModel = emailRes.ok ? await emailRes.text() : ''
    mateModels.value.documentModel = docRes.ok ? await docRes.text() : ''
    mateModels.value.logAnalysisModel = logRes.ok ? await logRes.text() : ''
    mateModels.value.coderModel = coderRes.ok ? await coderRes.text() : ''
  } catch (error) {
    console.error('Failed to load mate models:', error)
  }
}

// Einzelne Modell-Speicher-Funktionen (um Race Conditions zu vermeiden)
async function saveEmailModel(model = null) {
  if (model !== null) mateModels.value.emailModel = model
  try {
    await fetch('/api/settings/email-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.emailModel
    })
    success('Email-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save email model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveDocumentModel(model = null) {
  if (model !== null) mateModels.value.documentModel = model
  try {
    await fetch('/api/settings/document-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.documentModel
    })
    success('Dokument-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save document model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveLogAnalysisModel(model = null) {
  if (model !== null) mateModels.value.logAnalysisModel = model
  try {
    await fetch('/api/settings/log-analysis-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.logAnalysisModel
    })
    success('Log-Analyse-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save log analysis model:', error)
    errorToast('Fehler beim Speichern')
  }
}

async function saveCoderModel(model = null) {
  if (model !== null) mateModels.value.coderModel = model
  try {
    await fetch('/api/settings/coder-model', {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: mateModels.value.coderModel
    })
    success('Coder-Modell gespeichert')
  } catch (error) {
    console.error('Failed to save coder model:', error)
    errorToast('Fehler beim Speichern')
  }
}

// Fleet Mates Functions
async function loadTrustedMatesCount() {
  try {
    const response = await fetch('/api/pairing/trusted')
    if (response.ok) {
      const mates = await response.json()
      trustedMates.value = mates
      trustedMatesCount.value = mates.length
    }
  } catch (error) {
    console.error('Failed to load trusted mates count:', error)
  }
}

async function forgetAllMates() {
  const confirmed = await confirm({
    title: 'Alle Mates vergessen?',
    message: 'Wirklich ALLE gepairten Mates vergessen? Diese müssen danach erneut gepairt werden.',
    type: 'danger',
    confirmText: 'Alle vergessen'
  })
  if (!confirmed) return

  forgettingMates.value = true
  try {
    const response = await secureFetch('/api/pairing/trusted', { method: 'DELETE' })
    if (response.ok) {
      trustedMates.value = []
      trustedMatesCount.value = 0
      success('Alle Mates wurden vergessen!')
    } else {
      throw new Error('Failed to forget mates')
    }
  } catch (err) {
    console.error('Failed to forget all mates:', err)
    errorToast('Fehler beim Vergessen der Mates')
  } finally {
    forgettingMates.value = false
  }
}

async function removeTrustedMate(mateId) {
  const confirmed = await confirm({
    title: 'Mate vergessen?',
    message: 'Dieser Mate muss danach erneut gepairt werden.',
    type: 'warning',
    confirmText: 'Vergessen'
  })
  if (!confirmed) return

  removingMateId.value = mateId
  try {
    const response = await secureFetch(`/api/pairing/trusted/${mateId}`, { method: 'DELETE' })
    if (response.ok) {
      await loadTrustedMatesCount()
      success('Mate wurde vergessen!')
    } else {
      throw new Error('Failed to remove mate')
    }
  } catch (err) {
    console.error('Failed to remove mate:', err)
    errorToast('Fehler beim Entfernen des Mates')
  } finally {
    removingMateId.value = null
  }
}

// Pending Pairing Functions
async function loadPendingPairingRequests() {
  try {
    const response = await fetch('/api/pairing/pending')
    if (response.ok) {
      pendingPairingRequests.value = await response.json()
    }
  } catch (error) {
    console.error('Failed to load pending pairing requests:', error)
  }
}

async function approvePairingRequest(requestId) {
  processingPairing.value = true
  try {
    const response = await secureFetch(`/api/pairing/approve/${requestId}`, { method: 'POST' })
    if (response.ok) {
      pendingPairingRequests.value = pendingPairingRequests.value.filter(r => r.requestId !== requestId)
      await loadTrustedMatesCount()
      success('Mate erfolgreich verbunden!')
    } else {
      throw new Error('Failed to approve pairing')
    }
  } catch (error) {
    console.error('Failed to approve pairing:', error)
    errorToast('Fehler beim Genehmigen des Pairings')
  } finally {
    processingPairing.value = false
  }
}

async function rejectPairingRequest(requestId) {
  processingPairing.value = true
  try {
    const response = await secureFetch(`/api/pairing/reject/${requestId}`, { method: 'POST' })
    if (response.ok) {
      pendingPairingRequests.value = pendingPairingRequests.value.filter(r => r.requestId !== requestId)
      success('Pairing-Anfrage abgelehnt')
    }
  } catch (error) {
    console.error('Failed to reject pairing:', error)
  } finally {
    processingPairing.value = false
  }
}

// Start polling for pending pairings when mates tab is active
function startPairingPoll() {
  if (pairingPollInterval) return
  loadPendingPairingRequests()
  pairingPollInterval = setInterval(loadPendingPairingRequests, 3000)
}

function stopPairingPoll() {
  if (pairingPollInterval) {
    clearInterval(pairingPollInterval)
    pairingPollInterval = null
  }
}

function getMateTypeName(mateType) {
  const names = {
    'mail': 'Email Client',
    'os': 'Betriebssystem',
    'browser': 'Browser',
    'office': 'Office Suite'
  }
  return names[mateType] || mateType || 'Unbekannt'
}

// Watch for changes from store
watch(() => settingsStore.settings, (newSettings) => {
  settings.value = { ...newSettings }
}, { deep: true })

function close() {
  stopPairingPoll()
  emit('close')
}

async function save() {
  saving.value = true
  try {
    // Merge sampling params into settings before saving
    const mergedSettings = {
      ...settings.value,
      ...samplingParams.value
    }

    // Save general settings + sampling parameters
    settingsStore.updateSettings(mergedSettings)

    // Apply streaming setting to chatStore
    if (chatStore.streamingEnabled !== mergedSettings.streamingEnabled) {
      chatStore.toggleStreaming()
    }

    // Save model selection settings to backend
    await api.updateModelSelectionSettings(modelSelectionSettings.value)

    // Save web search settings
    await saveWebSearchSettings()

    // Save personal info if on that tab
    if (personalInfoTabRef.value && activeTab.value === 'personal') {
      await personalInfoTabRef.value.save()
    }

    success('Einstellungen gespeichert')
    emit('save')
    close()
  } catch (error) {
    console.error('Failed to save settings:', error)
    errorToast('Fehler beim Speichern der Einstellungen')
  } finally {
    saving.value = false
  }
}

async function handleResetAll() {
  // Build list of selected categories
  const selectedCategories = []
  if (resetSelection.value.chats) selectedCategories.push('Chats & Nachrichten')
  if (resetSelection.value.projects) selectedCategories.push('Projekte & Dateien')
  if (resetSelection.value.customModels) selectedCategories.push('Custom Models')
  if (resetSelection.value.settings) selectedCategories.push('Einstellungen')
  if (resetSelection.value.personalInfo) selectedCategories.push('Persönliche Informationen')
  if (resetSelection.value.templates) selectedCategories.push('Templates & Prompts')
  if (resetSelection.value.stats) selectedCategories.push('Statistiken')

  if (selectedCategories.length === 0) {
    errorToast('Bitte wählen Sie mindestens eine Kategorie aus!')
    return
  }

  // Confirmation with selected categories
  const confirmation1 = confirm(
    '⚠️ ACHTUNG! ⚠️\n\n' +
    'Sie sind dabei, folgende Daten zu löschen:\n\n' +
    selectedCategories.map(cat => `• ${cat}`).join('\n') + '\n\n' +
    'Diese Aktion kann NICHT rückgängig gemacht werden!\n\n' +
    'Möchten Sie wirklich fortfahren?'
  )

  if (!confirmation1) return

  const confirmation2 = confirm(
    '⚠️ LETZTE WARNUNG! ⚠️\n\n' +
    'Dies ist Ihre letzte Chance!\n\n' +
    'Die ausgewählten Daten werden unwiderruflich gelöscht.\n' +
    'Die Anwendung wird danach neu geladen.\n\n' +
    'Sind Sie ABSOLUT SICHER?'
  )

  if (!confirmation2) return

  resetting.value = true

  try {
    // Call backend to delete selected data
    await api.resetSelectedData(resetSelection.value)

    // Show success message
    success('Ausgewählte Daten wurden gelöscht. Die Anwendung wird jetzt neu geladen...')

    // Wait a moment to show the message
    await new Promise(resolve => setTimeout(resolve, 1500))

    // Reload the page to reset to initial state
    window.location.reload()
  } catch (error) {
    console.error('Failed to reset data:', error)
    errorToast('Fehler beim Löschen der Daten: ' + (error.message || 'Unbekannter Fehler'))
    resetting.value = false
  }
}

async function resetToDefaults() {
  const confirmed = await confirm({
    title: 'Einstellungen zurücksetzen?',
    message: 'Alle Einstellungen auf Standard zurücksetzen?',
    type: 'warning',
    confirmText: 'Zurücksetzen'
  })
  if (confirmed) {
    settingsStore.resetToDefaults()
    settings.value = { ...settingsStore.settings }
    success('Einstellungen zurückgesetzt')
  }
}

// System Prompts Functions
async function loadSystemPrompts() {
  try {
    const prompts = await api.getAllSystemPrompts()
    systemPrompts.value = prompts
  } catch (error) {
    console.error('Failed to load system prompts:', error)
    errorToast('Fehler beim Laden der System-Prompts')
  }
}

function resetPromptForm() {
  promptForm.value = {
    name: '',
    content: '',
    isDefault: false
  }
}

function editSystemPrompt(prompt) {
  editingPrompt.value = prompt
  promptForm.value = {
    name: prompt.name,
    content: prompt.content,
    isDefault: prompt.isDefault || false
  }
  showPromptEditor.value = true
}

async function saveSystemPrompt() {
  try {
    if (!promptForm.value.name.trim() || !promptForm.value.content.trim()) {
      errorToast('Name und Inhalt dürfen nicht leer sein')
      return
    }

    if (editingPrompt.value) {
      // Update existing prompt
      await api.updateSystemPrompt(editingPrompt.value.id, promptForm.value)
      success('System-Prompt erfolgreich aktualisiert!')
    } else {
      // Create new prompt
      await api.createSystemPrompt(promptForm.value)
      success('System-Prompt erfolgreich erstellt!')
    }

    // Reload prompts and close editor
    await loadSystemPrompts()
    showPromptEditor.value = false
    editingPrompt.value = null
    resetPromptForm()
  } catch (error) {
    console.error('Failed to save system prompt:', error)
    errorToast('Fehler beim Speichern des System-Prompts')
  }
}

async function deleteSystemPrompt(promptId) {
  const confirmed = await confirmDelete('System-Prompt', 'Diese Aktion kann nicht rückgängig gemacht werden.')
  if (!confirmed) return

  try {
    await api.deleteSystemPrompt(promptId)
    success('System-Prompt erfolgreich gelöscht!')
    await loadSystemPrompts()
  } catch (error) {
    console.error('Failed to delete system prompt:', error)
    errorToast('Fehler beim Löschen des System-Prompts')
  }
}

async function activateSystemPrompt(prompt) {
  // Wenn bereits aktiv, nichts tun
  if (prompt.isDefault) {
    return
  }

  try {
    // 1. Als Standard in DB speichern (neuer dedizierter Endpoint)
    await api.setDefaultSystemPrompt(prompt.id)

    // 2. chatStore aktualisieren (für TopBar und Chat-Anfragen)
    chatStore.systemPrompt = prompt.content
    chatStore.systemPromptTitle = prompt.name

    // 3. Liste neu laden (UI aktualisieren)
    await loadSystemPrompts()

    success(`"${prompt.name}" aktiviert!`)
    console.log(`✅ System-Prompt "${prompt.name}" aktiviert und in chatStore gesetzt`)
  } catch (error) {
    console.error('Failed to activate system prompt:', error)
    errorToast('Fehler beim Aktivieren des System-Prompts')
  }
}

// Web Search Functions
async function loadWebSearchSettings() {
  try {
    const response = await fetch('/api/search/settings')
    if (response.ok) {
      const data = await response.json()
      webSearchSettings.value = {
        braveApiKey: data.braveApiKey || '',
        braveConfigured: data.braveConfigured || false,
        searchCount: data.searchCount || 0,
        searchLimit: data.searchLimit || 2000,
        remainingSearches: data.remainingSearches || 2000,
        currentMonth: data.currentMonth || '',
        customSearxngInstance: data.customSearxngInstance || '',
        searxngInstances: data.searxngInstances?.length > 0 ? data.searxngInstances : [...defaultSearxngInstances],
        searxngTotalCount: data.searxngTotalCount || 0,
        searxngMonthCount: data.searxngMonthCount || 0,
        // Feature Flags
        queryOptimizationEnabled: data.queryOptimizationEnabled ?? true,
        contentScrapingEnabled: data.contentScrapingEnabled ?? true,
        multiQueryEnabled: data.multiQueryEnabled ?? false,
        reRankingEnabled: data.reRankingEnabled ?? true,
        queryOptimizationModel: data.queryOptimizationModel || 'llama3.2:3b',
        effectiveOptimizationModel: data.effectiveOptimizationModel || null,
        // UI Animation
        webSearchAnimation: data.webSearchAnimation || 'data-wave'
      }
    }
  } catch (error) {
    console.error('Failed to load web search settings:', error)
    // Fallback auf Defaults
    webSearchSettings.value.searxngInstances = [...defaultSearxngInstances]
  }
}

async function saveWebSearchSettings() {
  try {
    const response = await secureFetch('/api/search/settings', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        braveApiKey: webSearchSettings.value.braveApiKey,
        customSearxngInstance: webSearchSettings.value.customSearxngInstance,
        searxngInstances: webSearchSettings.value.searxngInstances.filter(i => i && i.trim()),
        // Feature Flags
        queryOptimizationEnabled: webSearchSettings.value.queryOptimizationEnabled,
        contentScrapingEnabled: webSearchSettings.value.contentScrapingEnabled,
        multiQueryEnabled: webSearchSettings.value.multiQueryEnabled,
        reRankingEnabled: webSearchSettings.value.reRankingEnabled,
        queryOptimizationModel: webSearchSettings.value.queryOptimizationModel,
        // UI Animation
        webSearchAnimation: webSearchSettings.value.webSearchAnimation
      })
    })
    if (!response.ok) {
      throw new Error('Failed to save')
    }
    // Reload to get updated status
    await loadWebSearchSettings()
  } catch (error) {
    console.error('Failed to save web search settings:', error)
    throw error
  }
}

async function testBraveSearch() {
  testingSearch.value = true
  try {
    await saveWebSearchSettings()
    const response = await secureFetch('/api/search/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'test' })
    })
    const result = await response.json()
    if (result.success) {
      success(`✓ Suche funktioniert! ${result.resultCount} Ergebnisse via ${result.source}`)
      await loadWebSearchSettings()
    } else {
      errorToast(`✗ Suche fehlgeschlagen: ${result.error || 'Unbekannter Fehler'}`)
    }
  } catch (error) {
    errorToast('✗ Test fehlgeschlagen: ' + error.message)
  } finally {
    testingSearch.value = false
  }
}

async function testCustomSearxng() {
  if (!webSearchSettings.value.customSearxngInstance) return
  testingSearch.value = true
  try {
    await saveWebSearchSettings()
    const response = await secureFetch('/api/search/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'test' })
    })
    const result = await response.json()
    if (result.success) {
      success(`✓ SearXNG funktioniert! ${result.resultCount} Ergebnisse`)
    } else {
      errorToast(`✗ SearXNG nicht erreichbar: ${result.error || 'Unbekannter Fehler'}`)
    }
  } catch (error) {
    errorToast('✗ Test fehlgeschlagen: ' + error.message)
  } finally {
    testingSearch.value = false
  }
}

function addSearxngInstance() {
  webSearchSettings.value.searxngInstances.push('')
}

function removeSearxngInstance(index) {
  webSearchSettings.value.searxngInstances.splice(index, 1)
}

async function resetSearxngInstances() {
  const confirmed = await confirm({
    title: 'Instanzen zurücksetzen?',
    message: 'Fallback-Instanzen auf Standard zurücksetzen?',
    type: 'warning',
    confirmText: 'Zurücksetzen'
  })
  if (confirmed) {
    webSearchSettings.value.searxngInstances = [...defaultSearxngInstances]
    success('Instanzen zurückgesetzt')
  }
}
</script>

<style scoped>
/* Custom Scrollbar */
.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(156, 163, 175, 0.3);
  border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(156, 163, 175, 0.5);
}

/* Modal Transitions */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active > div,
.modal-leave-active > div {
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.modal-enter-from > div {
  transform: scale(0.9) translateY(-20px);
}

.modal-leave-to > div {
  transform: scale(0.9) translateY(20px);
}

/* Font Size Slider - High Contrast */
.font-size-slider {
  background: linear-gradient(to right, #3b82f6, #60a5fa);
  border: 2px solid #60a5fa;
}

:deep(.dark) .font-size-slider,
.dark .font-size-slider {
  background: linear-gradient(to right, #1e40af, #3b82f6);
  border: 2px solid #60a5fa;
}

/* Slider Thumb - WebKit (Chrome, Safari) */
.font-size-slider::-webkit-slider-thumb {
  appearance: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ffffff 0%, #e5e7eb 100%);
  border: 3px solid #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), 0 0 0 2px rgba(59, 130, 246, 0.3);
  transition: all 0.2s ease;
}

.font-size-slider::-webkit-slider-thumb:hover {
  transform: scale(1.15);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4), 0 0 0 4px rgba(59, 130, 246, 0.4);
}

/* Slider Thumb - Firefox */
.font-size-slider::-moz-range-thumb {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ffffff 0%, #e5e7eb 100%);
  border: 3px solid #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3), 0 0 0 2px rgba(59, 130, 246, 0.3);
}

.font-size-slider::-moz-range-thumb:hover {
  transform: scale(1.15);
}

/* Slider Track - Firefox */
.font-size-slider::-moz-range-track {
  background: linear-gradient(to right, #3b82f6, #60a5fa);
  border-radius: 8px;
  height: 12px;
}
</style>
