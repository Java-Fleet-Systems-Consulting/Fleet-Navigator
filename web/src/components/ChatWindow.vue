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
          <h3 class="text-lg font-semibold text-white mb-2">Modell wird geladen</h3>
          <p class="text-purple-300 text-sm">{{ chatStore.modelSwapMessage || 'Wechsle zum passenden Modell...' }}</p>
          <!-- Progress bar -->
          <div class="mt-4 w-full bg-gray-700 rounded-full h-2">
            <div
              class="bg-gradient-to-r from-purple-500 to-blue-500 h-2 rounded-full transition-all duration-300"
              :style="{ width: chatStore.modelSwapProgress + '%' }"
            ></div>
          </div>
          <p class="text-gray-500 text-xs mt-3">Modell-Wechsel für optimale Antworten</p>
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

          <!-- Personalisierte Begrüßung -->
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
            <!-- Web Search: Data Wave Icon -->
            <div v-if="chatStore.isWebSearching" class="web-search-icon p-3 rounded-2xl bg-gradient-to-br from-cyan-500 via-blue-500 to-purple-600 shadow-lg shadow-blue-500/30">
              <GlobeAltIcon class="w-6 h-6 text-white" />
            </div>
            <!-- Normal CPU Icon for regular loading -->
            <div v-else class="p-3 rounded-2xl bg-gradient-to-br from-fleet-orange-500 to-fleet-orange-600 shadow-lg">
              <CpuChipIcon class="w-6 h-6 text-white animate-pulse" />
            </div>
          </div>
          <div class="flex-1">
            <!-- Web Search Animation (dynamisch basierend auf Setting) -->
            <template v-if="chatStore.isWebSearching">
              <div class="flex items-center gap-3 mb-3">
                <span class="text-transparent bg-clip-text bg-gradient-to-r from-cyan-400 via-blue-500 to-purple-500 font-semibold text-sm">
                  {{ loadingText }}
                </span>
              </div>

              <!-- Data Wave Animation -->
              <div v-if="settingsStore.settings.webSearchAnimation === 'data-wave'" class="data-wave-container">
                <div class="data-wave">
                  <div class="wave-line"></div>
                  <div class="data-particles">
                    <span class="particle" style="--delay: 0s; --size: 4px;"></span>
                    <span class="particle" style="--delay: 0.3s; --size: 6px;"></span>
                    <span class="particle" style="--delay: 0.6s; --size: 3px;"></span>
                    <span class="particle" style="--delay: 0.9s; --size: 5px;"></span>
                    <span class="particle" style="--delay: 1.2s; --size: 4px;"></span>
                    <span class="particle" style="--delay: 1.5s; --size: 7px;"></span>
                  </div>
                </div>
              </div>

              <!-- Orbit Animation -->
              <div v-else-if="settingsStore.settings.webSearchAnimation === 'orbit'" class="orbit-container">
                <div class="orbit-center">
                  <div class="orbit-ring"></div>
                  <div class="orbit-ring orbit-ring-2"></div>
                  <div class="orbit-dot"></div>
                  <div class="orbit-dot orbit-dot-2"></div>
                  <div class="orbit-dot orbit-dot-3"></div>
                </div>
              </div>

              <!-- Radar Animation -->
              <div v-else-if="settingsStore.settings.webSearchAnimation === 'radar'" class="radar-container">
                <div class="radar-sweep"></div>
                <div class="radar-ring"></div>
                <div class="radar-ring radar-ring-2"></div>
                <div class="radar-ring radar-ring-3"></div>
                <div class="radar-blip" style="--angle: 45deg; --distance: 30%;"></div>
                <div class="radar-blip" style="--angle: 120deg; --distance: 60%;"></div>
                <div class="radar-blip" style="--angle: 200deg; --distance: 45%;"></div>
                <div class="radar-blip" style="--angle: 300deg; --distance: 70%;"></div>
              </div>

              <!-- Constellation Animation -->
              <div v-else-if="settingsStore.settings.webSearchAnimation === 'constellation'" class="constellation-container">
                <svg class="constellation-svg" viewBox="0 0 200 40">
                  <line class="constellation-line" x1="20" y1="20" x2="60" y2="15" />
                  <line class="constellation-line" x1="60" y1="15" x2="100" y2="25" />
                  <line class="constellation-line" x1="100" y1="25" x2="140" y2="18" />
                  <line class="constellation-line" x1="140" y1="18" x2="180" y2="22" />
                  <circle class="constellation-star" cx="20" cy="20" r="3" style="--delay: 0s;" />
                  <circle class="constellation-star" cx="60" cy="15" r="4" style="--delay: 0.2s;" />
                  <circle class="constellation-star" cx="100" cy="25" r="3" style="--delay: 0.4s;" />
                  <circle class="constellation-star" cx="140" cy="18" r="5" style="--delay: 0.6s;" />
                  <circle class="constellation-star" cx="180" cy="22" r="3" style="--delay: 0.8s;" />
                </svg>
              </div>

              <!-- Fallback (Data Wave) -->
              <div v-else class="data-wave-container">
                <div class="data-wave">
                  <div class="wave-line"></div>
                  <div class="data-particles">
                    <span class="particle" style="--delay: 0s; --size: 4px;"></span>
                    <span class="particle" style="--delay: 0.3s; --size: 6px;"></span>
                    <span class="particle" style="--delay: 0.6s; --size: 3px;"></span>
                  </div>
                </div>
              </div>
            </template>
            <!-- Normal Loading Animation: Orange dots -->
            <template v-else>
              <div class="flex items-center gap-2 mb-2">
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce"></div>
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce" style="animation-delay: 0.15s"></div>
                <div class="w-2.5 h-2.5 bg-fleet-orange-500 rounded-full animate-bounce" style="animation-delay: 0.3s"></div>
                <span class="ml-2 text-fleet-orange-500 dark:text-fleet-orange-400 font-medium text-sm">
                  {{ loadingText }}
                </span>
              </div>
              <!-- Typing indicator bars -->
              <div class="flex gap-1">
                <div class="h-2 w-12 rounded-full animate-pulse bg-gray-200 dark:bg-gray-700"></div>
                <div class="h-2 w-20 rounded-full animate-pulse bg-gray-200 dark:bg-gray-700" style="animation-delay: 0.1s"></div>
                <div class="h-2 w-16 rounded-full animate-pulse bg-gray-200 dark:bg-gray-700" style="animation-delay: 0.2s"></div>
              </div>
            </template>
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
  // Wenn Laden fertig UND wir auf Voice-Response warten UND TTS aktiviert
  const ttsEnabled = localStorage.getItem('ttsEnabled') !== 'false' // Default: true
  if (!isLoading && wasLoading && pendingVoiceResponse.value && ttsEnabled) {
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

/* ============================================
   DATA WAVE - Web Search Animation
   ============================================ */

/* Icon Container with Glow Pulse */
.web-search-icon {
  animation: icon-glow 2s ease-in-out infinite;
}

@keyframes icon-glow {
  0%, 100% {
    box-shadow: 0 0 20px rgba(59, 130, 246, 0.4),
                0 0 40px rgba(139, 92, 246, 0.2);
  }
  50% {
    box-shadow: 0 0 30px rgba(59, 130, 246, 0.6),
                0 0 60px rgba(139, 92, 246, 0.4);
  }
}

/* Data Wave Container */
.data-wave-container {
  width: 200px;
  height: 40px;
  position: relative;
  overflow: hidden;
  border-radius: 12px;
  background: linear-gradient(135deg,
    rgba(6, 182, 212, 0.1) 0%,
    rgba(59, 130, 246, 0.15) 50%,
    rgba(139, 92, 246, 0.1) 100%);
  border: 1px solid rgba(59, 130, 246, 0.2);
}

/* Wave Line - Animated SVG-like Path */
.wave-line {
  position: absolute;
  top: 50%;
  left: 0;
  width: 400%;
  height: 2px;
  background: linear-gradient(90deg,
    transparent 0%,
    rgba(6, 182, 212, 0.8) 10%,
    rgba(59, 130, 246, 1) 25%,
    rgba(139, 92, 246, 0.8) 40%,
    rgba(59, 130, 246, 1) 55%,
    rgba(6, 182, 212, 0.8) 70%,
    transparent 100%);
  animation: wave-flow 2s linear infinite;
  transform: translateY(-50%);
}

.wave-line::before,
.wave-line::after {
  content: '';
  position: absolute;
  top: 0;
  width: 100%;
  height: 100%;
  background: inherit;
}

.wave-line::before {
  top: -8px;
  opacity: 0.4;
  animation: wave-secondary 2s linear infinite;
  animation-delay: 0.3s;
}

.wave-line::after {
  top: 8px;
  opacity: 0.3;
  animation: wave-secondary 2s linear infinite;
  animation-delay: 0.6s;
}

@keyframes wave-flow {
  0% {
    transform: translateX(-50%) translateY(-50%);
  }
  100% {
    transform: translateX(0%) translateY(-50%);
  }
}

@keyframes wave-secondary {
  0% {
    transform: translateX(-50%) scaleY(0.5);
  }
  50% {
    transform: translateX(-25%) scaleY(1.5);
  }
  100% {
    transform: translateX(0%) scaleY(0.5);
  }
}

/* Data Particles - Floating through the wave */
.data-particles {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.particle {
  position: absolute;
  width: var(--size, 4px);
  height: var(--size, 4px);
  border-radius: 50%;
  background: linear-gradient(135deg, #06b6d4, #3b82f6, #8b5cf6);
  box-shadow: 0 0 8px rgba(59, 130, 246, 0.8),
              0 0 16px rgba(139, 92, 246, 0.4);
  animation: particle-flow 2s ease-in-out infinite;
  animation-delay: var(--delay, 0s);
  top: 50%;
  transform: translateY(-50%);
}

.particle:nth-child(1) { top: 30%; }
.particle:nth-child(2) { top: 70%; }
.particle:nth-child(3) { top: 45%; }
.particle:nth-child(4) { top: 55%; }
.particle:nth-child(5) { top: 35%; }
.particle:nth-child(6) { top: 65%; }

@keyframes particle-flow {
  0% {
    left: -10%;
    opacity: 0;
    transform: translateY(-50%) scale(0.5);
  }
  10% {
    opacity: 1;
    transform: translateY(-50%) scale(1);
  }
  50% {
    transform: translateY(calc(-50% + 4px)) scale(1.2);
  }
  90% {
    opacity: 1;
    transform: translateY(-50%) scale(1);
  }
  100% {
    left: 110%;
    opacity: 0;
    transform: translateY(-50%) scale(0.5);
  }
}

/* Extra glow effect for the wave container */
.data-wave-container::before {
  content: '';
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(ellipse at center,
    rgba(59, 130, 246, 0.15) 0%,
    transparent 70%);
  animation: container-glow 3s ease-in-out infinite;
}

@keyframes container-glow {
  0%, 100% {
    opacity: 0.5;
    transform: scale(1);
  }
  50% {
    opacity: 1;
    transform: scale(1.1);
  }
}

/* ============================================
   ORBIT ANIMATION - Kreisende Datenpunkte
   ============================================ */

.orbit-container {
  width: 200px;
  height: 40px;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg,
    rgba(99, 102, 241, 0.1) 0%,
    rgba(139, 92, 246, 0.15) 100%);
  border-radius: 12px;
  border: 1px solid rgba(99, 102, 241, 0.2);
  overflow: hidden;
}

.orbit-center {
  width: 30px;
  height: 30px;
  position: relative;
}

.orbit-ring {
  position: absolute;
  inset: 0;
  border: 1px solid rgba(99, 102, 241, 0.3);
  border-radius: 50%;
  animation: orbit-pulse 2s ease-in-out infinite;
}

.orbit-ring-2 {
  inset: -5px;
  border-color: rgba(139, 92, 246, 0.2);
  animation-delay: 0.5s;
}

.orbit-dot {
  position: absolute;
  width: 6px;
  height: 6px;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border-radius: 50%;
  box-shadow: 0 0 10px rgba(99, 102, 241, 0.8);
  animation: orbit-rotate 2s linear infinite;
  top: 50%;
  left: 50%;
  transform-origin: 0 0;
}

.orbit-dot-2 {
  animation-delay: -0.66s;
  background: linear-gradient(135deg, #8b5cf6, #a855f7);
}

.orbit-dot-3 {
  animation-delay: -1.33s;
  background: linear-gradient(135deg, #a855f7, #6366f1);
}

@keyframes orbit-rotate {
  0% {
    transform: rotate(0deg) translateX(18px) rotate(0deg);
  }
  100% {
    transform: rotate(360deg) translateX(18px) rotate(-360deg);
  }
}

@keyframes orbit-pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 0.5;
  }
  50% {
    transform: scale(1.1);
    opacity: 1;
  }
}

/* ============================================
   RADAR ANIMATION - Scanning Effekt
   ============================================ */

.radar-container {
  width: 200px;
  height: 40px;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg,
    rgba(16, 185, 129, 0.1) 0%,
    rgba(5, 150, 105, 0.15) 100%);
  border-radius: 12px;
  border: 1px solid rgba(16, 185, 129, 0.2);
  overflow: hidden;
}

.radar-sweep {
  position: absolute;
  width: 40px;
  height: 40px;
  background: conic-gradient(
    from 0deg,
    transparent 0deg,
    rgba(16, 185, 129, 0.4) 30deg,
    transparent 60deg
  );
  border-radius: 50%;
  animation: radar-spin 2s linear infinite;
}

.radar-ring {
  position: absolute;
  width: 20px;
  height: 20px;
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 50%;
}

.radar-ring-2 {
  width: 30px;
  height: 30px;
  border-color: rgba(16, 185, 129, 0.2);
}

.radar-ring-3 {
  width: 38px;
  height: 38px;
  border-color: rgba(16, 185, 129, 0.15);
}

.radar-blip {
  position: absolute;
  width: 4px;
  height: 4px;
  background: #10b981;
  border-radius: 50%;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.8);
  animation: radar-blink 1.5s ease-in-out infinite;
  top: 50%;
  left: 50%;
  transform: rotate(var(--angle)) translateX(calc(var(--distance) * 0.4)) translateY(-50%);
}

@keyframes radar-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes radar-blink {
  0%, 100% {
    opacity: 0.3;
    transform: rotate(var(--angle)) translateX(calc(var(--distance) * 0.4)) translateY(-50%) scale(0.8);
  }
  50% {
    opacity: 1;
    transform: rotate(var(--angle)) translateX(calc(var(--distance) * 0.4)) translateY(-50%) scale(1.2);
  }
}

/* ============================================
   CONSTELLATION ANIMATION - Sternbild-Netzwerk
   ============================================ */

.constellation-container {
  width: 200px;
  height: 40px;
  position: relative;
  background: linear-gradient(135deg,
    rgba(139, 92, 246, 0.1) 0%,
    rgba(236, 72, 153, 0.1) 100%);
  border-radius: 12px;
  border: 1px solid rgba(139, 92, 246, 0.2);
  overflow: hidden;
}

.constellation-svg {
  width: 100%;
  height: 100%;
}

.constellation-line {
  stroke: rgba(139, 92, 246, 0.4);
  stroke-width: 1;
  stroke-dasharray: 100;
  stroke-dashoffset: 100;
  animation: constellation-draw 2s ease-out infinite;
}

.constellation-star {
  fill: url(#star-gradient);
  animation: constellation-twinkle 1.5s ease-in-out infinite;
  animation-delay: var(--delay, 0s);
}

@keyframes constellation-draw {
  0% {
    stroke-dashoffset: 100;
    opacity: 0.3;
  }
  50% {
    stroke-dashoffset: 0;
    opacity: 1;
  }
  100% {
    stroke-dashoffset: -100;
    opacity: 0.3;
  }
}

@keyframes constellation-twinkle {
  0%, 100% {
    opacity: 0.5;
    transform: scale(0.8);
    fill: #8b5cf6;
  }
  50% {
    opacity: 1;
    transform: scale(1.2);
    fill: #ec4899;
  }
}
</style>
