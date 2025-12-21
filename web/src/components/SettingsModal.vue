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
          <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <GlobeAltIcon class="w-5 h-5 text-blue-500" />
              {{ $t('settings.general.title') }}
            </h3>

            <!-- Language -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                <LanguageIcon class="w-4 h-4" />
                {{ $t('settings.general.language') }}
              </label>
              <select
                v-model="settings.language"
                class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-fleet-orange-500 focus:border-transparent"
              >
                <option value="de">🇩🇪 Deutsch</option>
                <option value="en">🇬🇧 English</option>
              </select>
            </div>

            <!-- Theme -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                <SunIcon class="w-4 h-4" />
                {{ $t('settings.general.theme') }}
              </label>
              <select
                v-model="settings.theme"
                class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-fleet-orange-500 focus:border-transparent"
              >
                <option value="light">☀️ {{ $t('common.light') }}</option>
                <option value="dark">🌙 {{ $t('common.dark') }}</option>
                <option value="auto">🔄 {{ $t('common.auto') }}</option>
              </select>
            </div>

            <!-- TopBar Toggle -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <Bars3Icon class="w-4 h-4 text-blue-500" />
                    {{ $t('settings.general.showTopBar') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.general.showTopBarDesc') }}
                  </p>
                </div>
                <label class="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" v-model="settings.showTopBar" @change="saveTopBarSetting" class="sr-only peer">
                  <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-fleet-orange-300 dark:peer-focus:ring-fleet-orange-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-fleet-orange-500"></div>
                </label>
              </div>
            </div>

            <!-- Willkommen-Anzeige Toggle -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <SparklesIcon class="w-4 h-4 text-fleet-orange-500" />
                    {{ $t('settings.general.showWelcomeTiles') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.general.showWelcomeTilesDesc') }}
                  </p>
                </div>
                <label class="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" v-model="settings.showWelcomeTiles" class="sr-only peer">
                  <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-fleet-orange-300 dark:peer-focus:ring-fleet-orange-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-fleet-orange-500"></div>
                </label>
              </div>
            </div>

            <!-- Modus-Wechsel-Nachrichten Toggle -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <ArrowsRightLeftIcon class="w-4 h-4 text-purple-500" />
                    {{ $t('settings.general.showModeSwitchMessages') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.general.showModeSwitchMessagesDesc') }}
                  </p>
                </div>
                <label class="relative inline-flex items-center cursor-pointer">
                  <input type="checkbox" v-model="settings.showModeSwitchMessages" class="sr-only peer">
                  <div class="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-purple-300 dark:peer-focus:ring-purple-800 rounded-full peer dark:bg-gray-600 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-500 peer-checked:bg-purple-500"></div>
                </label>
              </div>
            </div>

            <!-- Schriftgröße -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16m-7 6h7" />
                </svg>
                {{ $t('settings.general.fontSize') }}
              </label>
              <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
                {{ $t('settings.general.fontSizeDesc') }}
              </p>
              <!-- Slider mit Wertanzeige -->
              <div class="space-y-3">
                <div class="flex items-center gap-4">
                  <span class="text-xs text-gray-500 dark:text-gray-400 w-8">50%</span>
                  <input
                    type="range"
                    min="50"
                    max="150"
                    step="5"
                    :value="settings.fontSize || 100"
                    @input="setFontSize(Number($event.target.value))"
                    class="font-size-slider flex-1 h-3 rounded-lg appearance-none cursor-pointer"
                  />
                  <span class="text-xs text-gray-500 dark:text-gray-400 w-10">150%</span>
                </div>
                <!-- Aktuelle Größe und Reset-Button -->
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <span class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ settings.fontSize || 100 }}%</span>
                    <span class="text-xs text-gray-500" :style="{ fontSize: (settings.fontSize || 100) * 0.14 + 'px' }">
                      {{ $t('settings.general.sampleText') }}
                    </span>
                  </div>
                  <button
                    type="button"
                    @click="setFontSize(100)"
                    class="px-3 py-1 text-xs rounded-lg bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 transition-colors"
                  >
                    {{ $t('common.reset') }}
                  </button>
                </div>
              </div>
            </div>

            <!-- CPU-Only Mode Toggle (für Screencasts/Präsentationen) -->
            <div class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <CpuChipIcon class="w-4 h-4 text-orange-500" />
                    {{ $t('settings.hardware.cpuOnly') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.hardware.cpuOnlyDesc') }}
                  </p>
                </div>
                <ToggleSwitch v-model="settings.cpuOnly" color="orange" />
              </div>
              <!-- Info Box -->
              <div v-if="settings.cpuOnly" class="mt-2 p-2 rounded-lg bg-orange-50 dark:bg-orange-900/30 border border-orange-200 dark:border-orange-800">
                <p class="text-xs text-orange-800 dark:text-orange-200 flex items-center gap-2">
                  <ExclamationTriangleIcon class="w-4 h-4 flex-shrink-0" />
                  <span>{{ $t('settings.hardware.cpuOnlyActive') }}</span>
                </p>
              </div>
            </div>

          </section>
          </div>

          <!-- TAB: Fleet Mates -->
          <div v-if="activeTab === 'mates'">
            <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                <UsersIcon class="w-5 h-5 text-blue-500" />
                {{ $t('mates.title') }}
              </h3>
              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {{ $t('settings.mates.subtitle') }}
              </p>

              <!-- Pending Pairing Requests -->
              <div v-if="pendingPairingRequests.length > 0" class="mb-6">
                <h4 class="text-md font-semibold text-amber-600 dark:text-amber-400 mb-3 flex items-center gap-2">
                  <ExclamationTriangleIcon class="w-5 h-5" />
                  {{ $t('settings.mates.pendingRequests') }} ({{ pendingPairingRequests.length }})
                </h4>
                <div class="space-y-3">
                  <div
                    v-for="request in pendingPairingRequests"
                    :key="request.requestId"
                    class="p-4 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700/50"
                  >
                    <div class="flex items-center justify-between">
                      <div class="flex items-center gap-3">
                        <div class="p-2 rounded-lg" :class="getMateTypeColor(request.mateType)">
                          <component :is="getMateTypeIcon(request.mateType)" class="w-5 h-5 text-white" />
                        </div>
                        <div>
                          <h5 class="font-semibold text-gray-900 dark:text-white">{{ request.mateName }}</h5>
                          <p class="text-xs text-gray-500 dark:text-gray-400">{{ getMateTypeLabel(request.mateType) }}</p>
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        <!-- Pairing Code Display -->
                        <div class="flex gap-1 mr-4">
                          <span
                            v-for="(digit, index) in request.pairingCode.split('')"
                            :key="index"
                            class="w-8 h-10 flex items-center justify-center text-lg font-mono font-bold bg-gradient-to-br from-blue-500 to-cyan-500 text-white rounded"
                          >
                            {{ digit }}
                          </span>
                        </div>
                        <button
                          @click="rejectPairingRequest(request.requestId)"
                          :disabled="processingPairing"
                          class="px-3 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition-all disabled:opacity-50"
                        >
                          <XMarkIcon class="w-5 h-5" />
                        </button>
                        <button
                          @click="approvePairingRequest(request.requestId)"
                          :disabled="processingPairing"
                          class="px-3 py-2 bg-gradient-to-r from-green-500 to-emerald-600 text-white rounded-lg hover:from-green-600 hover:to-emerald-700 transition-all shadow-lg disabled:opacity-50"
                        >
                          <CheckIcon class="w-5 h-5" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Trusted Mates List -->
              <div>
                <h4 class="text-md font-semibold text-gray-700 dark:text-gray-300 mb-3 flex items-center gap-2">
                  <LinkIcon class="w-5 h-5" />
                  {{ $t('settings.mates.connectedMates') }} ({{ trustedMates.length }})
                </h4>

                <div v-if="trustedMates.length === 0" class="p-6 text-center rounded-xl bg-gray-100 dark:bg-gray-800 border border-dashed border-gray-300 dark:border-gray-600">
                  <UsersIcon class="w-12 h-12 text-gray-400 mx-auto mb-3" />
                  <p class="text-gray-500 dark:text-gray-400">{{ $t('mates.noMates') }}</p>
                  <p class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ $t('settings.mates.connectViaSystem') }}</p>
                </div>

                <div v-else class="space-y-3">
                  <div
                    v-for="mate in trustedMates"
                    :key="mate.mateId"
                    class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 flex items-center justify-between"
                  >
                    <div class="flex items-center gap-3">
                      <div class="p-2 rounded-lg" :class="getMateTypeColor(mate.mateType)">
                        <component :is="getMateTypeIcon(mate.mateType)" class="w-5 h-5 text-white" />
                      </div>
                      <div>
                        <h5 class="font-semibold text-gray-900 dark:text-white">{{ mate.name }}</h5>
                        <p class="text-xs text-gray-500 dark:text-gray-400">
                          {{ getMateTypeLabel(mate.mateType) }}
                          <span v-if="mate.lastSeen" class="ml-2">• {{ $t('mates.lastSeen') }}: {{ formatDateAbsolute(mate.lastSeen) }}</span>
                        </p>
                      </div>
                    </div>
                    <button
                      @click="removeTrustedMate(mate.mateId)"
                      :disabled="removingMateId === mate.mateId"
                      class="px-3 py-2 text-red-600 hover:text-red-700 dark:text-red-400 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all disabled:opacity-50"
                      :title="$t('mates.forgetDevice')"
                    >
                      <TrashIcon v-if="removingMateId !== mate.mateId" class="w-5 h-5" />
                      <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
                    </button>
                  </div>
                </div>

                <!-- Forget All Button -->
                <div v-if="trustedMates.length > 1" class="mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
                  <button
                    @click="forgetAllMates"
                    :disabled="forgettingMates"
                    class="w-full px-4 py-2 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg transition-all border border-red-200 dark:border-red-800 flex items-center justify-center gap-2"
                  >
                    <TrashIcon v-if="!forgettingMates" class="w-5 h-5" />
                    <ArrowPathIcon v-else class="w-5 h-5 animate-spin" />
                    {{ forgettingMates ? $t('settings.mates.forgetting') : $t('settings.mates.forgetAllButton') }}
                  </button>
                </div>
              </div>
            </section>

            <!-- Fleet Mates Model Assignment (integriert in Fleet Mates Tab) -->
            <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm mt-6">
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
                <CpuChipIcon class="w-5 h-5 text-purple-500" />
                {{ $t('settings.mates.modelAssignment') }}
              </h3>

              <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
                {{ $t('settings.mates.modelAssignmentDesc') }}
              </p>

              <div class="space-y-4">
                <!-- Email Model (Thunderbird Mate) -->
                <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                    <span class="text-xl">📧</span>
                    {{ $t('settings.models.emailModel') }}
                    <span class="text-xs text-gray-500 dark:text-gray-400">(Thunderbird Mate)</span>
                  </label>
                  <select
                    v-model="mateModels.emailModel"
                    @change="saveEmailModel"
                    class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">-- {{ $t('common.default') }} (llama3.2:3b) --</option>
                    <option v-for="model in fastModels" :key="model.name" :value="model.name">
                      {{ model.name }} ({{ formatSize(model.size) }})
                    </option>
                  </select>
                  <p class="text-xs text-gray-500 mt-1">{{ $t('settings.models.emailModelDesc') }}</p>
                </div>

                <!-- Document Model (Writer Mate) -->
                <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                    <span class="text-xl">✍️</span>
                    {{ $t('settings.models.documentModel') }}
                    <span class="text-xs text-gray-500 dark:text-gray-400">(Writer Mate)</span>
                  </label>
                  <select
                    v-model="mateModels.documentModel"
                    @change="saveDocumentModel"
                    class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-green-500"
                  >
                    <option value="">-- {{ $t('common.default') }} --</option>
                    <option v-for="model in availableModels" :key="model.name" :value="model.name">
                      {{ model.name }} ({{ formatSize(model.size) }})
                    </option>
                  </select>
                  <p class="text-xs text-gray-500 mt-1">{{ $t('settings.models.documentModelDesc') }}</p>
                </div>

                <!-- Log Analysis Model (OS Mate) -->
                <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                    <span class="text-xl">📊</span>
                    Log-Analyse-Modell
                    <span class="text-xs text-gray-500 dark:text-gray-400">(OS Mate)</span>
                  </label>
                  <select
                    v-model="mateModels.logAnalysisModel"
                    @change="saveLogAnalysisModel"
                    class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-orange-500"
                  >
                    <option value="">-- Standard --</option>
                    <option v-for="model in availableModels" :key="model.name" :value="model.name">
                      {{ model.name }} ({{ formatSize(model.size) }})
                    </option>
                  </select>
                  <p class="text-xs text-gray-500 mt-1">Für Log-Datei-Analyse und Fehlersuche</p>
                </div>

                <!-- Coder Model (FleetCoder) -->
                <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                    <span class="text-xl">💻</span>
                    Coder-Modell
                    <span class="text-xs text-gray-500 dark:text-gray-400">(FleetCoder)</span>
                  </label>
                  <select
                    v-model="mateModels.coderModel"
                    @change="saveCoderModel"
                    class="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-cyan-500"
                  >
                    <option value="">-- Standard --</option>
                    <option v-for="model in availableModels" :key="model.name" :value="model.name">
                      {{ model.name }} ({{ formatSize(model.size) }})
                    </option>
                  </select>
                  <p class="text-xs text-gray-500 mt-1">Für Code-Assistenz und Programmierung (größeres Modell empfohlen: 14B+)</p>
                </div>
              </div>

              <div class="mt-4 p-3 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700/50">
                <div class="flex items-start gap-2">
                  <InformationCircleIcon class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0 mt-0.5" />
                  <p class="text-xs text-blue-800 dark:text-blue-200">
                    <strong>Tipp:</strong> Für Email-Klassifizierung eignen sich schnelle Modelle wie <code>llama3.2:3b</code> oder <code>qwen2.5:7b</code>.
                  </p>
                </div>
              </div>
            </section>
          </div>

          <!-- TAB: LLM Provider -->
          <div v-if="activeTab === 'providers'">
            <ProviderSettings />
          </div>

          <!-- TAB: GPU/VRAM Settings -->
          <div v-if="activeTab === 'gpu'">
            <VRAMSettings />
          </div>

          <!-- TAB: Database (PostgreSQL + Vector DB) -->
          <div v-if="activeTab === 'database'">
            <PostgreSQLMigration @status-change="onPostgresStatusChange" />
          </div>

          <!-- TAB: Model Parameters -->
          <div v-if="activeTab === 'parameters'">
          <!-- Model Parameters -->
          <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <AdjustmentsHorizontalIcon class="w-5 h-5 text-orange-500" />
              🎛️ LLM Sampling Parameter
            </h3>

            <SimpleSamplingParams
              v-model="samplingParams"
            />
          </section>
          </div>

          <!-- TAB: Templates / System Prompts -->
          <div v-if="activeTab === 'templates'">
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
                  @click="showPromptEditor = true; editingPrompt = null; resetPromptForm()"
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
                        @click="activateSystemPrompt(prompt)"
                        class="p-1.5 rounded transition-colors"
                        :class="prompt.isDefault
                          ? 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30'
                          : 'text-gray-400 dark:text-gray-500 hover:text-green-600 dark:hover:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20'"
                        :title="prompt.isDefault ? $t('settings.prompts.activePrompt') : $t('settings.prompts.activateAsDefault')"
                      >
                        <CheckCircleIcon class="w-4 h-4" />
                      </button>
                      <button
                        @click="editSystemPrompt(prompt)"
                        class="p-1.5 text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors"
                        :title="$t('common.edit')"
                      >
                        <WrenchScrewdriverIcon class="w-4 h-4" />
                      </button>
                      <button
                        @click="deleteSystemPrompt(prompt.id)"
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
          </div>

          <!-- TAB: Personal Info -->
          <div v-if="activeTab === 'personal'">
            <PersonalInfoTab ref="personalInfoTabRef" />
          </div>

          <!-- TAB: Agents -->
          <div v-if="activeTab === 'agents'">
          <!-- Vision Settings -->
          <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <PhotoIcon class="w-5 h-5 text-indigo-500" />
              {{ $t('settings.agents.visionModel') }}
            </h3>

            <!-- Auto-select Vision Model -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <EyeIcon class="w-4 h-4" />
                    {{ $t('settings.agents.autoVisionModel') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.agents.autoVisionModelDesc') }}
                  </p>
                </div>
                <ToggleSwitch v-model="settings.autoSelectVisionModel" color="indigo" />
              </div>
            </div>

            <!-- Preferred Vision Model -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('settings.agents.preferredVisionModel') }}
              </label>
              <select
                v-model="modelSelectionSettings.visionModel"
                class="w-full px-4 py-2.5 border border-gray-300 dark:border-gray-600 rounded-xl bg-white dark:bg-gray-700 text-gray-900 dark:text-white transition-all focus:ring-2 focus:ring-indigo-500"
              >
                <option v-for="model in visionModels" :key="model" :value="model">
                  {{ model }}
                </option>
              </select>
              <p v-if="visionModels.length > 0" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                {{ visionModels.length }} {{ $t('settings.agents.visionModelsAvailable') }}
              </p>
              <p v-else class="mt-2 text-xs text-yellow-600 dark:text-yellow-400">
                ⚠️ {{ $t('settings.agents.noVisionModels') }}
              </p>
            </div>

            <!-- Vision Chaining -->
            <div class="mb-4 p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <LinkIcon class="w-4 h-4" />
                    {{ $t('settings.agents.visionChaining') }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.agents.visionChainingDesc') }}
                  </p>
                </div>
                <ToggleSwitch v-model="modelSelectionSettings.visionChainingEnabled" color="indigo" />
              </div>
            </div>

            <!-- Web Search: Think First -->
            <div class="mb-4 p-4 rounded-xl bg-blue-50/50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700">
              <div class="flex items-center justify-between">
                <div class="flex-1">
                  <label class="block text-sm font-semibold text-gray-700 dark:text-gray-300 flex items-center gap-2">
                    <LightBulbIcon class="w-4 h-4 text-blue-500" />
                    {{ $t('settings.agents.thinkFirst') || '🧠 Websuche: Erst nachdenken' }}
                  </label>
                  <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                    {{ $t('settings.agents.thinkFirstDesc') || 'LLM antwortet erst selbst, bei Unsicherheit → automatische Websuche' }}
                  </p>
                </div>
                <ToggleSwitch v-model="webSearchThinkFirst" color="blue" />
              </div>
              <!-- Explanation -->
              <div v-if="webSearchThinkFirst" class="mt-3 p-2 bg-green-50 dark:bg-green-900/30 rounded-lg">
                <p class="text-xs text-green-700 dark:text-green-300">
                  ✅ <strong>Aktiv:</strong> Das LLM versucht erst selbst zu antworten. Nur bei Unsicherheit wird automatisch eine Websuche durchgeführt.
                </p>
              </div>
              <div v-else class="mt-3 p-2 bg-orange-50 dark:bg-orange-900/30 rounded-lg">
                <p class="text-xs text-orange-700 dark:text-orange-300">
                  ⚡ <strong>Sofort-Modus:</strong> Bei aktivierter Experten-Websuche wird sofort gesucht (schneller, aber mehr API-Calls).
                </p>
              </div>
            </div>
          </section>

          <!-- OS Mate - File Search (RAG) -->
          <section class="bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-800 p-5 rounded-xl border border-gray-200/50 dark:border-gray-700/50 shadow-sm mt-6">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4 flex items-center gap-2">
              <FolderIcon class="w-5 h-5 text-amber-500" />
              {{ $t('settings.agents.osMateTitle') }}
            </h3>
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
              {{ $t('settings.agents.osMateDesc') }}
            </p>

            <!-- File Search Status -->
            <div v-if="fileSearchStatus" class="mb-4 p-3 rounded-lg bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800">
              <div class="flex items-center gap-2 text-sm text-blue-700 dark:text-blue-300">
                <span v-if="fileSearchStatus.indexingInProgress" class="animate-spin">⏳</span>
                <span v-else>📚</span>
                <span>
                  {{ fileSearchStatus.indexedFileCount }} {{ $t('settings.agents.filesIndexed') }}
                  <span v-if="fileSearchStatus.locateAvailable" class="text-xs text-green-600 dark:text-green-400 ml-2">({{ $t('settings.agents.locateAvailable') }})</span>
                </span>
              </div>
            </div>

            <!-- Search Folders List -->
            <div class="space-y-3 mb-4">
              <div v-for="folder in fileSearchFolders" :key="folder.folderId"
                   class="p-3 rounded-lg bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 flex items-center justify-between">
                <div class="flex items-center gap-3 flex-1">
                  <FolderOpenIcon class="w-5 h-5 text-amber-500" />
                  <div class="flex-1 min-w-0">
                    <div class="font-medium text-gray-900 dark:text-white truncate">{{ folder.name }}</div>
                    <div class="text-xs text-gray-500 dark:text-gray-400 truncate">{{ folder.folderPath }}</div>
                    <div class="text-xs text-gray-400 dark:text-gray-500">
                      {{ folder.fileCount || 0 }} {{ $t('settings.agents.files') }}
                      <span v-if="folder.lastIndexed" class="ml-2">• {{ $t('settings.agents.indexed') }}: {{ formatDateAbsolute(folder.lastIndexed) }}</span>
                    </div>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <button @click="reindexFolder(folder.folderId)"
                          class="p-1.5 rounded hover:bg-blue-100 dark:hover:bg-blue-900/30 text-blue-600 dark:text-blue-400"
                          :title="$t('settings.agents.reindex')">
                    <ArrowPathIcon class="w-4 h-4" />
                  </button>
                  <button @click="removeSearchFolder(folder.folderId)"
                          class="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-red-600 dark:text-red-400"
                          :title="$t('common.remove')">
                    <TrashIcon class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <div v-if="fileSearchFolders.length === 0"
                   class="p-4 rounded-lg bg-gray-100/50 dark:bg-gray-800/30 border border-dashed border-gray-300 dark:border-gray-600 text-center">
                <FolderIcon class="w-8 h-8 text-gray-400 mx-auto mb-2" />
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('settings.agents.noFoldersConfigured') }}</p>
              </div>
            </div>

            <!-- Add Folder Form -->
            <div class="p-4 rounded-xl bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">{{ $t('settings.agents.addFolder') }}</label>
              <div class="flex gap-2">
                <input type="text" v-model="newFolderPath"
                       :placeholder="$t('settings.agents.folderPathPlaceholder')"
                       class="flex-1 px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-amber-500" />
                <button @click="addSearchFolder"
                        :disabled="!newFolderPath"
                        class="px-4 py-2 bg-amber-500 hover:bg-amber-600 disabled:bg-gray-300 disabled:cursor-not-allowed text-white rounded-lg text-sm font-medium transition-colors">
                  {{ $t('common.add') }}
                </button>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                {{ $t('settings.agents.folderPathExample') }}
              </p>
            </div>
          </section>
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
          </div>

          <!-- TAB: Danger Zone -->
          <div v-if="activeTab === 'danger'">
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
                          v-model="resetSelection.chats"
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
                          v-model="resetSelection.projects"
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
                          v-model="resetSelection.customModels"
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
                          v-model="resetSelection.settings"
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
                          v-model="resetSelection.personalInfo"
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
                          v-model="resetSelection.templates"
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
                          v-model="resetSelection.stats"
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
  LightBulbIcon
} from '@heroicons/vue/24/outline'
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
import { filterVisionModels, filterCodeModels } from '../utils/modelFilters'

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
  }
})

// Sync showWelcomeTiles with settingsStore (persistent)
watch(() => settings.value.showWelcomeTiles, async (newValue) => {
  settingsStore.settings.showWelcomeTiles = newValue
  // Save to database
  try {
    await secureFetch('/api/settings/show-welcome-tiles', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newValue)
    })
  } catch (error) {
    console.error('Failed to save showWelcomeTiles:', error)
  }
}, { immediate: false })

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

async function addSearchFolder() {
  if (!newFolderPath.value) return

  try {
    const response = await secureFetch('/api/file-search/folders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ folderPath: newFolderPath.value })
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
  { id: 'gpu', label: 'GPU/VRAM', icon: CpuChipIcon },
  { id: 'database', label: 'Datenbank', icon: ServerIcon },
  { id: 'parameters', label: 'Parameter', icon: AdjustmentsHorizontalIcon },
  { id: 'templates', label: 'System-Prompts', icon: DocumentTextIcon },
  { id: 'personal', label: 'Persönliche Daten', icon: UserIcon },
  { id: 'agents', label: 'Agents', icon: SparklesIcon },
  { id: 'web-search', label: 'Web-Suche', icon: MagnifyingGlassIcon },
  { id: 'voice', label: 'Sprache', icon: MicrophoneIcon },
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
  effectiveOptimizationModel: null  // Das tatsächlich verwendete Modell (nach Fallback)
})

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

// VoiceStore ref
const voiceStoreRef = ref(null)

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
  await loadShowWelcomeTiles()
  await loadFileSearchStatus()
})

async function loadShowWelcomeTiles() {
  try {
    const response = await fetch('/api/settings/show-welcome-tiles')
    if (response.ok) {
      const value = await response.json()
      settings.value.showWelcomeTiles = value
      settingsStore.settings.showWelcomeTiles = value
    }
  } catch (error) {
    console.error('Failed to load showWelcomeTiles:', error)
  }
}

// TopBar Setting speichern (sofort bei Änderung)
async function saveTopBarSetting() {
  try {
    // Update settingsStore
    settingsStore.settings.showTopBar = settings.value.showTopBar
    // Save to backend database
    await settingsStore.saveShowTopBarToBackend(settings.value.showTopBar)
    success('TopBar-Einstellung gespeichert')
  } catch (error) {
    console.error('Failed to save showTopBar:', error)
    errorToast('Fehler beim Speichern der TopBar-Einstellung')
  }
}

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
async function saveEmailModel() {
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

async function saveDocumentModel() {
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

async function saveLogAnalysisModel() {
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

async function saveCoderModel() {
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
        effectiveOptimizationModel: data.effectiveOptimizationModel || null
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
        queryOptimizationModel: webSearchSettings.value.queryOptimizationModel
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
