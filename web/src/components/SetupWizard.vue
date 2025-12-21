<template>
  <div class="setup-wizard-overlay">
    <div class="setup-wizard">
      <!-- Header mit Schritt-Anzeige -->
      <div class="wizard-header">
        <div class="logo">
          <span class="logo-icon">🚢</span>
          <span class="logo-text">Fleet Navigator</span>
        </div>
        <div class="steps-indicator">
          <div
            v-for="(step, index) in steps"
            :key="index"
            class="step-dot"
            :class="{
              active: currentStep === index,
              completed: currentStep > index
            }"
          >
            <span class="step-number">{{ index + 1 }}</span>
          </div>
        </div>
      </div>

      <!-- Wizard Content -->
      <div class="wizard-content">
        <!-- Step 0: Sprachauswahl -->
        <div v-if="currentStep === 0" class="step-content language-step">
          <div class="language-icon">🌍</div>
          <h1>{{ t('setup.language.title') }}</h1>
          <p class="subtitle">{{ t('languages.chooseYourLanguage') }}</p>

          <div class="language-grid">
            <button
              class="language-card"
              :class="{ selected: selectedLocale === 'de' }"
              @click="selectLanguage('de')"
            >
              <span class="language-flag">🇩🇪</span>
              <span class="language-name">Deutsch</span>
            </button>
            <button
              class="language-card"
              :class="{ selected: selectedLocale === 'en' }"
              @click="selectLanguage('en')"
            >
              <span class="language-flag">🇬🇧</span>
              <span class="language-name">English</span>
            </button>
            <button
              class="language-card"
              :class="{ selected: selectedLocale === 'tr' }"
              @click="selectLanguage('tr')"
            >
              <span class="language-flag">🇹🇷</span>
              <span class="language-name">Türkçe</span>
            </button>
          </div>
        </div>

        <!-- Step 1: Willkommen -->
        <div v-if="currentStep === 1" class="step-content welcome-step">
          <div class="welcome-icon">🎉</div>
          <h1>{{ t('setup.welcome.title') }}</h1>
          <p class="subtitle">{{ t('setup.welcome.subtitle') }}</p>
          <div class="features-grid">
            <div class="feature">
              <span class="feature-icon">🤖</span>
              <span class="feature-text">{{ t('setup.welcome.features.localAI') }}</span>
            </div>
            <div class="feature">
              <span class="feature-icon">👨‍💼</span>
              <span class="feature-text">{{ t('setup.welcome.features.experts') }}</span>
            </div>
            <div class="feature">
              <span class="feature-icon">🔒</span>
              <span class="feature-text">{{ t('setup.welcome.features.privacy') }}</span>
            </div>
            <div class="feature">
              <span class="feature-icon">🎤</span>
              <span class="feature-text">{{ t('setup.welcome.features.voice') }}</span>
            </div>
          </div>
          <p class="setup-info">{{ t('setup.welcome.setupInfo') }}</p>
        </div>

        <!-- Step 2: Systemprüfung -->
        <div v-if="currentStep === 2" class="step-content system-step">
          <h2>🖥️ {{ t('setup.system.title') }}</h2>
          <p>{{ t('setup.system.subtitle') }}</p>

          <div v-if="isLoading" class="loading-spinner">
            <div class="spinner"></div>
            <span>{{ t('setup.system.analyzing') }}</span>
          </div>

          <div v-else-if="systemInfo" class="system-info-card">
            <div class="info-row">
              <span class="label">{{ t('setup.system.os') }}:</span>
              <span class="value">{{ formatOS(systemInfo.os) }} ({{ systemInfo.arch }})</span>
            </div>
            <div class="info-row">
              <span class="label">{{ t('setup.system.cpuCores') }}:</span>
              <span class="value">{{ systemInfo.cpuCores }}</span>
            </div>
            <div class="info-row">
              <span class="label">{{ t('setup.system.ram') }}:</span>
              <span class="value">{{ systemInfo.totalRamGB }} GB ({{ systemInfo.availableRamGB }} GB {{ t('setup.system.ramFree') }})</span>
            </div>
            <div class="info-row" :class="{ 'has-gpu': systemInfo.hasGpu }">
              <span class="label">{{ t('setup.system.gpu') }}:</span>
              <span class="value">
                <template v-if="systemInfo.hasGpu">
                  ✅ {{ systemInfo.gpuName }} ({{ systemInfo.gpuMemoryGB }} GB)
                </template>
                <template v-else>
                  ❌ {{ t('setup.system.noGpu') }}
                </template>
              </span>
            </div>
          </div>

          <div v-if="systemInfo" class="info-box" :class="systemInfo.hasGpu ? 'success' : 'info'">
            <span class="icon">{{ systemInfo.hasGpu ? '🚀' : '💡' }}</span>
            <span v-if="systemInfo.hasGpu">{{ t('setup.system.withGpuInfo') }}</span>
            <span v-else>{{ t('setup.system.withoutGpuInfo') }}</span>
          </div>
        </div>

        <!-- Step 3: Modell-Auswahl -->
        <div v-if="currentStep === 3" class="step-content model-step">
          <h2>🧠 {{ t('setup.model.title') }}</h2>
          <p>{{ t('setup.model.subtitle') }}</p>

          <!-- Hardware-Warnung wenn RAM < 8GB und keine GPU -->
          <div v-if="systemInfo && systemInfo.totalRamGB < 8 && !systemInfo.hasGpu" class="info-box warning">
            <span class="icon">⛔</span>
            <span>{{ t('setup.model.ramWarning', { ram: systemInfo.totalRamGB }) }}</span>
          </div>

          <!-- CPU-Info Tabelle wenn keine GPU -->
          <div v-if="systemInfo && !systemInfo.hasGpu && systemInfo.totalRamGB >= 8" class="cpu-info-card">
            <h3>💡 {{ t('setup.model.cpuInfo.title') }}</h3>
            <table class="cpu-comparison">
              <thead>
                <tr>
                  <th>{{ t('setup.model.cpuInfo.model') }}</th>
                  <th>{{ t('setup.model.cpuInfo.responseTime') }}</th>
                  <th>{{ t('setup.model.cpuInfo.quality') }}</th>
                  <th>{{ t('setup.model.cpuInfo.ram') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 8 && systemInfo.totalRamGB < 16 }">
                  <td><strong>1.5B</strong></td>
                  <td>⚡ 2-5 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.simple') }}</td>
                  <td>8 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 16 && systemInfo.totalRamGB < 32 }">
                  <td><strong>3B</strong></td>
                  <td>⏱️ 5-15 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.everyday') }}</td>
                  <td>16 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 32 && systemInfo.totalRamGB < 48 }">
                  <td><strong>7B</strong></td>
                  <td>🕐 20-45 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.good') }}</td>
                  <td>32 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 32 && systemInfo.totalRamGB < 40 }">
                  <td><strong>9B</strong> <small>(Gemma)</small></td>
                  <td>🕐 30-60 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.veryGood') }}</td>
                  <td>32 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 40 && systemInfo.totalRamGB < 48 }">
                  <td><strong>Phi-4</strong> <small>(Microsoft)</small></td>
                  <td>🕐 45-75 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.strongReasoning') }}</td>
                  <td>40 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 48 && systemInfo.totalRamGB < 64 }">
                  <td><strong>14B</strong></td>
                  <td>🕐 60-90 {{ t('setup.model.seconds') }}</td>
                  <td>{{ t('setup.model.qualities.premium') }}</td>
                  <td>48 GB+</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.totalRamGB >= 64 }">
                  <td><strong>32B</strong></td>
                  <td>🕐 2-3 {{ t('setup.model.minutes') }}</td>
                  <td>{{ t('setup.model.qualities.best') }}</td>
                  <td>64 GB+</td>
                </tr>
              </tbody>
            </table>
            <p class="cpu-hint">{{ t('setup.model.cpuInfo.hint') }}</p>
          </div>

          <!-- GPU-Info wenn GPU vorhanden -->
          <div v-if="systemInfo && systemInfo.hasGpu" class="gpu-info-card">
            <h3>🚀 {{ t('setup.model.gpuInfo.title', { gpu: systemInfo.gpuName, vram: systemInfo.gpuMemoryGB }) }}</h3>
            <table class="gpu-comparison">
              <thead>
                <tr><th>{{ t('setup.model.cpuInfo.model') }}</th><th>{{ t('setup.model.gpuInfo.vram') }}</th><th>{{ t('setup.model.gpuInfo.speed') }}</th></tr>
              </thead>
              <tbody>
                <tr :class="{ 'recommended-row': systemInfo.gpuMemoryGB >= 8 && systemInfo.gpuMemoryGB < 10 }">
                  <td><strong>7B</strong></td><td>8 GB+</td><td>⚡ 15-25 t/s</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.gpuMemoryGB >= 10 && systemInfo.gpuMemoryGB < 12 }">
                  <td><strong>9B</strong> <small>(Gemma)</small></td><td>10 GB+</td><td>⚡ 12-20 t/s</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.gpuMemoryGB >= 12 && systemInfo.gpuMemoryGB < 16 }">
                  <td><strong>Phi-4</strong> <small>(Microsoft)</small></td><td>12 GB+</td><td>⚡ 10-18 t/s</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.gpuMemoryGB >= 16 && systemInfo.gpuMemoryGB < 24 }">
                  <td><strong>14B</strong></td><td>16 GB+</td><td>⏱️ 10-18 t/s</td>
                </tr>
                <tr :class="{ 'recommended-row': systemInfo.gpuMemoryGB >= 24 }">
                  <td><strong>32B</strong></td><td>24 GB+</td><td>🕐 5-10 t/s</td>
                </tr>
              </tbody>
            </table>
            <p class="gpu-hint">t/s = Tokens pro Sekunde. Wähle nur verfügbare Modelle!</p>
          </div>

          <div class="models-list">
            <div
              v-for="model in modelRecommendations"
              :key="model.modelId"
              class="model-card"
              :class="{
                selected: selectedModel === model.modelId,
                recommended: model.recommended,
                disabled: !model.available
              }"
              @click="selectModel(model)"
            >
              <div class="model-header">
                <span class="model-name">{{ model.modelName }}</span>
                <span v-if="model.recommended" class="recommended-badge">Empfohlen</span>
                <span v-if="!model.available" class="unavailable-badge">Nicht verfügbar</span>
              </div>
              <div class="model-details">
                <span class="model-size">{{ model.sizeGB }} GB</span>
                <span class="model-desc">{{ model.description }}</span>
              </div>
              <div v-if="model.reason" class="model-reason" :class="{ 'reason-warning': !model.available }">{{ model.reason }}</div>
            </div>
          </div>

          <!-- Download Modal (wiederverwendete Komponente) -->
          <ModelDownloadModal
            :isVisible="isDownloading"
            :currentModel="selectedModelName"
            :progress="Math.round(downloadProgress.percent || 0)"
            :downloadedSize="formatBytes(downloadProgress.bytesDone)"
            :totalSize="formatBytes(downloadProgress.bytesTotal)"
            :speed="downloadProgress.speedMBps ? downloadProgress.speedMBps.toFixed(1) : '0.0'"
            :statusMessages="downloadStatusMessages"
            @cancel="cancelDownload"
          />
        </div>

        <!-- Step 4: Vision/Dokumentenerkennung -->
        <div v-if="currentStep === 4" class="step-content vision-step">
          <h2>📷 {{ t('setup.vision.title') }}</h2>
          <p>{{ t('setup.vision.subtitle') }}</p>

          <!-- Erklärung für User -->
          <div class="info-box info" style="margin-bottom: 20px;">
            <span class="icon">💡</span>
            <span>
              <strong>{{ t('setup.vision.infoTitle') }}</strong><br>
              • {{ t('setup.vision.infoList.documents') }}<br>
              • {{ t('setup.vision.infoList.sender') }}<br>
              • {{ t('setup.vision.infoList.priority') }}<br>
              • {{ t('setup.vision.infoList.photos') }}
            </span>
          </div>

          <!-- Wenn Vision-Optionen geladen -->
          <div v-if="visionOptions">
            <!-- Hardware-Warnung wenn keine Option verfügbar -->
            <div v-if="!visionOptions.models?.some(m => m.available)" class="info-box warning">
              <span class="icon">⚠️</span>
              <span>{{ t('setup.vision.noResourcesWarning') }}</span>
            </div>

            <!-- Vision-Modell Auswahl -->
            <div class="models-list">
              <div
                v-for="model in visionOptions.models"
                :key="model.id"
                class="model-card"
                :class="{
                  selected: selectedVisionModel === model.id,
                  recommended: model.recommended,
                  disabled: !model.available
                }"
                @click="selectVisionModel(model)"
              >
                <div class="model-header">
                  <span class="model-name">{{ model.name }}</span>
                  <span v-if="model.recommended && model.available" class="recommended-badge">{{ t('common.recommended') }}</span>
                  <span v-if="!model.available" class="unavailable-badge">{{ model.unavailableReason || t('common.notAvailable') }}</span>
                </div>
                <div class="model-details">
                  <span class="model-size">{{ (model.sizeMB / 1024).toFixed(1) }} GB</span>
                  <span class="model-vram">VRAM: {{ model.minVramGB }} GB</span>
                  <span class="model-ram">RAM: {{ model.minRamGB }} GB</span>
                </div>
                <div class="model-desc">{{ model.description }}</div>
              </div>

              <!-- Keine Vision Option -->
              <div
                class="model-card no-vision"
                :class="{ selected: !visionEnabled && !selectedVisionModel }"
                @click="skipVision"
              >
                <div class="model-header">
                  <span class="model-name">⏭️ {{ t('setup.vision.skipOption.title') }}</span>
                </div>
                <div class="model-desc">{{ t('setup.vision.skipOption.description') }}</div>
              </div>
            </div>
          </div>

          <!-- Loading -->
          <div v-else class="loading-spinner">
            <div class="spinner"></div>
            <span>{{ t('setup.vision.loading') }}</span>
          </div>

          <!-- Download Modal für Vision -->
          <ModelDownloadModal
            :isVisible="isDownloading && currentStep === 4"
            :currentModel="selectedVisionModelName"
            :progress="Math.round(downloadProgress.percent || 0)"
            :downloadedSize="formatBytes(downloadProgress.bytesDone)"
            :totalSize="formatBytes(downloadProgress.bytesTotal)"
            :speed="downloadProgress.speedMBps ? downloadProgress.speedMBps.toFixed(1) : '0.0'"
            :statusMessages="downloadStatusMessages"
            @cancel="cancelDownload"
          />
        </div>

        <!-- Step 5: Voice-Features -->
        <div v-if="currentStep === 5" class="step-content voice-step">
          <h2>🎤 {{ t('setup.voice.title') }}</h2>
          <p>{{ t('setup.voice.subtitle') }}</p>

          <!-- Loading -->
          <div v-if="!voiceOptions" class="loading-spinner">
            <div class="spinner"></div>
            <span>{{ t('setup.voice.loading') }}</span>
          </div>

          <!-- Voice Options geladen -->
          <div v-else>
            <!-- Plattform-Hinweis -->
            <div v-if="voiceOptions.platformNote" class="info-box info" style="margin-bottom: 20px;">
              <span class="icon">💡</span>
              <span>{{ voiceOptions.platformNote }}</span>
            </div>

            <!-- Voice Toggle -->
            <div class="voice-toggle">
              <label class="toggle-switch">
                <input type="checkbox" v-model="voiceEnabled" :disabled="!voiceOptions.whisperAvailable && !voiceOptions.piperAvailable">
                <span class="slider"></span>
              </label>
              <span class="toggle-label">{{ t('setup.voice.enableVoice') }}</span>
            </div>

            <!-- Voice Optionen (wenn aktiviert) -->
            <div v-if="voiceEnabled" class="voice-options">
              <!-- Whisper (STT) -->
              <div class="voice-section" :class="{ disabled: !voiceOptions.whisperAvailable }">
                <h3>🎙️ {{ t('setup.voice.whisper.title') }}</h3>
                <p class="voice-section-desc">{{ t('setup.voice.whisper.description') }}</p>
                <div v-if="voiceOptions.whisperAvailable" class="voice-models">
                  <div
                    v-for="model in voiceOptions.whisperModels"
                    :key="model.id"
                    class="voice-model-card"
                    :class="{ selected: selectedWhisperModel === model.id }"
                    @click="selectedWhisperModel = model.id"
                  >
                    <span class="model-name">{{ model.name }}</span>
                    <span class="model-size">{{ model.sizeMB }} MB</span>
                    <span class="model-desc">{{ model.description }}</span>
                  </div>
                </div>
                <p v-else class="not-available">{{ t('setup.voice.whisper.notAvailable') }}</p>
              </div>

              <!-- Piper (TTS) -->
              <div class="voice-section" :class="{ disabled: !voiceOptions.piperAvailable }">
                <h3>🔊 {{ t('setup.voice.piper.title') }}</h3>
                <p class="voice-section-desc">{{ t('setup.voice.piper.description') }}</p>
                <div v-if="voiceOptions.piperAvailable" class="voice-models">
                  <div
                    v-for="voice in voiceOptions.piperVoices"
                    :key="voice.id"
                    class="voice-model-card"
                    :class="{ selected: selectedPiperVoice === voice.id }"
                    @click="selectedPiperVoice = voice.id"
                  >
                    <span class="model-name">{{ voice.name }}</span>
                    <span class="model-size">{{ voice.sizeMB }} MB</span>
                    <span class="model-desc">{{ voice.description }}</span>
                  </div>
                </div>
                <p v-else class="not-available">{{ t('setup.voice.piper.notAvailable') }}</p>
              </div>
            </div>

            <!-- Überspringen Option -->
            <div v-if="!voiceEnabled" class="skip-voice-info">
              <p>{{ t('setup.voice.skipInfo') }}</p>
            </div>
          </div>

          <!-- Download Modal für Voice -->
          <ModelDownloadModal
            :isVisible="isDownloading && currentStep === 5"
            :currentModel="currentVoiceDownload"
            :progress="Math.round(downloadProgress.percent || 0)"
            :downloadedSize="formatBytes(downloadProgress.bytesDone)"
            :totalSize="formatBytes(downloadProgress.bytesTotal)"
            :speed="downloadProgress.speedMBps ? downloadProgress.speedMBps.toFixed(1) : '0.0'"
            :statusMessages="downloadStatusMessages"
            @cancel="cancelDownload"
          />
        </div>

        <!-- Step 6: Fertig -->
        <div v-if="currentStep === 6" class="step-content complete-step">
          <div class="complete-icon">{{ t('setup.complete.icon') }}</div>
          <h2>{{ t('setup.complete.title') }}</h2>
          <p>{{ t('setup.complete.subtitle') }}</p>

          <div class="summary-card">
            <h3>{{ t('setup.complete.summary.title') }}</h3>
            <div class="summary-row">
              <span class="label">{{ t('setup.complete.summary.model') }}:</span>
              <span class="value">{{ selectedModelName || t('setup.complete.summary.modelNone') }}</span>
            </div>
            <div class="summary-row">
              <span class="label">{{ t('setup.complete.summary.vision') }}:</span>
              <span class="value">
                <template v-if="selectedVisionModel">{{ selectedVisionModelName }}</template>
                <template v-else>{{ t('setup.complete.summary.visionNone') }}</template>
              </span>
            </div>
            <div class="summary-row">
              <span class="label">{{ t('setup.complete.summary.voice') }}:</span>
              <span class="value">{{ voiceEnabled ? t('setup.complete.enabled') : t('setup.complete.comingSoon') }}</span>
            </div>
          </div>

          <div class="next-steps">
            <h3>{{ t('setup.complete.nextSteps.title') }}</h3>
            <ul>
              <li v-if="!selectedModel">{{ t('setup.complete.nextSteps.downloadModel') }}</li>
              <li>{{ t('setup.complete.nextSteps.startChat') }}</li>
              <li>{{ t('setup.complete.nextSteps.createExperts') }}</li>
              <li>{{ t('setup.complete.nextSteps.connectMates') }}</li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Footer mit Buttons -->
      <div class="wizard-footer">
        <button
          v-if="currentStep > 0 && currentStep < 6"
          class="btn btn-secondary"
          @click="prevStep"
          :disabled="isDownloading"
        >
          ← {{ t('common.back') }}
        </button>

        <button
          v-if="currentStep === 0"
          class="btn btn-text"
          @click="skipSetup"
        >
          {{ t('common.skip') }}
        </button>

        <div class="spacer"></div>

        <button
          v-if="currentStep < 6"
          class="btn btn-primary"
          @click="nextStep"
          :disabled="isLoading || isDownloading || (currentStep === 0 && !selectedLocale)"
        >
          {{ getNextButtonText() }}
        </button>

        <button
          v-if="currentStep === 6"
          class="btn btn-primary btn-large"
          @click="finishSetup"
        >
          🚀 {{ t('setup.complete.finishButton') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { setLocale } from '../i18n'
import ModelDownloadModal from './ModelDownloadModal.vue'

const { t, locale } = useI18n()
const emit = defineEmits(['complete'])

// Ausgewählte Sprache (für visuelle Auswahl)
const selectedLocale = ref(locale.value)

// Steps - dynamisch basierend auf Sprache
const steps = computed(() => [
  t('setup.steps.language'),
  t('setup.steps.welcome'),
  t('setup.steps.system'),
  t('setup.steps.model'),
  t('setup.steps.vision'),
  t('setup.steps.voice'),
  t('setup.steps.complete')
])
const currentStep = ref(0)
const isLoading = ref(false)
const isDownloading = ref(false)

// System Info
const systemInfo = ref(null)

// Model Selection
const modelRecommendations = ref([])
const selectedModel = ref('')
const downloadProgress = ref({
  message: '',
  percent: 0,
  bytesTotal: 0,
  bytesDone: 0,
  speedMBps: 0
})
const downloadStatusMessages = ref([])
let downloadEventSource = null

// Computed: Name des ausgewählten Modells
const selectedModelName = computed(() => {
  const model = modelRecommendations.value.find(m => m.modelId === selectedModel.value)
  return model ? model.modelName : selectedModel.value
})

// Computed: Name des ausgewählten Vision-Modells
const selectedVisionModelName = computed(() => {
  if (!visionOptions.value?.models) return selectedVisionModel.value
  const model = visionOptions.value.models.find(m => m.id === selectedVisionModel.value)
  return model ? model.name : selectedVisionModel.value
})

// Computed: Name des aktuellen Voice-Downloads
const currentVoiceDownload = computed(() => {
  if (!voiceOptions.value) return ''
  // Während Download: zeige was gerade heruntergeladen wird
  if (downloadingWhisper.value) {
    const model = voiceOptions.value.whisperModels?.find(m => m.id === selectedWhisperModel.value)
    return model ? `Whisper ${model.name}` : 'Whisper'
  }
  if (downloadingPiper.value) {
    const voice = voiceOptions.value.piperVoices?.find(v => v.id === selectedPiperVoice.value)
    return voice ? `Piper ${voice.name}` : 'Piper'
  }
  return ''
})

// Download-Status für Voice-Komponenten
const downloadingWhisper = ref(false)
const downloadingPiper = ref(false)

// Voice Options
const voiceOptions = ref(null)
const voiceEnabled = ref(false)
const selectedWhisperModel = ref('base')
const selectedPiperVoice = ref('de_DE-thorsten-medium')

// Vision Options (Dokumentenerkennung)
const visionOptions = ref(null)
const visionEnabled = ref(true) // Standard: aktiviert
const selectedVisionModel = ref('')

// API Base URL
const API_BASE = ''

// Lifecycle
onMounted(async () => {
  // Prüfe ob Setup bereits abgeschlossen
  try {
    const resp = await fetch(`${API_BASE}/api/setup/status`)
    const data = await resp.json()
    if (!data.isFirstRun) {
      emit('complete')
    }
  } catch (e) {
    console.error('Setup Status Fehler:', e)
  }
})

// Sprache auswählen
function selectLanguage(lang) {
  selectedLocale.value = lang
  setLocale(lang)
}

// Navigation
async function nextStep() {
  // Step 0: Sprache → Step 1: Willkommen
  if (currentStep.value === 0) {
    // Sprache wurde gewählt, weiter zu Willkommen
    currentStep.value++
    return
  }

  // Step 1: Willkommen → Step 2: System
  if (currentStep.value === 1) {
    currentStep.value++
    await loadSystemInfo()
    return
  }

  // Step 2: System → Step 3: Modell
  if (currentStep.value === 2) {
    await loadModelRecommendations()
  }

  // Step 3: Modell → Step 4: Vision
  if (currentStep.value === 3) {
    if (selectedModel.value) {
      await downloadModel()
    }
    await loadVisionOptions()
  }

  // Step 4: Vision → Step 5: Voice
  if (currentStep.value === 4) {
    await saveVisionSettings()
    await loadVoiceOptions()
  }

  // Step 5: Voice → Step 6: Complete
  if (currentStep.value === 5) {
    await saveVoiceSettings()
  }

  currentStep.value++
}

function prevStep() {
  if (currentStep.value > 0) {
    currentStep.value--
  }
}

function getNextButtonText() {
  switch (currentStep.value) {
    case 0: return t('common.next') + ' →'  // Sprache gewählt
    case 1: return t('setup.welcome.startButton') + ' →'
    case 2: return t('common.next') + ' →'
    case 3: return selectedModel.value ? t('setup.model.downloadButton') : t('setup.model.skipButton') + ' →'
    case 4: return visionEnabled.value && selectedVisionModel.value ? t('setup.vision.installButton') + ' →' : t('common.next') + ' →'
    case 5: return t('common.next') + ' →' // Voice (Coming Soon)
    default: return t('common.next') + ' →'
  }
}

// API Calls
async function loadSystemInfo() {
  isLoading.value = true
  try {
    const resp = await fetch(`${API_BASE}/api/setup/system-info`)
    systemInfo.value = await resp.json()
  } catch (e) {
    console.error('System Info Fehler:', e)
    // Fallback
    systemInfo.value = {
      os: 'unknown',
      arch: 'unknown',
      cpuCores: 4,
      totalRamGB: 8,
      availableRamGB: 4,
      hasGpu: false
    }
  }
  isLoading.value = false
}

async function loadModelRecommendations() {
  isLoading.value = true
  try {
    const resp = await fetch(`${API_BASE}/api/setup/model-recommendations`)
    modelRecommendations.value = await resp.json()

    // Auto-Select empfohlenes Modell (nur wenn verfügbar)
    const recommended = modelRecommendations.value.find(m => m.recommended && m.available)
    if (recommended) {
      selectedModel.value = recommended.modelId
    } else {
      // Fallback: Erstes verfügbares Modell auswählen
      const firstAvailable = modelRecommendations.value.find(m => m.available)
      if (firstAvailable) {
        selectedModel.value = firstAvailable.modelId
      }
    }
  } catch (e) {
    console.error('Model Recommendations Fehler:', e)
  }
  isLoading.value = false
}

async function loadVoiceOptions() {
  try {
    const resp = await fetch(`${API_BASE}/api/setup/voice-options`)
    voiceOptions.value = await resp.json()
  } catch (e) {
    console.error('Voice Options Fehler:', e)
  }
}

// Vision API Calls
async function loadVisionOptions() {
  try {
    const resp = await fetch(`${API_BASE}/api/setup/vision-options`)
    visionOptions.value = await resp.json()

    // Auto-Select empfohlenes Vision-Modell (nur wenn verfügbar)
    if (visionOptions.value?.models) {
      const recommended = visionOptions.value.models.find(m => m.recommended && m.available)
      if (recommended) {
        selectedVisionModel.value = recommended.id
        visionEnabled.value = true
      } else {
        // Kein verfügbares Modell
        const anyAvailable = visionOptions.value.models.find(m => m.available)
        if (!anyAvailable) {
          visionEnabled.value = false
        }
      }
    }
  } catch (e) {
    console.error('Vision Options Fehler:', e)
  }
}

async function saveVisionSettings() {
  // Wenn kein Vision-Modell gewählt (überspringen)
  if (!visionEnabled.value || !selectedVisionModel.value) {
    try {
      await fetch(`${API_BASE}/api/setup/vision-settings`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: false,
          modelId: ''
        })
      })
    } catch (e) {
      console.error('Vision Settings Fehler:', e)
    }
    return
  }

  // Vision-Modell downloaden via SSE
  await downloadVisionModel()
}

async function downloadVisionModel() {
  if (!selectedVisionModel.value) return

  isDownloading.value = true
  downloadProgress.value = { message: t('setup.model.startingVisionDownload'), percent: 0 }
  downloadStatusMessages.value = [t('setup.model.preparingVision')]

  // Promise das auf Download-Ende wartet
  return new Promise((resolve, reject) => {
    try {
      // SSE für Progress
      const eventSource = new EventSource(
        `${API_BASE}/api/setup/download-vision?modelId=${encodeURIComponent(selectedVisionModel.value)}&lang=${selectedLocale.value}`
      )

      eventSource.onmessage = (event) => {
        try {
          const progress = JSON.parse(event.data)

          // Progress aktualisieren
          if (progress.percent !== undefined) {
            downloadProgress.value.percent = progress.percent
          }
          if (progress.message) {
            downloadProgress.value.message = progress.message
            // Nur wichtige Nachrichten hinzufügen
            if (!progress.message.includes('MB/s')) {
              downloadStatusMessages.value.push(progress.message)
            }
          }
          if (progress.speedMBps) {
            downloadProgress.value.speedMBps = progress.speedMBps
          }
          if (progress.bytesTotal) {
            downloadProgress.value.bytesTotal = progress.bytesTotal
          }
          if (progress.bytesDone) {
            downloadProgress.value.bytesDone = progress.bytesDone
          }

          // Fertig
          if (progress.done) {
            eventSource.close()
            downloadStatusMessages.value.push('✅ Vision-Modell installiert!')
            isDownloading.value = false
            resolve()
          }
          // Fehler
          if (progress.error) {
            eventSource.close()
            downloadStatusMessages.value.push('❌ Fehler: ' + progress.error)
            isDownloading.value = false
            reject(new Error(progress.error))
          }
        } catch (e) {
          console.error('Progress Parse Fehler:', e)
        }
      }

      eventSource.onerror = (e) => {
        console.error('Vision Download SSE Fehler:', e)
        eventSource.close()
        downloadStatusMessages.value.push('❌ Verbindungsfehler')
        isDownloading.value = false
        reject(new Error('Verbindungsfehler'))
      }

    } catch (e) {
      console.error('Vision Download Fehler:', e)
      downloadStatusMessages.value.push('❌ Fehler: ' + e.message)
      isDownloading.value = false
      reject(e)
    }
  })
}

function selectVisionModel(model) {
  if (!model.available) return
  selectedVisionModel.value = model.id
  visionEnabled.value = true
}

function skipVision() {
  visionEnabled.value = false
  selectedVisionModel.value = ''
}

function selectModel(model) {
  // Nur auswählbar wenn verfügbar
  if (!model.available) {
    return
  }
  selectedModel.value = model.modelId
}

async function downloadModel() {
  console.log('[Setup] downloadModel() called, selectedModel:', selectedModel.value)
  if (!selectedModel.value) return

  // Sicherheitsprüfung: Nur verfügbare Modelle herunterladen
  const model = modelRecommendations.value.find(m => m.modelId === selectedModel.value)
  if (!model || !model.available) {
    console.error('[Setup] Download blockiert: Modell nicht verfügbar', selectedModel.value)
    return
  }

  isDownloading.value = true
  downloadProgress.value = { message: 'Starte Setup...', percent: 0, bytesTotal: 0, bytesDone: 0, speedMBps: 0 }
  downloadStatusMessages.value = ['Setup wird vorbereitet...']

  try {
    // Schritt 0: Modell-Auswahl ans Backend senden (WICHTIG für HandleComplete!)
    console.log('[Setup] Step 0: Calling select-model')
    await fetch(`${API_BASE}/api/setup/select-model`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ modelId: selectedModel.value })
    })

    // Schritt 1: Verzeichnisse erstellen
    console.log('[Setup] Step 1: Calling create-directories')
    downloadStatusMessages.value.push('📁 Erstelle Verzeichnisstruktur...')
    downloadProgress.value = { ...downloadProgress.value, message: 'Erstelle Verzeichnisse...', percent: 2 }

    const dirResp = await fetch(`${API_BASE}/api/setup/create-directories`, { method: 'POST' })
    console.log('[Setup] create-directories response:', dirResp.status, dirResp.ok)
    if (!dirResp.ok) throw new Error('Verzeichnisse konnten nicht erstellt werden')
    downloadStatusMessages.value.push('✅ Verzeichnisse erstellt')

    // Schritt 2: llama-server herunterladen (falls nicht vorhanden)
    console.log('[Setup] Step 2: Calling llama-server-status')
    downloadStatusMessages.value.push('🔍 Prüfe KI-Engine...')
    downloadProgress.value = { ...downloadProgress.value, message: 'Prüfe KI-Engine...', percent: 5 }

    const statusResp = await fetch(`${API_BASE}/api/setup/llama-server-status`)
    const serverStatus = await statusResp.json()
    console.log('[Setup] llama-server-status response:', serverStatus)
    console.log('[Setup] installed =', serverStatus.installed, 'type:', typeof serverStatus.installed)

    if (!serverStatus.installed) {
      console.log('[Setup] Step 3: llama-server NOT installed, starting download...')
      downloadStatusMessages.value.push('⬇️ ' + t('setup.model.downloadingEngine'))

      await new Promise((resolve, reject) => {
        console.log('[Setup] Opening EventSource for llama-server download...')
        const serverSource = new EventSource(`${API_BASE}/api/setup/download-llama-server?lang=${selectedLocale.value}`)
        console.log('[Setup] EventSource created, waiting for messages...')

        serverSource.onmessage = (event) => {
          const data = JSON.parse(event.data)
          // Progress anpassen (5-40% für llama-server)
          const adjustedPercent = 5 + (data.percent * 0.35)
          downloadProgress.value = { ...data, percent: adjustedPercent }

          if (data.message && !downloadStatusMessages.value.includes(data.message)) {
            downloadStatusMessages.value.push(data.message)
            if (downloadStatusMessages.value.length > 6) downloadStatusMessages.value.shift()
          }

          if (data.done) {
            serverSource.close()
            if (data.error) {
              reject(new Error(data.error))
            } else {
              downloadStatusMessages.value.push('✅ KI-Engine installiert')
              resolve()
            }
          }
        }

        serverSource.onerror = (err) => {
          console.error('[Setup] EventSource error:', err)
          serverSource.close()
          reject(new Error('KI-Engine Download fehlgeschlagen'))
        }
      })
    } else {
      console.log('[Setup] llama-server ALREADY installed, skipping download')
      downloadStatusMessages.value.push('✅ KI-Engine bereits installiert')
      downloadProgress.value = { ...downloadProgress.value, percent: 40 }
    }

    // Schritt 3: KI-Modell herunterladen
    console.log('[Setup] Step 4: Starting model download')
    downloadStatusMessages.value.push('⬇️ ' + t('setup.model.downloadingModel'))
    downloadProgress.value = { ...downloadProgress.value, message: t('setup.model.downloadingModel'), percent: 42 }

    await new Promise((resolve, reject) => {
      downloadEventSource = new EventSource(`${API_BASE}/api/setup/download-model?modelId=${encodeURIComponent(selectedModel.value)}&lang=${selectedLocale.value}`)

      downloadEventSource.onmessage = (event) => {
        const data = JSON.parse(event.data)
        // Progress anpassen (42-100% für Modell)
        const adjustedPercent = 42 + (data.percent * 0.58)
        downloadProgress.value = { ...data, percent: adjustedPercent }

        if (data.message && !downloadStatusMessages.value.includes(data.message)) {
          downloadStatusMessages.value.push(data.message)
          if (downloadStatusMessages.value.length > 6) downloadStatusMessages.value.shift()
        }

        if (data.done) {
          downloadEventSource.close()
          downloadEventSource = null
          if (data.error) {
            downloadStatusMessages.value.push('❌ Fehler: ' + data.error)
            reject(new Error(data.error))
          } else {
            downloadStatusMessages.value.push('✅ Modell heruntergeladen!')
            resolve()
          }
        }
      }

      downloadEventSource.onerror = () => {
        downloadEventSource.close()
        downloadEventSource = null
        downloadStatusMessages.value.push('❌ Verbindung verloren')
        reject(new Error('Download-Verbindung verloren'))
      }
    })

    downloadStatusMessages.value.push('🎉 Setup abgeschlossen!')
    downloadProgress.value = { ...downloadProgress.value, message: 'Setup abgeschlossen!', percent: 100 }

  } catch (e) {
    console.error('Setup Fehler:', e)
    alert('Setup fehlgeschlagen: ' + e.message)
  }

  isDownloading.value = false
}

function cancelDownload() {
  if (downloadEventSource) {
    downloadEventSource.close()
    downloadEventSource = null
  }
  isDownloading.value = false
  downloadStatusMessages.value.push('⚠️ Download abgebrochen')
}

async function saveVoiceSettings() {
  // Wenn Voice deaktiviert, nur speichern
  if (!voiceEnabled.value) {
    try {
      await fetch(`${API_BASE}/api/setup/select-voice`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          enabled: false,
          whisperModel: '',
          piperVoice: ''
        })
      })
    } catch (e) {
      console.error('Voice Settings Fehler:', e)
    }
    return
  }

  // Voice aktiviert - Downloads starten
  isDownloading.value = true
  downloadProgress.value = { message: t('setup.voice.startingDownload'), percent: 0 }
  downloadStatusMessages.value = [t('setup.voice.preparingDownload')]

  try {
    // Whisper downloaden (wenn verfügbar und ausgewählt)
    if (voiceOptions.value?.whisperAvailable && selectedWhisperModel.value) {
      downloadingWhisper.value = true
      await downloadVoiceComponent('whisper', selectedWhisperModel.value)
      downloadingWhisper.value = false
    }

    // Piper downloaden (wenn verfügbar und ausgewählt)
    if (voiceOptions.value?.piperAvailable && selectedPiperVoice.value) {
      downloadingPiper.value = true
      await downloadVoiceComponent('piper', selectedPiperVoice.value)
      downloadingPiper.value = false
    }

    // Settings speichern
    await fetch(`${API_BASE}/api/setup/select-voice`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        enabled: voiceEnabled.value,
        whisperModel: selectedWhisperModel.value,
        piperVoice: selectedPiperVoice.value
      })
    })

    downloadStatusMessages.value.push('✅ ' + t('setup.voice.downloadComplete'))
  } catch (e) {
    console.error('Voice Setup Fehler:', e)
    downloadStatusMessages.value.push('❌ ' + t('common.error') + ': ' + e.message)
  }

  isDownloading.value = false
  downloadingWhisper.value = false
  downloadingPiper.value = false
}

// Voice-Komponente (Whisper oder Piper) downloaden
async function downloadVoiceComponent(component, modelId) {
  return new Promise((resolve, reject) => {
    const eventSource = new EventSource(
      `${API_BASE}/api/setup/download-voice?component=${component}&modelId=${encodeURIComponent(modelId)}&lang=${selectedLocale.value}`
    )

    eventSource.onmessage = (event) => {
      try {
        const progress = JSON.parse(event.data)

        if (progress.percent !== undefined) {
          downloadProgress.value.percent = progress.percent
        }
        if (progress.message) {
          downloadProgress.value.message = progress.message
          if (!progress.message.includes('MB/s') && !downloadStatusMessages.value.includes(progress.message)) {
            downloadStatusMessages.value.push(progress.message)
            if (downloadStatusMessages.value.length > 6) downloadStatusMessages.value.shift()
          }
        }
        if (progress.speedMBps) {
          downloadProgress.value.speedMBps = progress.speedMBps
        }
        if (progress.bytesTotal) {
          downloadProgress.value.bytesTotal = progress.bytesTotal
        }
        if (progress.bytesDone) {
          downloadProgress.value.bytesDone = progress.bytesDone
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
        console.error('Voice Progress Parse Fehler:', e)
      }
    }

    eventSource.onerror = (e) => {
      console.error('Voice Download SSE Fehler:', e)
      eventSource.close()
      reject(new Error(t('common.connectionError')))
    }
  })
}

async function skipSetup() {
  if (confirm('Möchtest du das Setup wirklich überspringen? Du kannst Modelle später im Model Manager herunterladen.')) {
    try {
      await fetch(`${API_BASE}/api/setup/skip`, { method: 'POST' })
      emit('complete')
    } catch (e) {
      console.error('Skip Fehler:', e)
    }
  }
}

async function finishSetup() {
  isDownloading.value = true
  downloadProgress.value = { message: 'Schließe Setup ab...', percent: 0 }
  downloadStatusMessages.value = ['🚀 Finalisiere Setup...']

  try {
    // Schritt 1: Setup abschließen (Dateien schreiben, etc.)
    downloadStatusMessages.value.push('📝 Speichere Einstellungen...')
    const resp = await fetch(`${API_BASE}/api/setup/complete`, { method: 'POST' })
    const data = await resp.json()

    if (!data.success) {
      throw new Error(data.error || 'Setup fehlgeschlagen')
    }

    downloadStatusMessages.value.push('✅ Einstellungen gespeichert')
    downloadProgress.value = { message: 'Setup abgeschlossen!', percent: 100 }
    downloadStatusMessages.value.push('🎉 Setup erfolgreich abgeschlossen!')

    // Schritt 2: llama-server im HINTERGRUND starten (nicht blockierend!)
    // Der Server startet asynchron, während der User schon die Hauptansicht sieht
    if (selectedModel.value) {
      console.log('[Setup] Starte llama-server im Hintergrund...')
      // Fire-and-forget: Server-Start triggern ohne zu warten
      fetch(`${API_BASE}/api/setup/start-llama-server-async`, { method: 'POST' })
        .then(() => console.log('[Setup] llama-server Start angefordert'))
        .catch(e => console.warn('[Setup] llama-server Start-Request fehlgeschlagen:', e))
    }

    // Kurz warten damit der User die Erfolgsmeldung sieht
    await new Promise(r => setTimeout(r, 800))

    isDownloading.value = false
    emit('complete')

    // Seite neu laden damit alle Settings (inkl. neues Modell) frisch geladen werden
    // Der llama-server startet im Hintergrund weiter
    window.location.reload()
  } catch (e) {
    console.error('Finish Fehler:', e)
    downloadStatusMessages.value.push('❌ Fehler: ' + e.message)
    isDownloading.value = false
  }
}

// Helpers
function formatOS(os) {
  const osNames = {
    'linux': 'Linux',
    'darwin': 'macOS',
    'windows': 'Windows'
  }
  return osNames[os] || os
}

function formatBytes(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024
    i++
  }
  return `${bytes.toFixed(1)} ${units[i]}`
}
</script>

<style scoped>
.setup-wizard-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.setup-wizard {
  background: #1e1e2e;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  width: 90%;
  max-width: 700px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Header */
.wizard-header {
  background: linear-gradient(135deg, #2d2d44 0%, #1e1e2e 100%);
  padding: 20px 30px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #333;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  font-size: 28px;
}

.logo-text {
  font-size: 20px;
  font-weight: 600;
  color: #fff;
}

.steps-indicator {
  display: flex;
  gap: 8px;
}

.step-dot {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #333;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.3s;
}

.step-dot.active {
  background: #4f46e5;
  transform: scale(1.1);
}

.step-dot.completed {
  background: #10b981;
}

.step-number {
  color: #888;
  font-size: 12px;
  font-weight: 600;
}

.step-dot.active .step-number,
.step-dot.completed .step-number {
  color: #fff;
}

/* Content */
.wizard-content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
}

.step-content {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Language Step */
.language-step {
  text-align: center;
}

.language-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.language-step h1 {
  color: #fff;
  font-size: 24px;
  margin-bottom: 10px;
}

.language-grid {
  display: flex;
  justify-content: center;
  gap: 20px;
  margin-top: 40px;
}

.language-card {
  background: #2d2d44;
  border: 3px solid transparent;
  border-radius: 16px;
  padding: 30px 40px;
  cursor: pointer;
  transition: all 0.3s;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 15px;
  min-width: 140px;
}

.language-card:hover {
  border-color: #4f46e5;
  transform: translateY(-5px);
}

.language-card.selected {
  border-color: #10b981;
  background: #1e3a5f;
  transform: translateY(-5px);
}

.language-flag {
  font-size: 48px;
}

.language-name {
  color: #fff;
  font-size: 18px;
  font-weight: 600;
}

/* Welcome Step */
.welcome-step {
  text-align: center;
}

.welcome-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.welcome-step h1 {
  color: #fff;
  font-size: 28px;
  margin-bottom: 10px;
}

.subtitle {
  color: #888;
  font-size: 16px;
  margin-bottom: 30px;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 15px;
  margin-bottom: 30px;
}

.feature {
  background: #2d2d44;
  padding: 15px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.feature-icon {
  font-size: 24px;
}

.feature-text {
  color: #ccc;
  font-size: 14px;
}

.setup-info {
  color: #666;
  font-size: 14px;
}

/* System Step */
.system-info-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  margin-top: 20px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 12px 0;
  border-bottom: 1px solid #3d3d5c;
}

.info-row:last-child {
  border-bottom: none;
}

.info-row .label {
  color: #888;
}

.info-row .value {
  color: #fff;
  font-weight: 500;
}

.info-row.has-gpu .value {
  color: #10b981;
}

/* Info Box */
.info-box {
  background: #2d3748;
  border-left: 4px solid #4f46e5;
  padding: 15px;
  border-radius: 8px;
  margin-top: 20px;
  display: flex;
  align-items: center;
  gap: 10px;
  color: #ccc;
}

.info-box.warning {
  border-left-color: #f59e0b;
  background: #422006;
}

.info-box.info {
  border-left-color: #3b82f6;
  background: #1e3a5f;
}

.info-box.success {
  border-left-color: #10b981;
  background: #064e3b;
}

/* CPU Info Card */
.cpu-info-card {
  background: #1e3a5f;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  border-left: 4px solid #3b82f6;
}

.cpu-info-card h3 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 15px;
}

.cpu-comparison {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 12px;
}

.cpu-comparison th,
.cpu-comparison td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #2d4a6f;
}

.cpu-comparison th {
  color: #94a3b8;
  font-weight: 500;
  font-size: 13px;
}

.cpu-comparison td {
  color: #e2e8f0;
  font-size: 14px;
}

.cpu-comparison .recommended-row {
  background: rgba(59, 130, 246, 0.2);
}

.cpu-comparison .recommended-row td {
  color: #60a5fa;
  font-weight: 500;
}

.cpu-hint {
  color: #94a3b8;
  font-size: 12px;
  margin: 0;
  font-style: italic;
}

/* GPU Info Card */
.gpu-info-card {
  background: #064e3b;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  border-left: 4px solid #10b981;
}

.gpu-info-card h3 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 15px;
}

.gpu-comparison {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 12px;
}

.gpu-comparison th,
.gpu-comparison td {
  padding: 10px 12px;
  text-align: left;
  border-bottom: 1px solid #0d7a5f;
}

.gpu-comparison th {
  color: #94a3b8;
  font-weight: 500;
  font-size: 13px;
}

.gpu-comparison td {
  color: #e2e8f0;
  font-size: 14px;
}

.gpu-comparison .recommended-row {
  background: rgba(16, 185, 129, 0.2);
}

.gpu-comparison .recommended-row td {
  color: #34d399;
  font-weight: 500;
}

.gpu-hint {
  color: #94a3b8;
  font-size: 12px;
  margin: 0;
  font-style: italic;
}

/* Model Step */
.models-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 20px;
}

.model-card {
  background: #2d2d44;
  border: 2px solid transparent;
  border-radius: 12px;
  padding: 15px;
  cursor: pointer;
  transition: all 0.2s;
}

.model-card:hover {
  border-color: #4f46e5;
}

.model-card.selected {
  border-color: #4f46e5;
  background: #3d3d5c;
}

.model-card.recommended {
  border-color: #10b981;
}

.model-card.disabled {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none; /* Verhindert Klicks komplett */
  background: #1e1e2e;
  border-color: #333;
}

.model-card.disabled:hover {
  border-color: #333;
  transform: none;
}

.model-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.model-name {
  color: #fff;
  font-weight: 600;
  font-size: 16px;
}

.recommended-badge {
  background: #10b981;
  color: #fff;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
}

.unavailable-badge {
  background: #6b7280;
  color: #fff;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
}

.model-details {
  display: flex;
  gap: 15px;
  color: #888;
  font-size: 14px;
}

.model-reason {
  margin-top: 8px;
  color: #10b981;
  font-size: 13px;
}

.model-reason.reason-warning {
  color: #f59e0b;
}

/* GPU Info Card */
.gpu-info-card {
  background: #064e3b;
  border-radius: 12px;
  padding: 20px;
  margin-bottom: 20px;
  border-left: 4px solid #10b981;
}

.gpu-info-card h3 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 8px;
}

.gpu-hint {
  color: #a7f3d0;
  font-size: 14px;
  margin: 0;
}

/* Download Progress */
.download-progress {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  margin-top: 20px;
}

.progress-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  color: #fff;
}

.progress-bar {
  height: 8px;
  background: #1e1e2e;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #4f46e5, #7c3aed);
  border-radius: 4px;
  transition: width 0.3s;
}

.progress-details {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  color: #888;
  font-size: 13px;
}

/* Vision Step */
.vision-step h2 {
  color: #fff;
  font-size: 22px;
  margin-bottom: 10px;
}

.vision-step > p {
  color: #888;
  margin-bottom: 15px;
}

.vision-step .model-card {
  margin-bottom: 10px;
}

.vision-step .model-desc {
  color: #888;
  font-size: 13px;
  margin-top: 8px;
  line-height: 1.4;
}

.vision-step .model-vram,
.vision-step .model-ram {
  color: #888;
  font-size: 13px;
}

/* No Vision Card */
.model-card.no-vision {
  background: #2d2d3a;
  border-color: #444;
}

.model-card.no-vision:hover {
  border-color: #666;
}

.model-card.no-vision.selected {
  border-color: #666;
  background: #3d3d4a;
}

/* Voice Step */
.voice-toggle {
  display: flex;
  align-items: center;
  gap: 15px;
  margin: 20px 0;
}

.toggle-switch {
  position: relative;
  width: 50px;
  height: 26px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: #333;
  border-radius: 26px;
  transition: 0.3s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 20px;
  width: 20px;
  left: 3px;
  bottom: 3px;
  background: #fff;
  border-radius: 50%;
  transition: 0.3s;
}

input:checked + .slider {
  background: #4f46e5;
}

input:checked + .slider:before {
  transform: translateX(24px);
}

.toggle-label {
  color: #fff;
  font-size: 16px;
}

.voice-options {
  margin-top: 20px;
}

.voice-section {
  margin-bottom: 25px;
}

.voice-section.disabled {
  opacity: 0.5;
  pointer-events: none;
}

.voice-section h3 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 12px;
}

.voice-models {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
}

.voice-model-card {
  background: #2d2d44;
  border: 2px solid transparent;
  border-radius: 10px;
  padding: 12px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.voice-model-card:hover {
  border-color: #4f46e5;
}

.voice-model-card.selected {
  border-color: #4f46e5;
  background: #3d3d5c;
}

.voice-model-card .model-name {
  font-size: 14px;
}

.voice-model-card .model-size {
  color: #888;
  font-size: 12px;
}

.voice-model-card .model-desc {
  color: #666;
  font-size: 11px;
}

.not-available {
  color: #666;
  font-style: italic;
  padding: 15px;
  background: #2d2d44;
  border-radius: 10px;
}

/* Voice Section Description */
.voice-section-desc {
  color: #888;
  font-size: 13px;
  margin-bottom: 12px;
}

/* Skip Voice Info */
.skip-voice-info {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  margin-top: 20px;
  text-align: center;
}

.skip-voice-info p {
  color: #888;
  margin: 0;
}

/* Complete Step */
.complete-step {
  text-align: center;
}

.complete-icon {
  font-size: 64px;
  margin-bottom: 20px;
}

.complete-step h2 {
  color: #fff;
  font-size: 24px;
  margin-bottom: 10px;
}

.summary-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  margin: 25px 0;
  text-align: left;
}

.summary-card h3 {
  color: #fff;
  margin-bottom: 15px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid #3d3d5c;
}

.summary-row:last-child {
  border-bottom: none;
}

.summary-row .label {
  color: #888;
}

.summary-row .value {
  color: #fff;
}

.next-steps {
  text-align: left;
  background: #1e3a5f;
  border-radius: 12px;
  padding: 20px;
}

.next-steps h3 {
  color: #fff;
  margin-bottom: 15px;
}

.next-steps ul {
  color: #ccc;
  padding-left: 20px;
}

.next-steps li {
  margin-bottom: 8px;
}

/* Footer */
.wizard-footer {
  background: #2d2d44;
  padding: 20px 30px;
  display: flex;
  align-items: center;
  gap: 15px;
  border-top: 1px solid #333;
}

.spacer {
  flex: 1;
}

.btn {
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: none;
}

.btn-primary {
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  color: #fff;
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 5px 20px rgba(79, 70, 229, 0.4);
}

.btn-secondary {
  background: #333;
  color: #fff;
}

.btn-secondary:hover:not(:disabled) {
  background: #444;
}

.btn-text {
  background: transparent;
  color: #888;
}

.btn-text:hover {
  color: #fff;
}

.btn-large {
  padding: 15px 40px;
  font-size: 17px;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Loading Spinner */
.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px;
  gap: 15px;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #333;
  border-top-color: #4f46e5;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Responsive */
@media (max-width: 600px) {
  .setup-wizard {
    width: 100%;
    height: 100%;
    max-height: 100%;
    border-radius: 0;
  }

  .features-grid {
    grid-template-columns: 1fr;
  }

  .wizard-header {
    flex-direction: column;
    gap: 15px;
  }
}
</style>
