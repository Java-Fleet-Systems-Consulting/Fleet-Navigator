<template>
  <div class="chat-window-container relative bg-gradient-to-b from-gray-50 to-gray-100 dark:from-gray-900 dark:to-gray-950">
    <!-- Error Banner - Shows LLM/Server errors to user -->
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="opacity-0 -translate-y-full"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-full"
    >
      <div
        v-if="chatStore.error"
        class="absolute top-0 left-0 right-0 z-50 mx-4 mt-4"
      >
        <div class="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-xl p-4 shadow-lg">
          <div class="flex items-start gap-3">
            <div class="flex-shrink-0">
              <svg class="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div class="flex-1">
              <h4 class="font-semibold text-red-800 dark:text-red-200 mb-1">{{ t('errors.generic') }}</h4>
              <p class="text-red-700 dark:text-red-300 text-sm">{{ chatStore.error }}</p>
            </div>
            <button
              @click="chatStore.clearError()"
              class="flex-shrink-0 p-1 rounded-lg hover:bg-red-100 dark:hover:bg-red-800/50 transition-colors"
            >
              <svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Expert Switching Overlay - Shows when context change requires server restart -->
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="chatStore.isSwitchingExpert"
        class="absolute inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      >
        <div class="bg-gray-800/95 border border-gray-700 rounded-2xl p-8 shadow-2xl text-center max-w-sm">
          <!-- Animated spinner -->
          <div class="mb-4 flex justify-center">
            <div class="relative w-16 h-16">
              <div class="absolute inset-0 rounded-full border-4 border-gray-600"></div>
              <div class="absolute inset-0 rounded-full border-4 border-t-fleet-orange-500 animate-spin"></div>
              <div class="absolute inset-2 rounded-full border-2 border-t-purple-500 animate-spin" style="animation-direction: reverse; animation-duration: 1.5s;"></div>
            </div>
          </div>
          <!-- Message -->
          <h3 class="text-lg font-semibold text-white mb-2">{{ t('chat.switchingExpert') }}</h3>
          <p class="text-gray-400 text-sm">{{ chatStore.switchingExpertMessage || t('chat.adjustingContext') }}</p>
          <p class="text-gray-500 text-xs mt-2">{{ t('chat.restartingServer') }}</p>
        </div>
      </div>
    </Transition>

    <!-- Model Swap Overlay - Shows when switching between Vision and Chat model (VRAM management) -->
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="chatStore.isSwappingModel"
        class="absolute inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      >
        <div class="bg-gray-800/95 border border-purple-700/50 rounded-2xl p-8 shadow-2xl text-center max-w-sm">
          <!-- Animated Vision spinner with eye icon -->
          <div class="mb-4 flex justify-center">
            <div class="relative w-20 h-20">
              <!-- Outer ring -->
              <div class="absolute inset-0 rounded-full border-4 border-gray-600"></div>
              <!-- Spinning purple ring -->
              <div class="absolute inset-0 rounded-full border-4 border-t-purple-500 border-r-purple-500/50 animate-spin"></div>
              <!-- Inner spinning ring -->
              <div class="absolute inset-2 rounded-full border-2 border-t-blue-400 animate-spin" style="animation-direction: reverse; animation-duration: 1.5s;"></div>
              <!-- Center eye icon -->
              <div class="absolute inset-0 flex items-center justify-center">
                <EyeIcon class="w-8 h-8 text-purple-400" />
              </div>
            </div>
          </div>
          <!-- Message -->
          <h3 class="text-lg font-semibold text-white mb-2">Vision-Modell wird geladen</h3>
          <p class="text-purple-300 text-sm">{{ chatStore.modelSwapMessage || 'Wechsle zum Bildanalyse-Modell...' }}</p>
          <!-- Progress bar -->
          <div class="mt-4 w-full bg-gray-700 rounded-full h-2">
            <div
              class="bg-gradient-to-r from-purple-500 to-blue-500 h-2 rounded-full transition-all duration-300"
              :style="{ width: chatStore.modelSwapProgress + '%' }"
            ></div>
          </div>
          <p class="text-gray-500 text-xs mt-3">VRAM-Optimierung für GPUs mit begrenztem Speicher</p>
        </div>
      </div>
    </Transition>

    <!-- Messages Area with Custom Scrollbar - scrollt bis zum Bildschirmende -->
    <div ref="messagesContainer" class="messages-area overflow-y-auto p-6 pb-32 space-y-4 custom-scrollbar">
      <!-- Welcome Message - Show when NO chat is selected -->
      <div v-if="!chatStore.currentChat" class="flex items-center justify-center min-h-full">
        <div class="text-center max-w-6xl mx-auto px-4 w-full">
          <!-- System Health Banner -->
          <SystemHealthBanner class="mb-6 text-left" />

          <!-- FLIP-FLOP: Werbung ODER personalisierte Begrüßung -->

          <!-- Option A: Werbung (Logo, Standard-Welcome, Tiles) -->
          <template v-if="settingsStore.settings.showWelcomeTiles">
            <!-- Animated Logo - JavaFleet Segelboot -->
            <div class="mb-8 flex justify-center">
              <div class="relative">
                <div class="absolute inset-0 bg-gradient-to-r from-fleet-orange-400 to-fleet-orange-600 rounded-full blur-3xl opacity-20 animate-pulse"></div>
                <div class="relative transform hover:scale-110 transition-transform duration-300">
                  <img
                    src="/javafleet-logo.png"
                    alt="JavaFleet Logo"
                    class="w-48 h-auto rounded-2xl shadow-2xl"
                  />
                </div>
              </div>
            </div>

            <!-- Welcome Text - Kreative Sprüche -->
            <h2 class="greeting-text text-3xl font-bold bg-gradient-to-r from-fleet-orange-500 via-orange-500 to-amber-500 bg-clip-text text-transparent mb-3 animate-fade-in">
              {{ greetingTitle }}
            </h2>
            <p class="text-lg text-gray-600 dark:text-gray-400 mb-2 animate-fade-in-delayed">
              {{ greetingSubtitle }}
            </p>
            <p class="text-sm text-gray-500 dark:text-gray-500 mb-8">
              {{ t('app.poweredBy') }}
            </p>
          </template>

          <!-- Option B: Personalisierte Begrüßung (wenn Werbung aus) -->
          <template v-else>
            <div class="py-8">
              <!-- Kreative Begrüßung mit Titel und Untertitel -->
              <h2 class="greeting-text text-4xl font-bold bg-gradient-to-r from-fleet-orange-500 via-orange-500 to-amber-500 bg-clip-text text-transparent mb-3 animate-fade-in">
                {{ greetingTitle }}
              </h2>
              <p class="text-xl text-gray-600 dark:text-gray-400 mb-8 animate-fade-in-delayed">
                {{ greetingSubtitle }}
              </p>
              <!-- Zentrierte Eingabe im Hero-Stil -->
              <div class="max-w-6xl mx-auto w-full px-4">
                <MessageInput @send="handleSendMessage"  :hero-mode="true" />
              </div>
            </div>
          </template>

          <!-- Suggestion Cards - nur wenn Werbung an -->
          <div v-if="settingsStore.settings.showWelcomeTiles" class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <!-- Brief schreiben -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.letter.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-blue-500/10 group-hover:bg-blue-500/20 transition-colors">
                  <DocumentTextIcon class="w-6 h-6 text-blue-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.letter.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.letter.description') }}</div>
            </button>

            <!-- Fragen stellen -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.question.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-green-500/10 group-hover:bg-green-500/20 transition-colors">
                  <LightBulbIcon class="w-6 h-6 text-green-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.question.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.question.description') }}</div>
            </button>

            <!-- Übersetzen -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.translate.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-purple-500/10 group-hover:bg-purple-500/20 transition-colors">
                  <LanguageIcon class="w-6 h-6 text-purple-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.translate.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.translate.description') }}</div>
            </button>

            <!-- Lernen -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.learn.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-orange-500/10 group-hover:bg-orange-500/20 transition-colors">
                  <AcademicCapIcon class="w-6 h-6 text-orange-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.learn.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.learn.description') }}</div>
            </button>

            <!-- Programmieren -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.code.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-indigo-500/10 group-hover:bg-indigo-500/20 transition-colors">
                  <CodeBracketIcon class="w-6 h-6 text-indigo-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.code.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.code.description') }}</div>
            </button>

            <!-- Kreativ schreiben -->
            <button
              @click="sendSuggestion(t('welcome.suggestions.creative.prompt'))"
              class="
                group p-5 rounded-2xl
                bg-white/80 dark:bg-gray-800/80
                backdrop-blur-sm
                border border-gray-200/50 dark:border-gray-700/50
                hover:border-fleet-orange-400 dark:hover:border-fleet-orange-500
                transition-all duration-300
                transform hover:scale-105 hover:shadow-xl
                text-left
              "
            >
              <div class="flex items-center gap-3 mb-2">
                <div class="p-2 rounded-xl bg-pink-500/10 group-hover:bg-pink-500/20 transition-colors">
                  <SparklesIcon class="w-6 h-6 text-pink-500" />
                </div>
                <div class="font-semibold text-gray-800 dark:text-gray-100">{{ t('welcome.suggestions.creative.title') }}</div>
              </div>
              <div class="text-sm text-gray-500 dark:text-gray-400 ml-11">{{ t('welcome.suggestions.creative.description') }}</div>
            </button>

            <!-- Zentrierte Eingabe unter den Tiles - NUR wenn Tiles sichtbar -->
            <div class="col-span-full max-w-6xl mx-auto w-full mt-8 px-4">
              <MessageInput @send="handleSendMessage"  :hero-mode="true" />
            </div>
          </div>
        </div>
      </div>

      <!-- Messages with Smooth Transitions - Only show when chat is selected -->
      <template v-if="chatStore.currentChat">
        <TransitionGroup name="message">
          <div
            v-for="(message, index) in chatStore.messages"
            :key="(message.id || index) + '-' + (message.isStreaming ? 'stream' : 'done')"
            class="message-item"
          >
            <MessageBubble :message="message" @delete="handleDeleteMessage" />
          </div>
        </TransitionGroup>

        <!-- Enhanced Loading Indicator -->
        <div v-if="chatStore.isLoading" class="flex items-start gap-4 p-4">
          <div class="flex-shrink-0">
            <!-- Spinning Globe for Web Search -->
            <div v-if="chatStore.isWebSearching" class="p-3 rounded-2xl bg-gradient-to-br from-blue-500 to-blue-600 shadow-lg">
              <GlobeAltIcon class="w-6 h-6 text-white animate-spin-slow" />
            </div>
            <!-- Normal CPU Icon for regular loading -->
            <div v-else class="p-3 rounded-2xl bg-gradient-to-br from-fleet-orange-500 to-fleet-orange-600 shadow-lg">
              <CpuChipIcon class="w-6 h-6 text-white animate-pulse" />
            </div>
          </div>
          <div class="flex-1">
            <div class="flex items-center gap-2 mb-2">
              <!-- Web Search Animation: Pulsing blue dots -->
              <template v-if="chatStore.isWebSearching">
                <div class="w-2.5 h-2.5 bg-blue-500 rounded-full animate-bounce"></div>
                <div class="w-2.5 h-2.5 bg-blue-500 rounded-full animate-bounce" style="animation-delay: 0.15s"></div>
                <div class="w-2.5 h-2.5 bg-blue-500 rounded-full animate-bounce" style="animation-delay: 0.3s"></div>
                <span class="ml-2 text-blue-500 dark:text-blue-400 font-medium text-sm">
                  {{ loadingText }}
                </span>
              </template>
              <!-- Normal Loading Animation: Orange dots -->
              <template v-else>
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce"></div>
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce" style="animation-delay: 0.15s"></div>
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce" style="animation-delay: 0.3s"></div>
                <span class="ml-2 text-fleet-orange-500 dark:text-fleet-orange-400 font-medium text-sm">
                  {{ loadingText }}
                </span>
              </template>
            </div>
            <!-- Typing indicator bars -->
            <div class="flex gap-1">
              <div class="h-2 w-12 rounded-full animate-pulse" :class="chatStore.isWebSearching ? 'bg-blue-200 dark:bg-blue-800' : 'bg-gray-200 dark:bg-gray-700'"></div>
              <div class="h-2 w-20 rounded-full animate-pulse" :class="chatStore.isWebSearching ? 'bg-blue-200 dark:bg-blue-800' : 'bg-gray-200 dark:bg-gray-700'" style="animation-delay: 0.1s"></div>
              <div class="h-2 w-16 rounded-full animate-pulse" :class="chatStore.isWebSearching ? 'bg-blue-200 dark:bg-blue-800' : 'bg-gray-200 dark:bg-gray-700'" style="animation-delay: 0.2s"></div>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Message Input - fixiert am unteren Rand -->
    <div v-if="chatStore.currentChat" class="fixed-input-container absolute bottom-0 left-0 right-0 z-10">
      <MessageInput @send="handleSendMessage"  />
    </div>

  </div>
</template>

<script setup>
import { ref, watch, nextTick, computed, onMounted } from 'vue'
import {
  LightBulbIcon,
  CodeBracketIcon,
  CpuChipIcon,
  DocumentTextIcon,
  LanguageIcon,
  AcademicCapIcon,
  SparklesIcon,
  GlobeAltIcon,
  DocumentArrowDownIcon,
  ArrowPathIcon,
  EyeIcon
} from '@heroicons/vue/24/outline'
import { useChatStore } from '../stores/chatStore'
import { useSettingsStore } from '../stores/settingsStore'
import { useLocale } from '../composables/useLocale'
import MessageBubble from './MessageBubble.vue'
import MessageInput from './MessageInput.vue'
import SystemHealthBanner from './SystemHealthBanner.vue'
import api from '../services/api'
import { secureFetch } from '../utils/secureFetch'

const chatStore = useChatStore()
const settingsStore = useSettingsStore()
const { t } = useLocale()
const messagesContainer = ref(null)
const isGeneratingPdf = ref(false)

// Auto-TTS nach Spracheingabe
const pendingVoiceResponse = ref(false)
let currentAudio = null

// Persönliche Daten für personalisierte Begrüßung
const personalInfo = ref(null)

// Computed: Current expert name for display
const currentExpertName = computed(() => {
  if (chatStore.selectedExpertId) {
    const expert = chatStore.getExpertById(chatStore.selectedExpertId)
    return expert ? expert.name : t('experts.title')
  }
  return t('experts.title')
})

// Generate PDF summary with expert
async function generateExpertSummaryPdf() {
  if (!chatStore.currentChat?.id || !chatStore.selectedExpertId) return

  isGeneratingPdf.value = true
  try {
    const response = await secureFetch(`/api/chat/${chatStore.currentChat.id}/expert-summary-pdf`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        expertId: chatStore.selectedExpertId
      })
    })

    if (!response.ok) {
      throw new Error(`HTTP error: ${response.status}`)
    }

    // Get the blob and download
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)

    // Get filename from Content-Disposition header or create default
    const contentDisposition = response.headers.get('Content-Disposition')
    let filename = `Abschlussbericht_${currentExpertName.value}_${new Date().toISOString().split('T')[0]}.pdf`
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="?([^";\n]+)"?/)
      if (filenameMatch) {
        filename = filenameMatch[1]
      }
    }

    // Trigger download
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)

    console.log('✅ PDF downloaded:', filename)
  } catch (error) {
    console.error('Failed to generate PDF summary:', error)
    alert(t('errors.generic') + ': ' + error.message)
  } finally {
    isGeneratingPdf.value = false
  }
}

// Kreative, freundliche und motivierende Begrüßungen nach Tageszeit
const creativeGreetings = {
  morning: [
    { title: "Guten Morgen, Sonnenschein!", subtitle: "Ein wundervoller Tag wartet auf dich." },
    { title: "Einen wunderschönen guten Morgen!", subtitle: "Heute wird ein großartiger Tag!" },
    { title: "Schön, dass du da bist!", subtitle: "Der perfekte Start in den Tag." },
    { title: "Guten Morgen!", subtitle: "Lass uns gemeinsam Großes erreichen." },
    { title: "Die Sonne lacht - und ich auch!", subtitle: "Worauf freust du dich heute?" },
    { title: "Frisch und motiviert!", subtitle: "Zusammen schaffen wir alles." },
    { title: "Ein neuer Tag voller Chancen!", subtitle: "Ich freue mich, dir zu helfen." },
    { title: "Morgenstund hat Gold im Mund!", subtitle: "Was möchtest du heute erreichen?" },
  ],
  noon: [
    { title: "Schönen Mittag!", subtitle: "Du machst das großartig heute!" },
    { title: "Halbzeit - und du rockst!", subtitle: "Weiter so, ich unterstütze dich." },
    { title: "Die Sonne steht hoch!", subtitle: "Perfekter Moment für neue Ideen." },
    { title: "Mittags-Power!", subtitle: "Gemeinsam schaffen wir das." },
    { title: "Schön, dich zu sehen!", subtitle: "Wie kann ich deinen Tag bereichern?" },
    { title: "Der Tag läuft!", subtitle: "Ich bin hier, wenn du mich brauchst." },
  ],
  afternoon: [
    { title: "Einen schönen Nachmittag!", subtitle: "Du schaffst das - ich glaub an dich!" },
    { title: "Hey, schön dass du da bist!", subtitle: "Lass uns zusammen produktiv sein." },
    { title: "Nachmittags-Energie!", subtitle: "Zeit für die spannenden Aufgaben." },
    { title: "Der Nachmittag gehört dir!", subtitle: "Ich bin bereit, dir zu helfen." },
    { title: "Wunderbar, dich zu sehen!", subtitle: "Was steht heute noch an?" },
    { title: "Auf geht's!", subtitle: "Gemeinsam sind wir unschlagbar." },
  ],
  evening: [
    { title: "Einen schönen Abend!", subtitle: "Zeit für entspannte Produktivität." },
    { title: "Guten Abend!", subtitle: "Schön, dass du noch vorbeischaust." },
    { title: "Der Abend ist jung!", subtitle: "Lass uns noch etwas Tolles machen." },
    { title: "Abendstimmung!", subtitle: "Die perfekte Zeit für gute Ideen." },
    { title: "Willkommen am Abend!", subtitle: "Ich bin gerne für dich da." },
    { title: "Schöner Abend!", subtitle: "Was beschäftigt dich gerade?" },
  ],
  night: [
    { title: "Gute Nacht... oder doch nicht?", subtitle: "Schön, dass du noch Zeit für mich hast!" },
    { title: "Nachts sind die besten Ideen wach!", subtitle: "Lass uns kreativ sein." },
    { title: "Nachteulen-Club!", subtitle: "Willkommen, ich freue mich auf dich." },
    { title: "Die Welt schläft - wir nicht!", subtitle: "Zeit für die großen Gedanken." },
    { title: "Stille Nacht, kreative Nacht!", subtitle: "Was darf ich für dich tun?" },
    { title: "Späte Stunde, gute Stunde!", subtitle: "Ich bin hellwach und bereit." },
  ]
}

// Zufällige Begrüßung auswählen (einmal pro Session)
const selectedGreeting = ref(null)

function getRandomGreeting() {
  const hour = new Date().getHours()
  let timeOfDay = 'afternoon'

  if (hour >= 5 && hour < 10) timeOfDay = 'morning'
  else if (hour >= 10 && hour < 12) timeOfDay = 'noon'
  else if (hour >= 12 && hour < 17) timeOfDay = 'afternoon'
  else if (hour >= 17 && hour < 22) timeOfDay = 'evening'
  else timeOfDay = 'night'

  const greetings = creativeGreetings[timeOfDay]
  const randomIndex = Math.floor(Math.random() * greetings.length)
  return greetings[randomIndex]
}

// Initialisiere Begrüßung nur einmal
if (!selectedGreeting.value) {
  selectedGreeting.value = getRandomGreeting()
}

// Computed: Personalisierte Begrüßung mit Name
const personalGreeting = computed(() => {
  const greeting = selectedGreeting.value || getRandomGreeting()

  // Wenn persönliche Daten vorhanden, Name einbauen
  if (personalInfo.value) {
    const { title, firstName, lastName } = personalInfo.value
    let name = ''
    if (title) name += title + ' '
    if (firstName) name += firstName
    else if (lastName) name += lastName
    name = name.trim()

    if (name) {
      // Ersetze generische Anrede durch personalisierte
      return {
        title: `Hey ${name}! ${greeting.title}`,
        subtitle: greeting.subtitle
      }
    }
  }

  return greeting
})

// Computed: Greeting Subtitle
const greetingSubtitle = computed(() => {
  if (typeof personalGreeting.value === 'object') {
    return personalGreeting.value.subtitle
  }
  return "Wie kann ich dir heute helfen?"
})

// Computed: Greeting Title (für Template-Kompatibilität)
const greetingTitle = computed(() => {
  if (typeof personalGreeting.value === 'object') {
    return personalGreeting.value.title
  }
  return personalGreeting.value || "Willkommen zurück!"
})

// Lade persönliche Daten beim Start
onMounted(async () => {
  try {
    personalInfo.value = await api.getPersonalInfo()
  } catch (error) {
    console.debug('No personal info available')
  }
})

// Computed: Loading-Text basierend auf Websuche-Status
const loadingText = computed(() => {
  if (chatStore.isWebSearching) {
    return t('loading.searchingAndThinking')
  }
  return t('loading.thinking')
})

// Auto-scroll to bottom when new messages arrive
watch(() => chatStore.messages?.length ?? 0, async () => {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTo({
      top: messagesContainer.value.scrollHeight,
      behavior: 'smooth'
    })
  }
})

async function handleSendMessage(messageData) {
  // Handle both string (from suggestions) and object (from MessageInput with files)
  console.log('📨 handleSendMessage called:', typeof messageData, messageData)
  if (typeof messageData === 'string') {
    await chatStore.sendMessage({ text: messageData, files: [] })
  } else {
    // Prüfen ob Spracheingabe → Auto-TTS aktivieren
    console.log('📨 messageData.wasVoiceInput:', messageData.wasVoiceInput)
    if (messageData.wasVoiceInput) {
      pendingVoiceResponse.value = true
      console.log('🎤 Voice input detected - will speak response, pendingVoiceResponse:', pendingVoiceResponse.value)
    }
    await chatStore.sendMessage(messageData)
  }
}

// Watch für Auto-TTS nach Antwort
watch(() => chatStore.isLoading, async (isLoading, wasLoading) => {
  console.log('👁️ isLoading changed:', wasLoading, '→', isLoading, '| pendingVoiceResponse:', pendingVoiceResponse.value)
  // Wenn Laden fertig UND wir auf Voice-Response warten
  if (!isLoading && wasLoading && pendingVoiceResponse.value) {
    pendingVoiceResponse.value = false
    console.log('✅ Triggering Auto-TTS!')

    // Kurz warten damit der Store die finale Nachricht hat
    await new Promise(resolve => setTimeout(resolve, 500))

    // Letzte Assistenten-Nachricht holen
    const lastMessage = chatStore.messages[chatStore.messages.length - 1]
    console.log('📝 Last message:', lastMessage?.role, lastMessage?.content?.substring(0, 50))
    if (lastMessage && lastMessage.role === 'ASSISTANT' && lastMessage.content) {
      console.log('🔊 Speaking response...')

      // Experten-Stimme holen
      let voiceId = null
      if (chatStore.selectedExpertId) {
        const expert = chatStore.getExpertById(chatStore.selectedExpertId)
        if (expert?.voice) {
          voiceId = expert.voice
          console.log('🎤 Using expert voice:', voiceId)
        }
      }

      // TTS aufrufen
      try {
        const audioBlob = await api.synthesizeSpeech(lastMessage.content, voiceId)
        if (currentAudio) {
          currentAudio.pause()
        }
        const audioUrl = URL.createObjectURL(audioBlob)
        currentAudio = new Audio(audioUrl)
        currentAudio.onended = () => URL.revokeObjectURL(audioUrl)
        await currentAudio.play()
      } catch (err) {
        console.error('TTS error:', err)
      }
    }
  }
})

async function handleDeleteMessage(messageId) {
  if (!chatStore.currentChat?.id) return

  try {
    await secureFetch(`/api/chat/${chatStore.currentChat.id}/messages/${messageId}`, {
      method: 'DELETE'
    })
    // Remove message from local state
    chatStore.messages = chatStore.messages.filter(m => m.id !== messageId)
  } catch (err) {
    console.error('Failed to delete message:', err)
  }
}

function sendSuggestion(text) {
  handleSendMessage(text)
}
</script>

<style scoped>
/* Chat Window Layout - Messages scrollen unter dem Input durch */
.chat-window-container {
  height: 100%;
  position: relative;
  overflow: hidden;
}

.messages-area {
  height: 100%;
  overflow-y: auto;
}

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

/* Message Transitions */
.message-enter-active {
  animation: message-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.message-leave-active {
  animation: message-out 0.3s ease-in;
}

@keyframes message-in {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes message-out {
  from {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
  to {
    opacity: 0;
    transform: translateY(-10px) scale(0.95);
  }
}

/* Slow spinning animation for globe icon */
@keyframes spin-slow {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.animate-spin-slow {
  animation: spin-slow 2s linear infinite;
}

/* Greeting Animations */
.animate-fade-in {
  animation: fade-in 0.8s ease-out forwards;
}

.animate-fade-in-delayed {
  animation: fade-in 0.8s ease-out 0.3s forwards;
  opacity: 0;
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Greeting Text Glow Effect */
.greeting-text {
  text-shadow: 0 0 40px rgba(249, 115, 22, 0.2);
}
</style>
