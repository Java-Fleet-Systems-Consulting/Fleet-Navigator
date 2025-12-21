<template>
  <Transition name="modal">
    <div v-if="show" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-[70] p-4">
      <div
        class="
          bg-white dark:bg-gray-800
          rounded-2xl shadow-2xl
          w-full max-w-4xl
          border border-gray-200 dark:border-gray-700
          flex flex-col
          overflow-hidden
        "
        style="height: 90vh; max-height: 90vh;"
      >
        <!-- Header mit Progress -->
        <div class="flex-shrink-0 bg-gradient-to-r from-purple-500/10 to-indigo-500/10 dark:from-purple-500/20 dark:to-indigo-500/20 border-b border-gray-200 dark:border-gray-700">
          <!-- Progress Bar -->
          <div class="px-6 pt-4">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium text-gray-600 dark:text-gray-400">
                {{ $t('expertWizard.stepOf', { current: currentStep, total: totalSteps }) }}
              </span>
              <span class="text-sm text-gray-500 dark:text-gray-500">
                {{ stepTitles[currentStep - 1] }}
              </span>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                class="bg-gradient-to-r from-purple-500 to-indigo-500 h-2 rounded-full transition-all duration-300"
                :style="{ width: `${(currentStep / totalSteps) * 100}%` }"
              ></div>
            </div>
          </div>

          <!-- Step Indicators -->
          <div class="flex items-center justify-center gap-2 px-6 py-4">
            <button
              v-for="step in totalSteps"
              :key="step"
              @click="goToStep(step)"
              :disabled="!canGoToStep(step)"
              class="
                flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium
                transition-all duration-200
              "
              :class="[
                currentStep === step
                  ? 'bg-purple-500 text-white shadow-lg'
                  : step < currentStep
                    ? 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 cursor-pointer hover:bg-purple-200 dark:hover:bg-purple-900/50'
                    : 'bg-gray-100 dark:bg-gray-700 text-gray-400 cursor-not-allowed'
              ]"
            >
              <span>{{ stepEmojis[step - 1] }}</span>
              <span class="hidden sm:inline">{{ stepTitles[step - 1] }}</span>
            </button>
          </div>
        </div>

        <!-- Content Area -->
        <div class="flex-1 overflow-y-auto p-6 min-h-0">
          <!-- Step 1: Modell wählen -->
          <div v-if="currentStep === 1" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step1.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step1.subtitle') }}</p>
            </div>

            <div v-if="isLoadingModels" class="flex justify-center py-12">
              <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-500"></div>
            </div>

            <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <button
                v-for="model in availableModels"
                :key="model.name"
                @click="selectModel(model)"
                class="
                  p-4 rounded-xl border-2 text-left
                  transition-all duration-200
                  hover:shadow-lg
                "
                :class="[
                  wizardData.selectedModel === model.name
                    ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/20 shadow-lg shadow-purple-500/20'
                    : 'border-gray-200 dark:border-gray-700 hover:border-purple-300 dark:hover:border-purple-600'
                ]"
              >
                <div class="flex items-start gap-3">
                  <div class="p-2 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-500 flex-shrink-0">
                    <CpuChipIcon class="w-6 h-6 text-white" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <h3 class="font-bold text-gray-900 dark:text-white truncate">
                      {{ model.displayName || model.name }}
                    </h3>
                    <div class="flex flex-wrap gap-2 mt-1">
                      <span v-if="model.size" class="text-xs px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
                        {{ formatSize(model.size) }}
                      </span>
                      <span v-if="model.category" class="text-xs px-2 py-0.5 rounded-full bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400">
                        {{ model.category }}
                      </span>
                      <span v-if="model.contextLength" class="text-xs px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400">
                        {{ formatContextLength(model.contextLength) }}
                      </span>
                    </div>
                    <p v-if="model.description" class="text-sm text-gray-500 dark:text-gray-400 mt-2 line-clamp-2">
                      {{ model.description }}
                    </p>
                  </div>
                  <div v-if="wizardData.selectedModel === model.name" class="flex-shrink-0">
                    <CheckCircleIcon class="w-6 h-6 text-purple-500" />
                  </div>
                </div>
              </button>
            </div>
          </div>

          <!-- Step 2: Werkzeuge -->
          <div v-if="currentStep === 2" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step2.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step2.subtitle') }}</p>
            </div>

            <div class="space-y-4">
              <!-- Websuche -->
              <div class="p-5 rounded-xl border-2 bg-gray-50 dark:bg-gray-700/80 border-gray-300 dark:border-gray-500 hover:border-blue-300 dark:hover:border-blue-600 transition-colors">
                <label class="flex items-start gap-4 cursor-pointer">
                  <input
                    type="checkbox"
                    v-model="wizardData.autoWebSearch"
                    class="w-5 h-5 mt-1 text-blue-500 rounded focus:ring-blue-500"
                  />
                  <div class="flex-1">
                    <div class="flex items-center gap-2">
                      <GlobeAltIcon class="w-5 h-5 text-blue-500" />
                      <span class="font-bold text-gray-900 dark:text-white">{{ $t('expertWizard.step2.webSearch') }}</span>
                    </div>
                    <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                      {{ $t('expertWizard.step2.webSearchDesc') }}
                    </p>

                    <!-- Websuche Optionen -->
                    <div v-if="wizardData.autoWebSearch" class="mt-4 space-y-3 pl-2 border-l-2 border-blue-200 dark:border-blue-800">
                      <div>
                        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">{{ $t('expertWizard.step2.searchDomains') }}</label>
                        <input
                          v-model="wizardData.searchDomains"
                          type="text"
                          :placeholder="$t('expertWizard.step2.searchDomainsPlaceholder')"
                          class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white"
                        />
                      </div>
                      <div>
                        <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">{{ $t('expertWizard.step2.maxResults', { count: wizardData.maxSearchResults }) }}</label>
                        <input
                          v-model.number="wizardData.maxSearchResults"
                          type="range"
                          min="1"
                          max="10"
                          class="w-full"
                        />
                      </div>
                    </div>
                  </div>
                </label>
              </div>

              <!-- Dateisuche -->
              <div class="p-5 rounded-xl border-2 bg-gray-50 dark:bg-gray-700/80 border-gray-300 dark:border-gray-500 hover:border-green-300 dark:hover:border-green-600 transition-colors">
                <label class="flex items-start gap-4 cursor-pointer">
                  <input
                    type="checkbox"
                    v-model="wizardData.autoFileSearch"
                    class="w-5 h-5 mt-1 text-green-500 rounded focus:ring-green-500"
                  />
                  <div class="flex-1">
                    <div class="flex items-center gap-2">
                      <DocumentMagnifyingGlassIcon class="w-5 h-5 text-green-500" />
                      <span class="font-bold text-gray-900 dark:text-white">{{ $t('expertWizard.step2.fileSearch') }}</span>
                    </div>
                    <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                      {{ $t('expertWizard.step2.fileSearchDesc') }}
                    </p>

                    <!-- Dateisuche Optionen -->
                    <div v-if="wizardData.autoFileSearch" class="mt-4 pl-2 border-l-2 border-green-200 dark:border-green-800">
                      <label class="block text-sm text-gray-600 dark:text-gray-400 mb-1">{{ $t('expertWizard.step2.documentDir') }}</label>
                      <div class="flex items-center gap-1">
                        <span class="text-xs text-gray-400">~/Dokumente/Fleet-Navigator/</span>
                        <input
                          v-model="wizardData.documentDirectory"
                          type="text"
                          placeholder="Name"
                          class="flex-1 px-2 py-1.5 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white"
                        />
                      </div>
                    </div>
                  </div>
                </label>
              </div>

              <!-- Vektordatenbank (Coming Soon) -->
              <div class="p-5 rounded-xl border-2 bg-gray-50 dark:bg-gray-700/80 border-gray-300 dark:border-gray-500 opacity-50">
                <div class="flex items-start gap-4">
                  <input
                    type="checkbox"
                    disabled
                    class="w-5 h-5 mt-1 text-purple-500 rounded cursor-not-allowed"
                  />
                  <div class="flex-1">
                    <div class="flex items-center gap-2">
                      <CircleStackIcon class="w-5 h-5 text-purple-400" />
                      <span class="font-bold text-gray-500 dark:text-gray-400">{{ $t('expertWizard.step2.vectorDb') }}</span>
                      <span class="text-xs px-2 py-0.5 rounded-full bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400">
                        {{ $t('common.comingSoon') }}
                      </span>
                    </div>
                    <p class="text-sm text-gray-400 mt-1">
                      {{ $t('expertWizard.step2.vectorDbDesc') }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Step 3: Parameter -->
          <div v-if="currentStep === 3" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step3.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step3.subtitle') }}</p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <!-- Context Size -->
              <div class="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">{{ $t('expertWizard.step3.contextSize') }}</label>
                  <span class="text-sm font-mono text-purple-600 dark:text-purple-400">
                    {{ wizardData.defaultNumCtx?.toLocaleString() }} {{ $t('expertWizard.step3.tokens') }}
                  </span>
                </div>
                <input
                  v-model.number="wizardData.defaultNumCtx"
                  type="range"
                  min="2048"
                  max="131072"
                  step="2048"
                  class="w-full"
                />
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  {{ $t('expertWizard.step3.contextSizeDesc') }}
                </p>
              </div>

              <!-- Max Tokens -->
              <div class="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">{{ $t('expertWizard.step3.maxResponseLength') }}</label>
                  <span class="text-sm font-mono text-purple-600 dark:text-purple-400">
                    {{ wizardData.defaultMaxTokens?.toLocaleString() }} {{ $t('expertWizard.step3.tokens') }}
                  </span>
                </div>
                <input
                  v-model.number="wizardData.defaultMaxTokens"
                  type="range"
                  min="256"
                  max="16384"
                  step="256"
                  class="w-full"
                />
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  {{ $t('expertWizard.step3.maxResponseLengthDesc') }}
                </p>
              </div>

              <!-- Temperature -->
              <div class="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">{{ $t('expertWizard.step3.creativity') }}</label>
                  <span class="text-sm font-mono text-purple-600 dark:text-purple-400">
                    {{ wizardData.defaultTemperature?.toFixed(1) }}
                  </span>
                </div>
                <input
                  v-model.number="wizardData.defaultTemperature"
                  type="range"
                  min="0"
                  max="2"
                  step="0.1"
                  class="w-full"
                />
                <div class="flex justify-between text-xs text-gray-400 mt-1">
                  <span>{{ $t('expertWizard.step3.precise') }}</span>
                  <span>{{ $t('expertWizard.step3.creative') }}</span>
                </div>
              </div>

              <!-- Top P -->
              <div class="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
                <div class="flex items-center justify-between mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">{{ $t('expertWizard.step3.topP') }}</label>
                  <span class="text-sm font-mono text-purple-600 dark:text-purple-400">
                    {{ wizardData.defaultTopP?.toFixed(2) }}
                  </span>
                </div>
                <input
                  v-model.number="wizardData.defaultTopP"
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  class="w-full"
                />
                <p class="text-xs text-gray-500 dark:text-gray-400 mt-2">
                  {{ $t('expertWizard.step3.topPDesc') }}
                </p>
              </div>
            </div>

            <!-- Erweiterte Parameter (optional) -->
            <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <button
                @click="showAdvancedParams = !showAdvancedParams"
                class="flex items-center gap-2 text-sm font-medium text-purple-600 dark:text-purple-400 hover:text-purple-700"
              >
                <ChevronDownIcon class="w-4 h-4 transition-transform" :class="{ 'rotate-180': showAdvancedParams }" />
                {{ $t('expertWizard.step3.advancedParams') }}
              </button>

              <div v-if="showAdvancedParams" class="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                <!-- Top K -->
                <div class="p-3 rounded-lg bg-gray-50 dark:bg-gray-900/50">
                  <div class="flex items-center justify-between mb-2">
                    <label class="text-sm text-gray-700 dark:text-gray-300">{{ $t('expertWizard.step3.topK') }}</label>
                    <span class="text-xs font-mono text-gray-500">{{ wizardData.topK || 40 }}</span>
                  </div>
                  <input
                    v-model.number="wizardData.topK"
                    type="range"
                    min="1"
                    max="100"
                    class="w-full"
                  />
                </div>

                <!-- Repeat Penalty -->
                <div class="p-3 rounded-lg bg-gray-50 dark:bg-gray-900/50">
                  <div class="flex items-center justify-between mb-2">
                    <label class="text-sm text-gray-700 dark:text-gray-300">{{ $t('expertWizard.step3.repeatPenalty') }}</label>
                    <span class="text-xs font-mono text-gray-500">{{ wizardData.repeatPenalty?.toFixed(1) || 1.1 }}</span>
                  </div>
                  <input
                    v-model.number="wizardData.repeatPenalty"
                    type="range"
                    min="1"
                    max="2"
                    step="0.1"
                    class="w-full"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Step 4: Persönlichkeit -->
          <div v-if="currentStep === 4" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step4.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step4.subtitle') }}</p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <!-- Name -->
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ $t('expertWizard.step4.name') }} <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="wizardData.name"
                  type="text"
                  :placeholder="$t('expertWizard.step4.namePlaceholder')"
                  class="w-full px-4 py-2.5 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500"
                />
              </div>

              <!-- Rolle -->
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  {{ $t('expertWizard.step4.role') }} <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="wizardData.role"
                  type="text"
                  :placeholder="$t('expertWizard.step4.rolePlaceholder')"
                  class="w-full px-4 py-2.5 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500"
                />
              </div>
            </div>

            <!-- Beschreibung -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {{ $t('expertWizard.step4.description') }}
              </label>
              <input
                v-model="wizardData.description"
                type="text"
                :placeholder="$t('expertWizard.step4.descriptionPlaceholder')"
                class="w-full px-4 py-2.5 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500"
              />
            </div>

            <!-- Avatar Upload -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('expertWizard.step4.avatar') }}
              </label>
              <div class="flex items-center gap-4">
                <div
                  v-if="wizardData.avatarUrl"
                  class="w-20 h-20 rounded-xl overflow-hidden border-2 border-purple-400 cursor-pointer relative group"
                  @click="triggerAvatarUpload"
                >
                  <img :src="wizardData.avatarUrl" alt="Avatar" class="w-full h-full object-cover" />
                  <div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                    <PhotoIcon class="w-6 h-6 text-white" />
                  </div>
                </div>
                <div
                  v-else
                  class="w-20 h-20 rounded-xl bg-gray-100 dark:bg-gray-700 flex items-center justify-center border-2 border-dashed border-gray-300 dark:border-gray-600 cursor-pointer hover:border-purple-400 transition-colors"
                  @click="triggerAvatarUpload"
                >
                  <PhotoIcon class="w-8 h-8 text-gray-400" />
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  {{ $t('expertWizard.step4.avatarUploadHint') }}<br/>
                  <span class="text-xs">{{ $t('expertWizard.step4.avatarSizeHint') }}</span>
                </div>
                <input ref="avatarInput" type="file" accept="image/*" @change="handleAvatarUpload" class="hidden" />
              </div>
            </div>

            <!-- Stimme (TTS) -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                {{ $t('expertWizard.step4.voice') }}
              </label>
              <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
                <button
                  v-for="voice in availableVoices"
                  :key="voice.id"
                  @click="wizardData.voice = voice.id"
                  :class="[
                    'p-3 rounded-xl border-2 transition-all text-left',
                    wizardData.voice === voice.id
                      ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/30'
                      : 'border-gray-200 dark:border-gray-700 hover:border-purple-300'
                  ]"
                >
                  <div class="flex items-center gap-2">
                    <span class="text-lg">{{ voice.gender === 'female' ? '👩' : '👨' }}</span>
                    <div>
                      <div class="font-medium text-sm text-gray-900 dark:text-white">{{ voice.name }}</div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">{{ voice.description }}</div>
                    </div>
                  </div>
                </button>
              </div>
            </div>

            <!-- Basis-Prompt -->
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ $t('expertWizard.step4.basePrompt') }} <span class="text-red-500">*</span>
                </label>
                <div class="flex gap-2">
                  <button
                    @click="showPromptTemplates = true"
                    class="text-xs px-2 py-1 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 hover:bg-purple-200"
                  >
                    {{ $t('expertWizard.step4.templates') }}
                  </button>
                  <button
                    @click="triggerPromptFileInput"
                    class="text-xs px-2 py-1 rounded bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 hover:bg-gray-200"
                  >
                    {{ $t('expertWizard.step4.fromFile') }}
                  </button>
                  <input ref="promptFileInput" type="file" accept=".txt,.md" @change="loadPromptFromFile" class="hidden" />
                </div>
              </div>
              <textarea
                v-model="wizardData.basePrompt"
                rows="5"
                :placeholder="$t('expertWizard.step4.basePromptPlaceholder')"
                class="w-full px-4 py-3 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 font-mono text-sm resize-none"
              ></textarea>
            </div>

            <!-- Personality Prompt -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                {{ $t('expertWizard.step4.communicationStyle') }}
              </label>
              <textarea
                v-model="wizardData.personalityPrompt"
                rows="2"
                :placeholder="$t('expertWizard.step4.communicationStylePlaceholder')"
                class="w-full px-4 py-3 rounded-xl border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 text-gray-900 dark:text-white focus:ring-2 focus:ring-purple-500 font-mono text-sm resize-none"
              ></textarea>
            </div>
          </div>

          <!-- Step 5: Fachbereiche -->
          <div v-if="currentStep === 5" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step5.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step5.subtitle') }}</p>
            </div>

            <!-- Bestehende Fachbereiche -->
            <div class="space-y-3">
              <div
                v-for="(mode, index) in wizardData.modes"
                :key="index"
                class="p-4 rounded-xl border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50"
              >
                <div class="flex items-start justify-between gap-4">
                  <div class="flex-1 grid grid-cols-1 md:grid-cols-2 gap-3">
                    <div>
                      <label class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('expertWizard.step5.modeName') }} *</label>
                      <input
                        v-model="mode.name"
                        type="text"
                        :placeholder="$t('expertWizard.step5.modeNamePlaceholder')"
                        class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      />
                    </div>
                    <div>
                      <label class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('expertWizard.step5.modeDescription') }}</label>
                      <input
                        v-model="mode.description"
                        type="text"
                        :placeholder="$t('expertWizard.step5.modeDescriptionPlaceholder')"
                        class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      />
                    </div>
                    <div class="md:col-span-2">
                      <label class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('expertWizard.step5.additionalPrompt') }}</label>
                      <textarea
                        v-model="mode.promptAddition"
                        rows="2"
                        :placeholder="$t('expertWizard.step5.additionalPromptPlaceholder')"
                        class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white resize-none"
                      ></textarea>
                    </div>
                    <div>
                      <label class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('expertWizard.step5.keywords') }}</label>
                      <input
                        v-model="mode.keywords"
                        type="text"
                        :placeholder="$t('expertWizard.step5.keywordsPlaceholder')"
                        class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      />
                    </div>
                    <div>
                      <label class="block text-xs text-gray-500 dark:text-gray-400 mb-1">{{ $t('expertWizard.step5.priority') }}</label>
                      <input
                        v-model.number="mode.priority"
                        type="number"
                        min="1"
                        max="10"
                        placeholder="5"
                        class="w-full px-3 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-gray-900 dark:text-white"
                      />
                    </div>
                  </div>
                  <button
                    v-if="wizardData.modes.length > 1"
                    @click="removeMode(index)"
                    class="p-2 text-red-500 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-lg transition-colors"
                    :title="$t('expertWizard.step5.remove')"
                  >
                    <TrashIcon class="w-5 h-5" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Neuen Fachbereich hinzufügen -->
            <button
              @click="addMode"
              class="w-full p-4 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:border-purple-400 hover:text-purple-500 transition-colors flex items-center justify-center gap-2"
            >
              <PlusIcon class="w-5 h-5" />
              {{ $t('expertWizard.step5.addMore') }}
            </button>
          </div>

          <!-- Step 6: Zusammenfassung -->
          <div v-if="currentStep === 6" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">{{ $t('expertWizard.step6.title') }}</h2>
              <p class="text-gray-600 dark:text-gray-400">{{ $t('expertWizard.step6.subtitle') }}</p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <!-- Modell -->
              <div class="p-4 rounded-xl bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
                <div class="flex items-center gap-2 mb-2">
                  <CpuChipIcon class="w-5 h-5 text-blue-500" />
                  <span class="font-bold text-blue-900 dark:text-blue-100">{{ $t('expertWizard.step6.model') }}</span>
                </div>
                <p class="text-sm text-blue-700 dark:text-blue-300">{{ wizardData.selectedModel || $t('expertWizard.step6.notSelected') }}</p>
              </div>

              <!-- Werkzeuge -->
              <div class="p-4 rounded-xl bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800">
                <div class="flex items-center gap-2 mb-2">
                  <WrenchScrewdriverIcon class="w-5 h-5 text-green-500" />
                  <span class="font-bold text-green-900 dark:text-green-100">{{ $t('expertWizard.step6.tools') }}</span>
                </div>
                <div class="text-sm text-green-700 dark:text-green-300 space-y-1">
                  <p>{{ wizardData.autoWebSearch ? $t('expertWizard.step6.webSearchActive') : $t('expertWizard.step6.webSearchOff') }}</p>
                  <p>{{ wizardData.autoFileSearch ? $t('expertWizard.step6.fileSearchActive') : $t('expertWizard.step6.fileSearchOff') }}</p>
                </div>
              </div>

              <!-- Parameter -->
              <div class="p-4 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
                <div class="flex items-center gap-2 mb-2">
                  <Cog6ToothIcon class="w-5 h-5 text-amber-500" />
                  <span class="font-bold text-amber-900 dark:text-amber-100">{{ $t('expertWizard.step6.parameters') }}</span>
                </div>
                <div class="text-sm text-amber-700 dark:text-amber-300 space-y-1">
                  <p>Temperature: {{ wizardData.defaultTemperature?.toFixed(1) }}</p>
                  <p>Context: {{ wizardData.defaultNumCtx?.toLocaleString() }}</p>
                  <p>Max Tokens: {{ wizardData.defaultMaxTokens?.toLocaleString() }}</p>
                </div>
              </div>

              <!-- Persönlichkeit -->
              <div class="p-4 rounded-xl bg-purple-50 dark:bg-purple-900/20 border border-purple-200 dark:border-purple-800">
                <div class="flex items-center gap-2 mb-2">
                  <UserIcon class="w-5 h-5 text-purple-500" />
                  <span class="font-bold text-purple-900 dark:text-purple-100">{{ $t('expertWizard.step6.personality') }}</span>
                </div>
                <div class="text-sm text-purple-700 dark:text-purple-300 space-y-1">
                  <p class="font-medium">{{ wizardData.name || $t('expertWizard.step6.noName') }}, {{ wizardData.role || $t('expertWizard.step6.noRole') }}</p>
                  <p class="text-xs truncate">{{ wizardData.description || $t('expertWizard.step6.noDescription') }}</p>
                </div>
              </div>

              <!-- Fachbereiche -->
              <div class="md:col-span-2 p-4 rounded-xl bg-indigo-50 dark:bg-indigo-900/20 border border-indigo-200 dark:border-indigo-800">
                <div class="flex items-center gap-2 mb-2">
                  <BookOpenIcon class="w-5 h-5 text-indigo-500" />
                  <span class="font-bold text-indigo-900 dark:text-indigo-100">{{ $t('expertWizard.step6.specialties') }} ({{ validModes.length }})</span>
                </div>
                <div class="flex flex-wrap gap-2">
                  <span
                    v-for="mode in validModes"
                    :key="mode.name"
                    class="px-3 py-1 text-sm rounded-full bg-indigo-100 dark:bg-indigo-800 text-indigo-700 dark:text-indigo-300"
                  >
                    {{ mode.name }}
                  </span>
                  <span v-if="validModes.length === 0" class="text-sm text-indigo-500">{{ $t('expertWizard.step6.noSpecialties') }}</span>
                </div>
              </div>
            </div>

            <!-- Basis-Prompt Preview -->
            <div class="p-4 rounded-xl bg-gray-50 dark:bg-gray-900/50 border border-gray-200 dark:border-gray-700">
              <div class="flex items-center gap-2 mb-2">
                <DocumentTextIcon class="w-5 h-5 text-gray-500" />
                <span class="font-bold text-gray-900 dark:text-white">{{ $t('expertWizard.step6.basePromptPreview') }}</span>
              </div>
              <p class="text-sm text-gray-600 dark:text-gray-400 font-mono line-clamp-3">
                {{ wizardData.basePrompt || $t('expertWizard.step6.noBasePrompt') }}
              </p>
            </div>
          </div>
        </div>

        <!-- Footer mit Navigation -->
        <div class="flex-shrink-0 flex items-center justify-between px-6 py-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
          <button
            @click="$emit('close')"
            class="px-4 py-2 text-sm rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
          >
            {{ $t('expertWizard.footer.cancel') }}
          </button>

          <div class="flex gap-3">
            <button
              v-if="currentStep > 1"
              @click="prevStep"
              class="px-4 py-2 text-sm rounded-lg border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors flex items-center gap-2"
            >
              <ArrowLeftIcon class="w-4 h-4" />
              {{ $t('expertWizard.footer.back') }}
            </button>

            <button
              v-if="currentStep < totalSteps"
              @click="nextStep"
              :disabled="!canProceed"
              class="px-6 py-2 text-sm rounded-lg bg-gradient-to-r from-purple-500 to-indigo-500 hover:from-purple-600 hover:to-indigo-600 text-white font-medium shadow-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              {{ $t('expertWizard.footer.next') }}
              <ArrowRightIcon class="w-4 h-4" />
            </button>

            <button
              v-if="currentStep === totalSteps"
              @click="createExpert"
              :disabled="!canCreate || isCreating"
              class="px-6 py-2 text-sm rounded-lg bg-gradient-to-r from-green-500 to-emerald-500 hover:from-green-600 hover:to-emerald-600 text-white font-medium shadow-lg transition-all disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
            >
              <SparklesIcon v-if="!isCreating" class="w-4 h-4" />
              <div v-else class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
              {{ isCreating ? $t('expertWizard.footer.creating') : $t('expertWizard.footer.createExpert') }}
            </button>
          </div>
        </div>
      </div>

      <!-- Prompt Templates Modal -->
      <Transition name="modal">
        <div v-if="showPromptTemplates" class="fixed inset-0 bg-black/50 flex items-center justify-center z-[80]" @click.self="showPromptTemplates = false">
          <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-lg max-h-[70vh] overflow-hidden flex flex-col">
            <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 class="font-bold text-gray-900 dark:text-white">{{ $t('expertWizard.promptTemplates.title') }}</h3>
              <button @click="showPromptTemplates = false" class="p-1 text-gray-400 hover:text-gray-600">
                <XMarkIcon class="w-5 h-5" />
              </button>
            </div>
            <div class="flex-1 overflow-y-auto p-4 space-y-2">
              <button
                v-for="template in promptTemplates"
                :key="template.name"
                @click="applyTemplate(template)"
                class="w-full p-3 text-left rounded-lg border border-gray-200 dark:border-gray-700 hover:border-purple-400 hover:bg-purple-50 dark:hover:bg-purple-900/20 transition-colors"
              >
                <div class="font-medium text-gray-900 dark:text-white">{{ template.name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ template.role }}</div>
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  XMarkIcon,
  ChevronDownIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  PlusIcon,
  TrashIcon,
  CheckCircleIcon,
  CpuChipIcon,
  GlobeAltIcon,
  DocumentMagnifyingGlassIcon,
  CircleStackIcon,
  Cog6ToothIcon,
  UserIcon,
  BookOpenIcon,
  DocumentTextIcon,
  WrenchScrewdriverIcon,
  PhotoIcon,
  SparklesIcon
} from '@heroicons/vue/24/outline'
import axios from 'axios'
import api from '../services/api'
import { useToast } from '../composables/useToast'

const { t } = useI18n()

const props = defineProps({
  show: Boolean,
  editExpert: Object
})

const emit = defineEmits(['close', 'created', 'updated'])
const { success, error } = useToast()

// State
const currentStep = ref(1)
const totalSteps = 6
const isCreating = ref(false)
const isLoadingModels = ref(false)
const showAdvancedParams = ref(false)
const showPromptTemplates = ref(false)
const availableModels = ref([])

const stepTitles = computed(() => [
  t('expertWizard.steps.model'),
  t('expertWizard.steps.tools'),
  t('expertWizard.steps.parameters'),
  t('expertWizard.steps.personality'),
  t('expertWizard.steps.specialties'),
  t('expertWizard.steps.summary')
])
const stepEmojis = ['🤖', '🔧', '⚙️', '👤', '📚', '✅']

// Verfügbare TTS Stimmen
const availableVoices = [
  { id: 'de_DE-thorsten-medium', name: 'Thorsten', description: 'Männlich, neutral', gender: 'male' },
  { id: 'de_DE-thorsten-high', name: 'Thorsten HD', description: 'Männlich, hohe Qualität', gender: 'male' },
  { id: 'de_DE-thorsten_emotional-medium', name: 'Thorsten Emotional', description: 'Männlich, expressiv', gender: 'male' },
  { id: 'de_DE-eva_k-x_low', name: 'Eva K', description: 'Weiblich', gender: 'female' },
  { id: 'de_DE-kerstin-low', name: 'Kerstin', description: 'Weiblich, klar', gender: 'female' },
  { id: 'de_DE-ramona-low', name: 'Ramona', description: 'Weiblich, warm', gender: 'female' },
  { id: 'de_DE-karlsson-low', name: 'Karlsson', description: 'Männlich, tief', gender: 'male' },
  { id: 'de_DE-pavoque-low', name: 'Pavoque', description: 'Männlich, professionell', gender: 'male' },
]

// Wizard Data
const wizardData = ref({
  // Step 1
  selectedModel: null,
  modelInfo: {},

  // Step 2
  autoWebSearch: false,
  searchDomains: '',
  maxSearchResults: 5,
  autoFileSearch: false,
  documentDirectory: '',

  // Step 3
  defaultNumCtx: 65536,  // 64K Default
  defaultMaxTokens: 4096,
  defaultTemperature: 0.7,
  defaultTopP: 0.9,
  topK: 40,
  repeatPenalty: 1.1,

  // Step 4
  name: '',
  role: '',
  description: '',
  avatarUrl: null,
  voice: 'de_DE-thorsten-medium', // Standard TTS Stimme
  basePrompt: '',
  personalityPrompt: '',

  // Step 5
  modes: [{
    name: '',
    description: '',
    promptAddition: '',
    keywords: '',
    priority: 5,
    temperature: null,
    topP: null,
    maxTokens: null
  }]
})

// File inputs
const avatarInput = ref(null)
const promptFileInput = ref(null)

// Prompt Templates
const promptTemplates = [
  {
    name: 'Rechtsanwalt',
    role: 'Rechtsanwalt',
    prompt: 'Du bist ein erfahrener deutscher Rechtsanwalt. Du berätst Mandanten zu rechtlichen Fragen und erklärst komplexe juristische Sachverhalte verständlich. Du weist immer darauf hin, dass deine Antworten keine Rechtsberatung ersetzen und bei konkreten Rechtsfragen ein Anwalt konsultiert werden sollte.',
    modes: ['Zivilrecht', 'Strafrecht', 'Verwaltungsrecht', 'Arbeitsrecht']
  },
  {
    name: 'Steuerberater',
    role: 'Steuerberater',
    prompt: 'Du bist ein erfahrener deutscher Steuerberater. Du hilfst bei Fragen zu Steuererklärungen, Steuerrecht und finanzieller Planung. Du erklärst steuerliche Konzepte verständlich und weist auf wichtige Fristen und Regelungen hin.',
    modes: ['Einkommensteuer', 'Umsatzsteuer', 'Gewerbesteuer', 'Erbschaftsteuer']
  },
  {
    name: 'Software-Entwickler',
    role: 'Senior Software-Entwickler',
    prompt: 'Du bist ein erfahrener Senior Software-Entwickler mit Expertise in mehreren Programmiersprachen und Frameworks. Du hilfst bei Code-Reviews, Architekturentscheidungen und Debugging. Du erklärst technische Konzepte klar und gibst praktische Beispiele.',
    modes: ['Code-Review', 'Debugging', 'Architektur', 'Best Practices']
  },
  {
    name: 'Arzt',
    role: 'Allgemeinmediziner',
    prompt: 'Du bist ein erfahrener Allgemeinmediziner. Du gibst allgemeine Gesundheitsinformationen und erklärst medizinische Zusammenhänge verständlich. Du weist immer darauf hin, dass deine Informationen keinen Arztbesuch ersetzen und bei Beschwerden ein Arzt aufgesucht werden sollte.',
    modes: ['Prävention', 'Symptome', 'Medikamente', 'Ernährung']
  },
  {
    name: 'Marketing-Experte',
    role: 'Marketing-Stratege',
    prompt: 'Du bist ein erfahrener Marketing-Stratege mit Expertise in digitalem Marketing, Branding und Kundenkommunikation. Du hilfst bei der Entwicklung von Marketing-Strategien und gibst praktische Tipps für verschiedene Kanäle.',
    modes: ['Social Media', 'Content Marketing', 'SEO', 'Branding']
  }
]

// Computed
const validModes = computed(() => {
  return wizardData.value.modes.filter(m => m.name?.trim())
})

const canProceed = computed(() => {
  switch (currentStep.value) {
    case 1:
      return !!wizardData.value.selectedModel
    case 2:
      return true // Werkzeuge sind optional
    case 3:
      return true // Parameter haben Defaults
    case 4:
      return wizardData.value.name?.trim() &&
             wizardData.value.role?.trim() &&
             wizardData.value.basePrompt?.trim()
    case 5:
      return validModes.value.length >= 1
    default:
      return true
  }
})

const canCreate = computed(() => {
  return wizardData.value.selectedModel &&
         wizardData.value.name?.trim() &&
         wizardData.value.role?.trim() &&
         wizardData.value.basePrompt?.trim() &&
         validModes.value.length >= 1
})

// Watchers
watch(() => props.show, async (show) => {
  if (show) {
    currentStep.value = 1
    if (props.editExpert) {
      loadExpertData(props.editExpert)
    } else {
      resetWizard()
    }
    await loadModels()
  }
})

// Methods
async function loadModels() {
  isLoadingModels.value = true
  try {
    const response = await api.getAvailableModels()
    // API kann verschiedene Formate zurückgeben - normalisieren
    let models = []

    if (response.models && Array.isArray(response.models)) {
      // Format: { models: ["model1.gguf", "model2.gguf"] }
      models = response.models.map(m => {
        if (typeof m === 'string') {
          // String zu Objekt konvertieren
          const displayName = m.replace('.gguf', '').replace(/-/g, ' ')
          return {
            name: m,
            displayName: displayName,
            description: '',
            size: null,
            contextLength: null,
            category: m.includes('coder') ? 'Code' : m.includes('llava') ? 'Vision' : 'Chat'
          }
        }
        return m // Bereits ein Objekt
      })
    } else if (Array.isArray(response)) {
      // Format: [{ name: "model1", ... }] oder ["model1", "model2"]
      models = response.map(m => {
        if (typeof m === 'string') {
          const displayName = m.replace('.gguf', '').replace(/-/g, ' ')
          return {
            name: m,
            displayName: displayName,
            description: '',
            size: null,
            contextLength: null,
            category: m.includes('coder') ? 'Code' : m.includes('llava') ? 'Vision' : 'Chat'
          }
        }
        return m
      })
    }

    // Filter Vision-Modelle raus (mmproj-*), die sind nur Hilfsmodelle
    availableModels.value = models.filter(m => !m.name.startsWith('mmproj-'))
  } catch (err) {
    console.error('Failed to load models:', err)
    error(t('expertWizard.toast.loadModelsError'))
  } finally {
    isLoadingModels.value = false
  }
}

function loadExpertData(expert) {
  wizardData.value = {
    selectedModel: expert.baseModel,
    modelInfo: {},
    autoWebSearch: expert.autoWebSearch || false,
    searchDomains: expert.searchDomains || '',
    maxSearchResults: expert.maxSearchResults || 5,
    autoFileSearch: expert.autoFileSearch || false,
    documentDirectory: expert.documentDirectory || '',
    defaultNumCtx: expert.defaultNumCtx || 65536,  // 64K Default
    defaultMaxTokens: expert.defaultMaxTokens || 4096,
    defaultTemperature: expert.defaultTemperature || 0.7,
    defaultTopP: expert.defaultTopP || 0.9,
    topK: expert.topK || 40,
    repeatPenalty: expert.repeatPenalty || 1.1,
    name: expert.name || '',
    role: expert.role || '',
    description: expert.description || '',
    avatarUrl: expert.avatarUrl || null,
    voice: expert.voice || 'de_DE-thorsten-medium',
    basePrompt: expert.basePrompt || '',
    personalityPrompt: expert.personalityPrompt || '',
    modes: expert.modes?.length ? expert.modes.map(m => ({
      name: m.name || '',
      description: m.description || '',
      promptAddition: m.promptAddition || '',
      keywords: m.keywords || '',
      priority: m.priority || 5,
      temperature: m.temperature,
      topP: m.topP,
      maxTokens: m.maxTokens
    })) : [{ name: '', description: '', promptAddition: '', keywords: '', priority: 5 }]
  }
}

function resetWizard() {
  wizardData.value = {
    selectedModel: null,
    modelInfo: {},
    autoWebSearch: false,
    searchDomains: '',
    maxSearchResults: 5,
    autoFileSearch: false,
    documentDirectory: '',
    defaultNumCtx: 65536,  // 64K Default
    defaultMaxTokens: 4096,
    defaultTemperature: 0.7,
    defaultTopP: 0.9,
    topK: 40,
    repeatPenalty: 1.1,
    name: '',
    role: '',
    description: '',
    avatarUrl: null,
    voice: 'de_DE-thorsten-medium',
    basePrompt: '',
    personalityPrompt: '',
    modes: [{ name: '', description: '', promptAddition: '', keywords: '', priority: 5 }]
  }
  showAdvancedParams.value = false
}

function selectModel(model) {
  wizardData.value.selectedModel = model.name
  wizardData.value.modelInfo = model

  // Auto-set context size from model
  if (model.contextLength) {
    wizardData.value.defaultNumCtx = Math.min(model.contextLength, 131072)
  }
}

function nextStep() {
  if (canProceed.value && currentStep.value < totalSteps) {
    currentStep.value++
  }
}

function prevStep() {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

function goToStep(step) {
  if (canGoToStep(step)) {
    currentStep.value = step
  }
}

function canGoToStep(step) {
  // Can only go to completed steps or current step
  return step <= currentStep.value
}

function addMode() {
  wizardData.value.modes.push({
    name: '',
    description: '',
    promptAddition: '',
    keywords: '',
    priority: 5,
    temperature: null,
    topP: null,
    maxTokens: null
  })
}

function removeMode(index) {
  if (wizardData.value.modes.length > 1) {
    wizardData.value.modes.splice(index, 1)
  }
}

function triggerAvatarUpload() {
  avatarInput.value?.click()
}

async function handleAvatarUpload(event) {
  const file = event.target.files?.[0]
  if (!file) return

  if (file.size > 5 * 1024 * 1024) {
    error(t('expertWizard.toast.fileTooLarge'))
    event.target.value = ''
    return
  }

  try {
    const formData = new FormData()
    formData.append('file', file)

    const response = await axios.post('/api/experts/avatar/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    if (response.data.success) {
      wizardData.value.avatarUrl = response.data.avatarUrl
      success(t('expertWizard.toast.avatarUploaded'))
    }
  } catch (err) {
    console.error('Avatar upload failed:', err)
    error(t('expertWizard.toast.uploadFailed'))
  } finally {
    event.target.value = ''
  }
}

function triggerPromptFileInput() {
  promptFileInput.value?.click()
}

async function loadPromptFromFile(event) {
  const file = event.target.files?.[0]
  if (!file) return

  try {
    const text = await file.text()
    wizardData.value.basePrompt = text.trim()
    success(t('expertWizard.toast.promptLoaded', { filename: file.name }))
  } catch (err) {
    error(t('expertWizard.toast.fileReadError'))
  }
  event.target.value = ''
}

function applyTemplate(template) {
  wizardData.value.name = template.name
  wizardData.value.role = template.role
  wizardData.value.basePrompt = template.prompt

  // Fachbereiche aus Template
  if (template.modes?.length) {
    wizardData.value.modes = template.modes.map(name => ({
      name,
      description: '',
      promptAddition: '',
      keywords: '',
      priority: 5
    }))
  }

  showPromptTemplates.value = false
  success(t('expertWizard.toast.templateApplied', { name: template.name }))
}

async function createExpert() {
  if (!canCreate.value) return

  isCreating.value = true

  try {
    const expertData = {
      name: wizardData.value.name.trim(),
      role: wizardData.value.role.trim(),
      description: wizardData.value.description?.trim() || null,
      avatarUrl: wizardData.value.avatarUrl,
      voice: wizardData.value.voice || 'de_DE-thorsten-medium',
      baseModel: wizardData.value.selectedModel,
      basePrompt: wizardData.value.basePrompt.trim(),
      personalityPrompt: wizardData.value.personalityPrompt?.trim() || null,
      defaultTemperature: wizardData.value.defaultTemperature,
      defaultTopP: wizardData.value.defaultTopP,
      defaultNumCtx: wizardData.value.defaultNumCtx,
      defaultMaxTokens: wizardData.value.defaultMaxTokens,
      autoWebSearch: wizardData.value.autoWebSearch,
      searchDomains: wizardData.value.searchDomains || null,
      maxSearchResults: wizardData.value.maxSearchResults,
      autoFileSearch: wizardData.value.autoFileSearch,
      documentDirectory: wizardData.value.documentDirectory || null
    }

    // Create expert
    const expert = await api.createExpert(expertData)

    // Create modes
    for (const mode of validModes.value) {
      await api.createExpertMode(expert.id, {
        name: mode.name.trim(),
        description: mode.description?.trim() || null,
        promptAddition: mode.promptAddition?.trim() || null,
        keywords: mode.keywords?.trim() || null,
        priority: mode.priority || 5,
        temperature: mode.temperature,
        topP: mode.topP,
        maxTokens: mode.maxTokens
      })
    }

    success(t('expertWizard.toast.expertCreated', { name: expertData.name }))
    emit('created', expert)
    emit('close')
  } catch (err) {
    console.error('Failed to create expert:', err)
    error(err.response?.data?.error || t('expertWizard.toast.createError'))
  } finally {
    isCreating.value = false
  }
}

function formatSize(bytes) {
  if (!bytes) return ''
  if (bytes < 1024 * 1024 * 1024) {
    return (bytes / (1024 * 1024)).toFixed(0) + ' MB'
  }
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
}

function formatContextLength(length) {
  if (!length) return ''
  if (length >= 1000) {
    return (length / 1000) + 'K'
  }
  return length.toString()
}
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

/* Range Slider Styles für bessere Sichtbarkeit im Dark Mode */
input[type="range"] {
  -webkit-appearance: none;
  appearance: none;
  background: transparent;
  cursor: pointer;
  height: 24px;
}

input[type="range"]::-webkit-slider-runnable-track {
  background: linear-gradient(to right, #a855f7, #6366f1);
  height: 8px;
  border-radius: 9999px;
}

input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  margin-top: -6px;
  background-color: #fff;
  border: 3px solid #a855f7;
  height: 20px;
  width: 20px;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

input[type="range"]::-moz-range-track {
  background: linear-gradient(to right, #a855f7, #6366f1);
  height: 8px;
  border-radius: 9999px;
}

input[type="range"]::-moz-range-thumb {
  background-color: #fff;
  border: 3px solid #a855f7;
  height: 16px;
  width: 16px;
  border-radius: 50%;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
}

/* Checkbox Styles für Dark Mode */
input[type="checkbox"] {
  accent-color: #a855f7;
}

/* Text-Inputs im Dark Mode besser sichtbar */
:deep(.dark) input[type="text"],
:deep(.dark) textarea {
  color: #fff !important;
}
</style>
