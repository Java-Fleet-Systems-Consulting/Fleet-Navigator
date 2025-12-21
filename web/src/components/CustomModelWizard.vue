<template>
  <Transition name="modal">
    <div v-if="show" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-[70] p-4">
      <div
        class="
          bg-white dark:bg-gray-800
          rounded-2xl shadow-2xl
          w-full max-w-3xl
          border border-gray-200 dark:border-gray-700
          flex flex-col
          overflow-hidden
        "
        style="height: 80vh; max-height: 80vh;"
      >
        <!-- Header mit Progress -->
        <div class="flex-shrink-0 bg-gradient-to-r from-indigo-500/10 to-purple-500/10 dark:from-indigo-500/20 dark:to-purple-500/20 border-b border-gray-200 dark:border-gray-700">
          <!-- Progress Bar -->
          <div class="px-6 pt-4">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium text-gray-600 dark:text-gray-400">
                Schritt {{ currentStep }} von {{ totalSteps }}
              </span>
              <span class="text-sm text-gray-500 dark:text-gray-500">
                {{ stepTitles[currentStep - 1] }}
              </span>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div
                class="bg-gradient-to-r from-indigo-500 to-purple-500 h-2 rounded-full transition-all duration-300"
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
                  ? 'bg-indigo-500 text-white shadow-lg'
                  : step < currentStep
                    ? 'bg-indigo-100 dark:bg-indigo-900/30 text-indigo-600 dark:text-indigo-400 cursor-pointer hover:bg-indigo-200 dark:hover:bg-indigo-900/50'
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

          <!-- Provider nicht verfügbar Warnung -->
          <div v-if="!providerAvailable && !isCheckingProvider" class="flex flex-col items-center justify-center h-full text-center">
            <div class="p-4 rounded-full bg-amber-100 dark:bg-amber-900/30 mb-4">
              <ExclamationTriangleIcon class="w-12 h-12 text-amber-500" />
            </div>
            <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Kein Provider verfügbar</h2>
            <p class="text-gray-600 dark:text-gray-400 mb-4 max-w-md">
              Bitte stellen Sie sicher, dass llama-server gestartet ist.
            </p>
            <div class="flex gap-3">
              <button
                @click="checkProviderStatus"
                class="px-4 py-2 bg-indigo-500 hover:bg-indigo-600 text-white rounded-lg transition-colors"
              >
                Erneut prüfen
              </button>
              <button
                @click="$emit('close')"
                class="px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-lg transition-colors"
              >
                Schließen
              </button>
            </div>
          </div>

          <!-- Loading -->
          <div v-else-if="isCheckingProvider" class="flex flex-col items-center justify-center h-full">
            <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500 mb-4"></div>
            <p class="text-gray-600 dark:text-gray-400">Prüfe Provider-Status...</p>
          </div>

          <!-- Step 1: Basis-Modell wählen -->
          <div v-else-if="currentStep === 1" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Wähle ein Basis-Modell</h2>
              <p class="text-gray-600 dark:text-gray-400">Das Basis-Modell bestimmt die Fähigkeiten deines Custom Models.</p>
              <!-- Provider-Info Badge -->
              <div class="mt-2 inline-flex items-center gap-2 px-3 py-1 rounded-full text-sm bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400">
                <span>⚡ llama-server</span>
                <span class="text-xs opacity-75">(GGUF Konfiguration)</span>
              </div>
            </div>

            <div v-if="isLoadingModels" class="flex justify-center py-12">
              <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
            </div>

            <div v-else-if="availableModels.length === 0" class="text-center py-12">
              <CpuChipIcon class="w-16 h-16 text-gray-300 dark:text-gray-600 mx-auto mb-4" />
              <p class="text-gray-500 dark:text-gray-400">Keine GGUF-Modelle verfügbar.</p>
              <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">Bitte laden Sie zuerst GGUF-Modelle herunter.</p>
            </div>

            <div v-else class="grid grid-cols-1 gap-4">
              <button
                v-for="model in availableModels"
                :key="model.name"
                @click="selectModel(model)"
                class="
                  p-4 rounded-xl border-2 text-left
                  transition-all duration-200
                  hover:shadow-lg
                  bg-gray-50 dark:bg-gray-700/80
                "
                :class="[
                  wizardData.baseModel === model.name
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/40 shadow-lg shadow-indigo-500/20'
                    : 'border-gray-300 dark:border-gray-500 hover:border-indigo-400 dark:hover:border-indigo-400'
                ]"
              >
                <div class="flex items-start gap-3">
                  <!-- Model Icon -->
                  <div class="p-3 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-500 flex-shrink-0 text-2xl">
                    {{ getModelInfo(model.name).emoji }}
                  </div>

                  <!-- Model Info -->
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                      <h3 class="font-bold text-gray-900 dark:text-gray-100">
                        {{ getModelInfo(model.name).displayName }}
                      </h3>
                      <!-- Params Badge -->
                      <span class="text-xs px-2 py-0.5 rounded-full bg-purple-100 dark:bg-purple-900/40 text-purple-700 dark:text-purple-300 font-medium">
                        {{ getModelInfo(model.name).params }}
                      </span>
                      <!-- Vision Badge -->
                      <span v-if="getModelInfo(model.name).isVision" class="text-xs px-2 py-0.5 rounded-full bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300">
                        👁️ Vision
                      </span>
                    </div>

                    <!-- Description -->
                    <p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
                      {{ getModelInfo(model.name).description }}
                    </p>

                    <!-- Capabilities -->
                    <div class="flex flex-wrap gap-1.5 mt-2">
                      <span
                        v-for="cap in getModelInfo(model.name).capabilities.slice(0, 4)"
                        :key="cap"
                        class="text-xs px-2 py-0.5 rounded-full bg-gray-100 dark:bg-gray-600 text-gray-600 dark:text-gray-300"
                      >
                        {{ cap }}
                      </span>
                    </div>

                    <!-- Footer: RAM + Link -->
                    <div class="flex items-center justify-between mt-3 pt-2 border-t border-gray-200 dark:border-gray-600">
                      <span class="text-xs text-gray-500 dark:text-gray-400">
                        💾 Min. {{ getModelInfo(model.name).minRam }} RAM
                      </span>
                      <a
                        v-if="getModelInfo(model.name).url"
                        :href="getModelInfo(model.name).url"
                        target="_blank"
                        rel="noopener noreferrer"
                        @click.stop
                        class="text-xs text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 flex items-center gap-1"
                      >
                        <LinkIcon class="w-3 h-3" />
                        Mehr Infos
                      </a>
                    </div>
                  </div>

                  <!-- Selection Indicator -->
                  <div v-if="wizardData.baseModel === model.name" class="flex-shrink-0">
                    <CheckCircleIcon class="w-7 h-7 text-indigo-500" />
                  </div>
                </div>
              </button>
            </div>
          </div>

          <!-- Step 2: Name und Eigenschaften -->
          <div v-else-if="currentStep === 2" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Name und Eigenschaften</h2>
              <p class="text-gray-600 dark:text-gray-400">Definiere die Identität und Persönlichkeit deines Modells.</p>
            </div>

            <!-- Model Name -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Modellname *
              </label>
              <input
                v-model="wizardData.name"
                type="text"
                placeholder="z.B. nova, assistant, coder"
                class="
                  w-full px-4 py-3
                  bg-white dark:bg-gray-900
                  border border-gray-300 dark:border-gray-600
                  text-gray-900 dark:text-gray-100
                  rounded-xl
                  focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent
                "
              />
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Nur Kleinbuchstaben, Zahlen und Bindestriche. Tag ":latest" wird automatisch hinzugefügt.
              </p>
            </div>

            <!-- Description -->
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Beschreibung (optional)
              </label>
              <input
                v-model="wizardData.description"
                type="text"
                placeholder="z.B. Hilfreicher deutscher Assistent"
                class="
                  w-full px-4 py-3
                  bg-white dark:bg-gray-900
                  border border-gray-300 dark:border-gray-600
                  text-gray-900 dark:text-gray-100
                  rounded-xl
                  focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent
                "
              />
            </div>

            <!-- System Prompt -->
            <div>
              <div class="flex items-center justify-between mb-2">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  System Prompt (Persönlichkeit)
                </label>
                <button
                  @click="triggerPromptFileInput"
                  type="button"
                  class="
                    flex items-center gap-2 px-3 py-1.5 text-sm
                    text-indigo-600 dark:text-indigo-400
                    hover:bg-indigo-50 dark:hover:bg-indigo-900/20
                    rounded-lg transition-all duration-200
                    border border-indigo-300 dark:border-indigo-600
                  "
                >
                  <ArrowUpTrayIcon class="w-4 h-4" />
                  <span>Aus Datei</span>
                </button>
                <input
                  ref="promptFileInput"
                  type="file"
                  @change="handlePromptFileSelect"
                  accept=".txt,.md"
                  class="hidden"
                />
              </div>

              <!-- Prompt-Vorlagen -->
              <div class="flex flex-wrap gap-2 mb-3">
                <button
                  v-for="template in promptTemplates"
                  :key="template.name"
                  @click="applyPromptTemplate(template)"
                  class="
                    px-3 py-1.5 text-xs font-medium rounded-lg
                    bg-indigo-100 dark:bg-indigo-900/50
                    hover:bg-indigo-200 dark:hover:bg-indigo-800/60
                    text-indigo-700 dark:text-indigo-200
                    hover:text-indigo-800 dark:hover:text-indigo-100
                    border border-indigo-200 dark:border-indigo-700
                    transition-colors
                  "
                >
                  {{ template.emoji }} {{ template.name }}
                </button>
              </div>

              <textarea
                v-model="wizardData.systemPrompt"
                rows="6"
                placeholder="z.B. Du bist Nova, ein hilfreicher deutscher KI-Assistent. Du antwortest immer höflich und präzise..."
                class="
                  w-full px-4 py-3
                  bg-white dark:bg-gray-900
                  border border-gray-300 dark:border-gray-600
                  text-gray-900 dark:text-gray-100
                  rounded-xl
                  focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent
                  resize-none
                "
              ></textarea>
            </div>

            <!-- Erweiterte Parameter (collapsed) -->
            <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
              <button
                @click="showAdvanced = !showAdvanced"
                class="flex items-center gap-2 text-sm font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 transition-colors"
              >
                <ChevronDownIcon class="w-5 h-5 transition-transform duration-200" :class="{ 'rotate-180': showAdvanced }" />
                Erweiterte Parameter
              </button>

              <Transition name="fade">
                <div v-if="showAdvanced" class="mt-4 grid grid-cols-2 gap-4">
                  <!-- Temperature -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Temperature: {{ wizardData.temperature }}
                    </label>
                    <input
                      v-model.number="wizardData.temperature"
                      type="range"
                      min="0"
                      max="2"
                      step="0.1"
                      class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg cursor-pointer accent-indigo-500"
                    />
                  </div>

                  <!-- Top P -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Top P: {{ wizardData.topP }}
                    </label>
                    <input
                      v-model.number="wizardData.topP"
                      type="range"
                      min="0"
                      max="1"
                      step="0.05"
                      class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg cursor-pointer accent-indigo-500"
                    />
                  </div>

                  <!-- Context Length -->
                  <div class="col-span-2">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                      Context Length: {{ wizardData.numCtx.toLocaleString() }} Tokens
                    </label>
                    <input
                      v-model.number="wizardData.numCtx"
                      type="range"
                      min="2048"
                      max="131072"
                      step="2048"
                      class="w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-lg cursor-pointer accent-indigo-500"
                    />
                    <p class="text-xs text-gray-500 mt-1">Mehr Context = mehr VRAM benötigt</p>
                  </div>
                </div>
              </Transition>
            </div>
          </div>

          <!-- Step 3: Zusammenfassung -->
          <div v-else-if="currentStep === 3" class="space-y-6">
            <div class="text-center mb-6">
              <h2 class="text-2xl font-bold text-gray-900 dark:text-white mb-2">Zusammenfassung</h2>
              <p class="text-gray-600 dark:text-gray-400">Prüfe deine Einstellungen und erstelle das Modell.</p>
            </div>

            <!-- Summary Card -->
            <div v-if="!isCreating" class="bg-gradient-to-br from-indigo-50 to-purple-50 dark:from-indigo-900/20 dark:to-purple-900/20 rounded-2xl p-6 border border-indigo-200 dark:border-indigo-800">
              <div class="flex items-start gap-4 mb-6">
                <div class="p-3 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-500 shadow-lg">
                  <SparklesIcon class="w-8 h-8 text-white" />
                </div>
                <div>
                  <h3 class="text-xl font-bold text-gray-900 dark:text-white">
                    {{ wizardData.name || 'Unbenannt' }}:latest
                  </h3>
                  <p class="text-gray-600 dark:text-gray-400">{{ wizardData.description || 'Keine Beschreibung' }}</p>
                </div>
              </div>

              <div class="space-y-3">
                <div class="flex justify-between py-2 border-b border-indigo-200 dark:border-indigo-700">
                  <span class="text-gray-600 dark:text-gray-400">Basis-Modell</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ wizardData.baseModel }}</span>
                </div>
                <div class="flex justify-between py-2 border-b border-indigo-200 dark:border-indigo-700">
                  <span class="text-gray-600 dark:text-gray-400">Temperature</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ wizardData.temperature }}</span>
                </div>
                <div class="flex justify-between py-2 border-b border-indigo-200 dark:border-indigo-700">
                  <span class="text-gray-600 dark:text-gray-400">Context Length</span>
                  <span class="font-medium text-gray-900 dark:text-white">{{ wizardData.numCtx.toLocaleString() }} Tokens</span>
                </div>
                <div class="py-2">
                  <span class="text-gray-600 dark:text-gray-400">System Prompt</span>
                  <p class="mt-1 text-sm text-gray-900 dark:text-white bg-white/50 dark:bg-gray-800/50 rounded-lg p-3 max-h-32 overflow-y-auto">
                    {{ wizardData.systemPrompt || 'Kein System Prompt definiert' }}
                  </p>
                </div>
              </div>
            </div>

            <!-- Creating Progress -->
            <div v-else class="flex flex-col items-center justify-center py-12">
              <div class="animate-spin rounded-full h-16 w-16 border-b-2 border-indigo-500 mb-6"></div>
              <p class="text-lg font-medium text-gray-900 dark:text-white mb-2">
                {{ creationProgress }}
              </p>
              <div class="w-full max-w-md bg-gray-200 dark:bg-gray-700 rounded-full h-2 mt-4">
                <div
                  class="bg-indigo-500 h-2 rounded-full transition-all duration-300"
                  :style="{ width: creationProgressPercent + '%' }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="flex-shrink-0 p-6 border-t border-gray-200 dark:border-gray-700 bg-gray-50/50 dark:bg-gray-900/50">
          <div class="flex justify-between items-center">
            <!-- Back Button -->
            <button
              v-if="currentStep > 1 && providerAvailable"
              @click="previousStep"
              :disabled="isCreating"
              class="
                px-6 py-2.5 rounded-xl
                text-gray-700 dark:text-gray-300
                bg-white dark:bg-gray-800
                border border-gray-300 dark:border-gray-600
                hover:bg-gray-50 dark:hover:bg-gray-700
                font-medium
                transition-all duration-200
                disabled:opacity-50 disabled:cursor-not-allowed
              "
            >
              Zurück
            </button>
            <div v-else></div>

            <div class="flex gap-3">
              <!-- Cancel Button -->
              <button
                @click="$emit('close')"
                :disabled="isCreating"
                class="
                  px-6 py-2.5 rounded-xl
                  text-gray-700 dark:text-gray-300
                  bg-white dark:bg-gray-800
                  border border-gray-300 dark:border-gray-600
                  hover:bg-gray-50 dark:hover:bg-gray-700
                  font-medium
                  transition-all duration-200
                  disabled:opacity-50 disabled:cursor-not-allowed
                "
              >
                Abbrechen
              </button>

              <!-- Next/Create Button -->
              <button
                v-if="currentStep < totalSteps"
                @click="nextStep"
                :disabled="!canProceed"
                class="
                  px-6 py-2.5 rounded-xl
                  bg-gradient-to-r from-indigo-500 to-purple-500
                  hover:from-indigo-600 hover:to-purple-600
                  text-white font-medium
                  shadow-sm hover:shadow-md
                  transition-all duration-200
                  disabled:opacity-50 disabled:cursor-not-allowed
                  flex items-center gap-2
                "
              >
                <span>Weiter</span>
                <ArrowRightIcon class="w-5 h-5" />
              </button>
              <button
                v-else
                @click="createModel"
                :disabled="!canCreate || isCreating"
                class="
                  px-6 py-2.5 rounded-xl
                  bg-gradient-to-r from-indigo-500 to-purple-500
                  hover:from-indigo-600 hover:to-purple-600
                  text-white font-medium
                  shadow-sm hover:shadow-md
                  transition-all duration-200
                  disabled:opacity-50 disabled:cursor-not-allowed
                  flex items-center gap-2
                "
              >
                <SparklesIcon class="w-5 h-5" />
                <span>Modell erstellen</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import {
  XMarkIcon,
  SparklesIcon,
  ChevronDownIcon,
  CpuChipIcon,
  CheckCircleIcon,
  ArrowRightIcon,
  ArrowUpTrayIcon,
  ExclamationTriangleIcon,
  LinkIcon
} from '@heroicons/vue/24/outline'
import api from '../services/api'
import { useToast } from '../composables/useToast'

const props = defineProps({
  show: Boolean
})

const emit = defineEmits(['close', 'created'])

const { success, error: errorToast } = useToast()

// Step configuration
const totalSteps = 3
const currentStep = ref(1)
const stepTitles = ['Basis-Modell', 'Eigenschaften', 'Erstellen']
const stepEmojis = ['1️⃣', '2️⃣', '3️⃣']

// Wizard data
const wizardData = ref({
  baseModel: '',
  name: '',
  description: '',
  systemPrompt: '',
  temperature: 0.8,
  topP: 0.9,
  topK: 40,
  repeatPenalty: 1.1,
  numPredict: 2048,
  numCtx: 8192
})

// UI state
const showAdvanced = ref(false)
const isCreating = ref(false)
const creationProgress = ref('')
const creationProgressPercent = ref(0)
const availableModels = ref([])
const isLoadingModels = ref(false)
const providerAvailable = ref(false)
const isCheckingProvider = ref(true)
const promptFileInput = ref(null)

// Prompt templates
const promptTemplates = [
  {
    name: 'Assistent',
    emoji: '🤖',
    prompt: 'Du bist ein hilfreicher, freundlicher KI-Assistent. Du antwortest auf Deutsch, präzise und gut strukturiert. Bei Unsicherheiten fragst du nach.'
  },
  {
    name: 'Coder',
    emoji: '💻',
    prompt: 'Du bist ein erfahrener Software-Entwickler. Du schreibst sauberen, gut dokumentierten Code und erklärst deine Lösungen. Du verwendest Best Practices und moderne Patterns.'
  },
  {
    name: 'Kreativ',
    emoji: '🎨',
    prompt: 'Du bist ein kreativer Schreibassistent. Du hilfst beim Verfassen von Texten, Geschichten und kreativen Inhalten. Du bist inspirierend und ideenreich.'
  },
  {
    name: 'Analyst',
    emoji: '📊',
    prompt: 'Du bist ein analytischer Datenexperte. Du analysierst Informationen gründlich, erkennst Muster und präsentierst Erkenntnisse klar strukturiert.'
  }
]

// Model information database
const MODEL_INFO = {
  // Llama Models
  'Llama-3.2-1B-Instruct-Q4_K_M.gguf': {
    displayName: 'Llama 3.2 1B',
    description: 'Kompaktes Modell für einfache Aufgaben. Schnell und ressourcenschonend.',
    capabilities: ['Chat', 'Schnelle Antworten', 'Geringe RAM-Nutzung'],
    params: '1B',
    minRam: '2 GB',
    url: 'https://huggingface.co/meta-llama/Llama-3.2-1B-Instruct',
    emoji: '🦙'
  },
  'Llama-3.2-3B-Instruct-Q4_K_M.gguf': {
    displayName: 'Llama 3.2 3B',
    description: 'Ausgewogenes Modell mit gutem Preis-Leistungs-Verhältnis.',
    capabilities: ['Chat', 'Zusammenfassungen', 'Übersetzungen'],
    params: '3B',
    minRam: '4 GB',
    url: 'https://huggingface.co/meta-llama/Llama-3.2-3B-Instruct',
    emoji: '🦙'
  },
  // Mistral Models
  'Mistral-7B-Instruct-v0.3.IQ4_XS.gguf': {
    displayName: 'Mistral 7B v0.3',
    description: 'Starkes 7B-Modell von Mistral AI. Sehr gut für Deutsch.',
    capabilities: ['Chat', 'Code', 'Analyse', 'Kreatives Schreiben'],
    params: '7B',
    minRam: '6 GB',
    url: 'https://huggingface.co/mistralai/Mistral-7B-Instruct-v0.3',
    emoji: '🌪️'
  },
  'Mistral-Nemo-Instruct-2407.Q4_K_M.gguf': {
    displayName: 'Mistral Nemo 12B',
    description: 'Fortgeschrittenes Mistral-Modell mit 12B Parametern. Sehr hohe Qualität.',
    capabilities: ['Chat', 'Code', 'Komplexe Analyse', 'Mehrsprachig'],
    params: '12B',
    minRam: '10 GB',
    url: 'https://huggingface.co/mistralai/Mistral-Nemo-Instruct-2407',
    emoji: '🌪️'
  },
  // Qwen Models
  'Qwen2.5-7B-Instruct-Q5_K_M.gguf': {
    displayName: 'Qwen 2.5 7B',
    description: 'Leistungsstarkes Modell von Alibaba. Exzellent für Chat und Code.',
    capabilities: ['Chat', 'Code', 'Mathematik', 'Mehrsprachig'],
    params: '7B',
    minRam: '8 GB',
    url: 'https://huggingface.co/Qwen/Qwen2.5-7B-Instruct',
    emoji: '🔮'
  },
  'qwen2.5-3b-instruct-q4_k_m.gguf': {
    displayName: 'Qwen 2.5 3B',
    description: 'Kompakte Qwen-Version. Gute Balance zwischen Geschwindigkeit und Qualität.',
    capabilities: ['Chat', 'Leichte Code-Aufgaben', 'Schnelle Antworten'],
    params: '3B',
    minRam: '4 GB',
    url: 'https://huggingface.co/Qwen/Qwen2.5-3B-Instruct',
    emoji: '🔮'
  },
  'qwen2.5-coder-3b-instruct-q4_k_m.gguf': {
    displayName: 'Qwen 2.5 Coder 3B',
    description: 'Spezialisiert auf Programmierung. Kompakt und effizient.',
    capabilities: ['Code-Generierung', 'Code-Erklärung', 'Debugging'],
    params: '3B',
    minRam: '4 GB',
    url: 'https://huggingface.co/Qwen/Qwen2.5-Coder-3B-Instruct',
    emoji: '💻'
  },
  'qwen2.5-coder-7b-instruct-q4_k_m.gguf': {
    displayName: 'Qwen 2.5 Coder 7B',
    description: 'Professionelles Coder-Modell. Hervorragend für Software-Entwicklung.',
    capabilities: ['Code-Generierung', 'Refactoring', 'Code-Review', 'Multi-Sprachen'],
    params: '7B',
    minRam: '8 GB',
    url: 'https://huggingface.co/Qwen/Qwen2.5-Coder-7B-Instruct',
    emoji: '💻'
  },
  // DeepSeek
  'deepseek-coder-6.7b-instruct.Q4_K_M.gguf': {
    displayName: 'DeepSeek Coder 6.7B',
    description: 'Spezialisiertes Coder-Modell von DeepSeek. Top für Code-Aufgaben.',
    capabilities: ['Code-Generierung', 'Debugging', 'Code-Analyse', 'Dokumentation'],
    params: '6.7B',
    minRam: '6 GB',
    url: 'https://huggingface.co/deepseek-ai/deepseek-coder-6.7b-instruct',
    emoji: '🔍'
  },
  // Gemma
  'gemma-2-9b-it-Q5_K_M.gguf': {
    displayName: 'Gemma 2 9B',
    description: 'Googles Open-Source-Modell. Stark in Logik und Reasoning.',
    capabilities: ['Chat', 'Reasoning', 'Analyse', 'Zusammenfassungen'],
    params: '9B',
    minRam: '10 GB',
    url: 'https://huggingface.co/google/gemma-2-9b-it',
    emoji: '💎'
  },
  // Phi
  'phi-4.Q2_K.gguf': {
    displayName: 'Phi-4',
    description: 'Microsofts neuestes kleines Sprachmodell. Erstaunliche Qualität für die Größe.',
    capabilities: ['Chat', 'Reasoning', 'Mathematik', 'Code'],
    params: '14B',
    minRam: '8 GB',
    url: 'https://huggingface.co/microsoft/phi-4',
    emoji: 'Φ'
  },
  // LLaVA (Vision Models)
  'llava-phi-3-mini-int4.gguf': {
    displayName: 'LLaVA Phi-3 Mini',
    description: 'Kompaktes Vision-Modell. Kann Bilder analysieren und beschreiben.',
    capabilities: ['Bildanalyse', 'Bildbeschreibung', 'OCR-ähnlich'],
    params: '3.8B',
    minRam: '4 GB',
    url: 'https://huggingface.co/xtuner/llava-phi-3-mini-hf',
    emoji: '👁️',
    isVision: true
  },
  'llava-v1.5-7b-Q2_K.gguf': {
    displayName: 'LLaVA 1.5 7B (Q2)',
    description: 'Vision-Modell für Bildverständnis. Stark komprimiert.',
    capabilities: ['Bildanalyse', 'Visual QA', 'Bildbeschreibung'],
    params: '7B',
    minRam: '4 GB',
    url: 'https://huggingface.co/liuhaotian/llava-v1.5-7b',
    emoji: '👁️',
    isVision: true
  },
  'llava-v1.5-7b-Q4_K.gguf': {
    displayName: 'LLaVA 1.5 7B (Q4)',
    description: 'Vision-Modell mit besserer Qualität als Q2-Version.',
    capabilities: ['Bildanalyse', 'Visual QA', 'Detaillierte Beschreibungen'],
    params: '7B',
    minRam: '6 GB',
    url: 'https://huggingface.co/liuhaotian/llava-v1.5-7b',
    emoji: '👁️',
    isVision: true
  },
  'llava-v1.6-mistral-7b.Q4_K_M.gguf': {
    displayName: 'LLaVA 1.6 Mistral 7B',
    description: 'Neustes LLaVA mit Mistral-Basis. Beste Bildqualität.',
    capabilities: ['Hochwertige Bildanalyse', 'OCR', 'Chart-Verständnis'],
    params: '7B',
    minRam: '8 GB',
    url: 'https://huggingface.co/liuhaotian/llava-v1.6-mistral-7b',
    emoji: '👁️',
    isVision: true
  },
  // Projector (für Vision-Modelle)
  'mmproj-model-f16.gguf': {
    displayName: 'Vision Projector',
    description: 'Projektions-Modell für LLaVA. Wird zusammen mit Vision-Modellen benötigt.',
    capabilities: ['Bild-Encoder', 'Multimodal-Brücke'],
    params: '-',
    minRam: '2 GB',
    url: 'https://huggingface.co/mys/ggml_llava-v1.5-7b',
    emoji: '🔗',
    isProjector: true
  }
}

// Get model info with fallback
function getModelInfo(modelName) {
  // Try exact match first
  if (MODEL_INFO[modelName]) {
    return MODEL_INFO[modelName]
  }
  // Try case-insensitive match
  const lowerName = modelName.toLowerCase()
  for (const [key, value] of Object.entries(MODEL_INFO)) {
    if (key.toLowerCase() === lowerName) {
      return value
    }
  }
  // Return default info
  return {
    displayName: modelName.replace('.gguf', '').replace(/_/g, ' '),
    description: 'Lokales GGUF-Modell',
    capabilities: ['Chat'],
    params: '?',
    minRam: '?',
    url: null,
    emoji: '🤖'
  }
}

// Computed
const canProceed = computed(() => {
  switch (currentStep.value) {
    case 1:
      return wizardData.value.baseModel !== ''
    case 2:
      return wizardData.value.name.trim() !== ''
    default:
      return true
  }
})

const canCreate = computed(() => {
  return wizardData.value.baseModel !== '' &&
         wizardData.value.name.trim() !== '' &&
         providerAvailable.value
})

// Watch for show changes to reset and check provider
watch(() => props.show, (newVal) => {
  if (newVal) {
    resetWizard()
    checkProviderStatus()
  }
})

onMounted(() => {
  if (props.show) {
    checkProviderStatus()
  }
})

async function checkProviderStatus() {
  isCheckingProvider.value = true
  try {
    // Check llama-server status
    const status = await api.getLlamaServerStatus()
    providerAvailable.value = status?.online === true

    if (providerAvailable.value) {
      await loadAvailableModels()
    }
  } catch (error) {
    console.error('Error checking provider status:', error)
    providerAvailable.value = false
  } finally {
    isCheckingProvider.value = false
  }
}

async function loadAvailableModels() {
  isLoadingModels.value = true
  try {
    // Load GGUF models
    const response = await api.getAvailableModels()
    const modelNames = response?.models || response || []
    availableModels.value = modelNames.map(name => ({
      name: typeof name === 'string' ? name : name.name,
      size: null,
      parameterSize: null
    }))
  } catch (error) {
    console.error('Failed to load available models:', error)
    errorToast('Fehler beim Laden der Modelle')
    availableModels.value = []
  } finally {
    isLoadingModels.value = false
  }
}

function selectModel(model) {
  wizardData.value.baseModel = model.name
}

function canGoToStep(step) {
  if (step > currentStep.value) return false
  return true
}

function goToStep(step) {
  if (canGoToStep(step)) {
    currentStep.value = step
  }
}

function nextStep() {
  if (canProceed.value && currentStep.value < totalSteps) {
    currentStep.value++
  }
}

function previousStep() {
  if (currentStep.value > 1) {
    currentStep.value--
  }
}

function applyPromptTemplate(template) {
  wizardData.value.systemPrompt = template.prompt
}

function triggerPromptFileInput() {
  promptFileInput.value?.click()
}

async function handlePromptFileSelect(event) {
  const file = event.target.files?.[0]
  if (!file) return

  const validExtensions = ['.txt', '.md']
  const isValidType = validExtensions.some(ext => file.name.toLowerCase().endsWith(ext))

  if (!isValidType) {
    errorToast('Bitte eine .txt oder .md Datei wählen')
    event.target.value = ''
    return
  }

  if (file.size > 1024 * 1024) {
    errorToast('Datei zu groß (max. 1MB)')
    event.target.value = ''
    return
  }

  try {
    const text = await file.text()
    wizardData.value.systemPrompt = text
    success(`Prompt aus "${file.name}" geladen`)
  } catch (error) {
    errorToast('Fehler beim Lesen der Datei')
  } finally {
    event.target.value = ''
  }
}

async function createModel() {
  if (!canCreate.value) return

  isCreating.value = true
  creationProgress.value = 'Speichere GGUF-Konfiguration...'
  creationProgressPercent.value = 50

  try {
    const config = {
      name: wizardData.value.name.trim(),
      baseModel: wizardData.value.baseModel.trim(),
      description: wizardData.value.description.trim() || null,
      systemPrompt: wizardData.value.systemPrompt.trim() || null,
      temperature: wizardData.value.temperature,
      topP: wizardData.value.topP,
      topK: wizardData.value.topK,
      repeatPenalty: wizardData.value.repeatPenalty,
      maxTokens: wizardData.value.numPredict,
      contextSize: wizardData.value.numCtx,
      gpuLayers: 999
    }

    await api.createGgufModelConfig(config)
    creationProgressPercent.value = 100

    success('GGUF Konfiguration erfolgreich erstellt!')
    emit('created')
    emit('close')

  } catch (error) {
    console.error('Failed to create custom model:', error)
    errorToast('Fehler beim Erstellen: ' + (error.message || 'Unbekannter Fehler'))
  } finally {
    isCreating.value = false
    creationProgress.value = ''
    creationProgressPercent.value = 0
  }
}

function resetWizard() {
  currentStep.value = 1
  wizardData.value = {
    baseModel: '',
    name: '',
    description: '',
    systemPrompt: '',
    temperature: 0.8,
    topP: 0.9,
    topK: 40,
    repeatPenalty: 1.1,
    numPredict: 2048,
    numCtx: 8192
  }
  showAdvanced.value = false
  isCreating.value = false
  creationProgress.value = ''
  creationProgressPercent.value = 0
}

function formatSize(bytes) {
  if (!bytes) return ''
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) {
    return `${gb.toFixed(1)} GB`
  }
  const mb = bytes / (1024 * 1024)
  return `${mb.toFixed(0)} MB`
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

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s, max-height 0.3s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
