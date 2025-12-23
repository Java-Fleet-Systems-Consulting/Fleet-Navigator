<template>
  <Transition name="modal">
    <div v-if="show" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-[60] p-2">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-xl max-h-[95vh] overflow-hidden flex flex-col">
        <!-- Header mit Buttons -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20">
          <h3 class="text-lg font-bold text-gray-900 dark:text-white">
            {{ $t(isEditing ? 'createExpertModal.titleEdit' : 'createExpertModal.titleNew') }}
          </h3>
          <div class="flex items-center gap-2">
            <button
              @click="$emit('close')"
              class="px-3 py-1.5 text-sm rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
            >
              {{ $t('common.cancel') }}
            </button>
            <button
              @click="save"
              :disabled="!canSave || isSaving"
              class="px-3 py-1.5 text-sm rounded-lg bg-purple-500 hover:bg-purple-600 text-white font-medium transition-colors disabled:opacity-50"
            >
              {{ isSaving ? $t('common.saving') : $t('common.save') }}
            </button>
          </div>
        </div>

        <!-- Scrollbarer Form-Bereich -->
        <div class="flex-1 overflow-y-auto p-4 space-y-3">
          <!-- Row 1: Name + Rolle -->
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('createExpertModal.name') }} *</label>
              <input
                v-model="form.name"
                type="text"
                :placeholder="$t('createExpertModal.namePlaceholder')"
                class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('createExpertModal.role') }} *</label>
              <input
                v-model="form.role"
                type="text"
                :placeholder="$t('createExpertModal.rolePlaceholder')"
                class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              />
            </div>
          </div>

          <!-- Row 2: Avatar + Beschreibung + Benachrichtigung -->
          <div class="flex gap-3">
            <!-- Avatar kompakt -->
            <div class="flex-shrink-0">
              <div
                v-if="form.avatarUrl"
                class="w-14 h-14 rounded-lg overflow-hidden border-2 border-purple-300 dark:border-purple-600 cursor-pointer relative group"
                @click="triggerAvatarFileInput"
              >
                <img :src="form.avatarUrl" alt="Avatar" class="w-full h-full object-cover" @error="handleAvatarError" />
                <div class="absolute inset-0 bg-black/50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                  <PhotoIcon class="w-5 h-5 text-white" />
                </div>
              </div>
              <div
                v-else
                class="w-14 h-14 rounded-lg bg-gray-200 dark:bg-gray-700 flex items-center justify-center border-2 border-dashed border-gray-300 dark:border-gray-600 cursor-pointer hover:border-purple-400 transition-colors"
                @click="triggerAvatarFileInput"
              >
                <PhotoIcon class="w-6 h-6 text-gray-400" />
              </div>
              <input ref="avatarFileInput" type="file" accept="image/*" @change="handleAvatarUpload" class="hidden" />
            </div>
            <div class="flex-1 space-y-2">
              <input
                v-model="form.description"
                type="text"
                :placeholder="$t('createExpertModal.descriptionPlaceholder')"
                class="w-full px-3 py-1.5 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              />
              <label class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400 cursor-pointer">
                <input type="checkbox" v-model="form.showSwitchNotification" class="w-3.5 h-3.5 text-purple-600 rounded" />
                {{ $t('createExpertModal.switchNotification') }}
              </label>
            </div>
          </div>

          <!-- Row 2b: TTS Stimme -->
          <div>
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
              🔊 {{ $t('createExpertModal.voiceLabel') }}
              <span v-if="isTestingVoice" class="ml-2 text-purple-500 animate-pulse">▶ {{ $t('createExpertModal.testing') }}</span>
            </label>
            <div class="grid grid-cols-4 gap-1.5">
              <button
                v-for="voice in availableVoices"
                :key="voice.id"
                type="button"
                @click="form.voice = voice.id"
                @dblclick.prevent="testVoice(voice)"
                :class="[
                  'p-2 rounded-lg border text-xs transition-all text-left relative group',
                  form.voice === voice.id
                    ? 'border-purple-500 bg-purple-50 dark:bg-purple-900/30 ring-1 ring-purple-500'
                    : 'border-gray-200 dark:border-gray-700 hover:border-purple-300 dark:hover:border-purple-600'
                ]"
                :title="$t('createExpertModal.dblClickTest')"
              >
                <div class="flex items-center gap-1.5">
                  <span>{{ voice.gender === 'female' ? '👩' : '👨' }}</span>
                  <div class="min-w-0 flex-1">
                    <div class="font-medium text-gray-900 dark:text-white truncate">{{ voice.name }}</div>
                    <div class="text-[10px] text-gray-500 dark:text-gray-400 truncate">{{ voice.description }}</div>
                  </div>
                </div>
                <!-- Play button on hover -->
                <button
                  type="button"
                  @click.stop="testVoice(voice)"
                  class="absolute -top-1 -right-1 w-5 h-5 bg-purple-500 hover:bg-purple-600 text-white rounded-full opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center shadow-md"
                  :disabled="isTestingVoice"
                  :title="$t('createExpertModal.testVoice')"
                >
                  <span class="text-[10px]">▶</span>
                </button>
              </button>
            </div>
            <p class="text-[10px] text-gray-400 mt-1">{{ $t('createExpertModal.testVoiceHint') }}</p>
          </div>

          <!-- Row 3: Provider + Modell -->
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('createExpertModal.provider') }}</label>
              <select
                v-model="form.providerType"
                class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              >
                <option value="">{{ $t('createExpertModal.providerDefault') }}</option>
                <option value="llama-server">llama-server</option>
              </select>
            </div>
            <div>
              <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">
                {{ $t('createExpertModal.model') }} * <span class="text-gray-400">({{ availableModels?.length || 0 }})</span>
              </label>
              <select
                v-model="form.baseModel"
                class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
              >
                <option value="" disabled>{{ $t('createExpertModal.selectModel') }}</option>
                <option v-for="model in availableModels" :key="model.name" :value="model.name">
                  {{ model.displayName || model.name }}
                </option>
              </select>
            </div>
          </div>

          <!-- GGUF Model (conditional) -->
          <div v-if="form.providerType === 'llama-server' || form.providerType === 'java-llama-cpp'">
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('createExpertModal.ggufModel') }}</label>
            <input
              v-model="form.ggufModel"
              type="text"
              :placeholder="$t('createExpertModal.ggufPlaceholder')"
              class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
          </div>

          <!-- Base Prompt -->
          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ $t('createExpertModal.basePrompt') }} *</label>
              <button
                @click="triggerBasePromptFileInput"
                type="button"
                class="flex items-center gap-1 px-2 py-1 text-xs rounded bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 hover:bg-purple-200 dark:hover:bg-purple-900/50"
              >
                <DocumentArrowUpIcon class="w-3 h-3" />
                {{ $t('createExpertModal.fromFile') }}
              </button>
              <input ref="basePromptFileInput" type="file" accept=".txt,.md" @change="loadBasePromptFromFile" class="hidden" />
            </div>
            <textarea
              v-model="form.basePrompt"
              rows="4"
              :placeholder="$t('createExpertModal.basePromptPlaceholder')"
              class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none font-mono"
            ></textarea>
          </div>

          <!-- Personality Prompt -->
          <div>
            <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('createExpertModal.communicationStyle') }}</label>
            <textarea
              v-model="form.personalityPrompt"
              rows="2"
              :placeholder="$t('createExpertModal.communicationStylePlaceholder')"
              class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none font-mono"
            ></textarea>
          </div>

          <!-- Collapsible: Erweiterte Einstellungen -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-3">
            <button
              @click="showAdvanced = !showAdvanced"
              class="flex items-center gap-1 text-xs font-medium text-purple-600 dark:text-purple-400"
            >
              <ChevronDownIcon class="w-3 h-3 transition-transform" :class="{ 'rotate-180': showAdvanced }" />
              {{ $t('createExpertModal.advancedSettings') }}
            </button>

            <div v-if="showAdvanced" class="mt-3 grid grid-cols-2 gap-3">
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.temperature') }}: {{ form.defaultTemperature }}</label>
                <input v-model.number="form.defaultTemperature" type="range" min="0" max="2" step="0.1" class="w-full h-1.5" />
              </div>
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.topP') }}: {{ form.defaultTopP }}</label>
                <input v-model.number="form.defaultTopP" type="range" min="0" max="1" step="0.05" class="w-full h-1.5" />
              </div>
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.context') }}: {{ form.defaultNumCtx?.toLocaleString() }}</label>
                <input v-model.number="form.defaultNumCtx" type="range" min="2048" max="131072" step="2048" class="w-full h-1.5" />
              </div>
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.maxTokens') }}: {{ form.defaultMaxTokens?.toLocaleString() }}</label>
                <input v-model.number="form.defaultMaxTokens" type="range" min="256" max="16384" step="256" class="w-full h-1.5" />
              </div>
            </div>
          </div>

          <!-- Collapsible: RAG-Einstellungen -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-3">
            <button
              @click="showRAG = !showRAG"
              class="flex items-center gap-1 text-xs font-medium text-purple-600 dark:text-purple-400"
            >
              <ChevronDownIcon class="w-3 h-3 transition-transform" :class="{ 'rotate-180': showRAG }" />
              <GlobeAltIcon class="w-3 h-3" />
              {{ $t('createExpertModal.ragWebSearch') }}
            </button>

            <div v-if="showRAG" class="mt-3 space-y-3">
              <!-- Websuche -->
              <label class="flex items-center gap-2 text-xs cursor-pointer">
                <input type="checkbox" v-model="form.autoWebSearch" class="w-3.5 h-3.5 text-blue-600 rounded" />
                <span class="text-gray-700 dark:text-gray-300">{{ $t('createExpertModal.alwaysSearchWeb') }}</span>
              </label>

              <div v-if="form.autoWebSearch" class="ml-5 space-y-2">
                <!-- Links anzeigen Toggle -->
                <label class="flex items-center gap-2 text-xs cursor-pointer">
                  <input type="checkbox" v-model="form.webSearchShowLinks" class="w-3.5 h-3.5 text-blue-600 rounded" />
                  <span class="text-gray-700 dark:text-gray-300">{{ $t('createExpertModal.showSourceLinks') }}</span>
                </label>
                <p class="text-[10px] text-gray-500 dark:text-gray-400 ml-5">
                  {{ $t('createExpertModal.showSourceLinksHint') }}
                </p>
              </div>

              <div v-if="form.autoWebSearch" class="ml-5 space-y-2">
                <div>
                  <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.searchDomains') }}</label>
                  <input
                    v-model="form.searchDomains"
                    type="text"
                    :placeholder="$t('createExpertModal.searchDomainsPlaceholder')"
                    class="w-full px-3 py-1.5 text-xs bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded-lg"
                  />
                </div>
                <div>
                  <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.maxResults') }}: {{ form.maxSearchResults }}</label>
                  <input v-model.number="form.maxSearchResults" type="range" min="1" max="10" class="w-full h-1.5" />
                </div>
              </div>

              <!-- Dateisuche -->
              <label class="flex items-center gap-2 text-xs cursor-pointer">
                <input type="checkbox" v-model="form.autoFileSearch" class="w-3.5 h-3.5 text-green-600 rounded" />
                <span class="text-gray-700 dark:text-gray-300">{{ $t('createExpertModal.enableFileSearch') }}</span>
              </label>

              <!-- Dokumenten-Verzeichnis -->
              <div>
                <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.documentDirectory') }}</label>
                <div class="flex items-center gap-1">
                  <span class="text-xs text-gray-400">~/Dokumente/Fleet-Navigator/</span>
                  <input
                    v-model="form.documentDirectory"
                    type="text"
                    placeholder="Name"
                    class="flex-1 px-2 py-1 text-xs bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded"
                  />
                </div>
              </div>

              <!-- Vektor-RAG (PostgreSQL erforderlich) -->
              <div class="border-t border-gray-200 dark:border-gray-700 pt-3 mt-3">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-xs font-medium text-gray-700 dark:text-gray-300 flex items-center gap-1">
                    <span>📚</span> {{ $t('createExpertModal.vectorRag') }}
                  </span>
                  <span
                    v-if="!postgresConnected"
                    class="px-1.5 py-0.5 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 rounded text-[10px]"
                  >
                    {{ $t('createExpertModal.postgresRequired') }}
                  </span>
                </div>

                <!-- RAG disabled overlay -->
                <div :class="{ 'opacity-40 pointer-events-none': !postgresConnected }">
                  <label class="flex items-center gap-2 text-xs cursor-pointer">
                    <input
                      type="checkbox"
                      v-model="form.ragEnabled"
                      :disabled="!postgresConnected"
                      class="w-3.5 h-3.5 text-purple-600 rounded"
                    />
                    <span class="text-gray-700 dark:text-gray-300">{{ $t('createExpertModal.enableVectorSearch') }}</span>
                  </label>

                  <div v-if="form.ragEnabled" class="ml-5 mt-2 space-y-2">
                    <div>
                      <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.embeddingModel') }}</label>
                      <select
                        v-model="form.ragEmbeddingModel"
                        :disabled="!postgresConnected"
                        class="w-full px-2 py-1 text-xs bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 rounded"
                      >
                        <option value="nomic-embed-text">nomic-embed-text ({{ $t('createExpertModal.recommended') }})</option>
                        <option value="mxbai-embed-large">mxbai-embed-large</option>
                        <option value="all-minilm">all-minilm ({{ $t('createExpertModal.fast') }})</option>
                      </select>
                    </div>
                    <div class="grid grid-cols-2 gap-2">
                      <div>
                        <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.topK') }}: {{ form.ragTopK }}</label>
                        <input v-model.number="form.ragTopK" type="range" min="1" max="10" class="w-full h-1.5" />
                      </div>
                      <div>
                        <label class="block text-xs text-gray-600 dark:text-gray-400 mb-1">{{ $t('createExpertModal.similarity') }}: {{ (form.ragThreshold * 100).toFixed(0) }}%</label>
                        <input v-model.number="form.ragThreshold" type="range" min="0.5" max="0.95" step="0.05" class="w-full h-1.5" />
                      </div>
                    </div>
                    <p class="text-[10px] text-gray-500 dark:text-gray-400">
                      {{ $t('createExpertModal.uploadDocsHint') }}
                    </p>
                  </div>
                </div>

                <!-- PostgreSQL Migration Hinweis -->
                <div v-if="!postgresConnected" class="mt-2 p-2 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded text-[10px] text-yellow-700 dark:text-yellow-300">
                  {{ $t('createExpertModal.migratePostgres') }}
                </div>
              </div>
            </div>
          </div>

          <!-- Collapsible: Blickwinkel (nur beim Bearbeiten) -->
          <div v-if="isEditing" class="border-t border-gray-200 dark:border-gray-700 pt-3">
            <button
              @click="showModes = !showModes"
              class="flex items-center gap-1 text-xs font-medium text-purple-600 dark:text-purple-400"
            >
              <ChevronDownIcon class="w-3 h-3 transition-transform" :class="{ 'rotate-180': showModes }" />
              <EyeIcon class="w-3 h-3" />
              {{ $t('createExpertModal.perspectives') }} ({{ modes?.length || 0 }})
            </button>

            <div v-if="showModes" class="mt-3 space-y-2">
              <!-- Existing Modes -->
              <div
                v-for="mode in modes"
                :key="mode.id"
                class="flex items-center justify-between p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg text-xs"
              >
                <span class="font-medium text-gray-900 dark:text-white">{{ mode.name }}</span>
                <div class="flex gap-1">
                  <button @click="openEditMode(mode)" class="p-1 text-amber-500 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded">
                    <PencilIcon class="w-3 h-3" />
                  </button>
                  <button @click="deleteMode(mode)" class="p-1 text-red-500 hover:bg-red-100 dark:hover:bg-red-900/30 rounded">
                    <TrashIcon class="w-3 h-3" />
                  </button>
                </div>
              </div>

              <!-- Add Mode -->
              <div v-if="!showAddMode">
                <button @click="showAddMode = true" class="text-xs text-purple-600 dark:text-purple-400 hover:underline">
                  + {{ $t('createExpertModal.addPerspective') }}
                </button>
              </div>
              <div v-else class="p-2 bg-gray-50 dark:bg-gray-900/50 rounded-lg space-y-2">
                <input
                  v-model="newMode.name"
                  type="text"
                  :placeholder="$t('createExpertModal.perspectiveNamePlaceholder')"
                  class="w-full px-2 py-1 text-xs bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded"
                />
                <textarea
                  v-model="newMode.promptAddition"
                  rows="2"
                  :placeholder="$t('createExpertModal.promptAdditionPlaceholder')"
                  class="w-full px-2 py-1 text-xs bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded resize-none"
                ></textarea>
                <div class="flex justify-end gap-2">
                  <button @click="showAddMode = false; resetNewMode()" class="px-2 py-1 text-xs text-gray-500">{{ $t('common.cancel') }}</button>
                  <button @click="addMode" :disabled="!newMode.name?.trim()" class="px-2 py-1 text-xs bg-purple-500 text-white rounded disabled:opacity-50">
                    {{ $t('createExpertModal.add') }}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer mit Buttons (auch unten) -->
        <div class="flex justify-end gap-2 px-4 py-3 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
          <button
            @click="$emit('close')"
            class="px-3 py-1.5 text-sm rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
          >
            {{ $t('common.cancel') }}
          </button>
          <button
            @click="save"
            :disabled="!canSave || isSaving"
            class="px-4 py-1.5 text-sm rounded-lg bg-purple-500 hover:bg-purple-600 text-white font-medium transition-colors disabled:opacity-50"
          >
            {{ isSaving ? $t('common.saving') : $t(isEditing ? 'common.save' : 'common.create') }}
          </button>
        </div>
      </div>
    </div>
  </Transition>

  <!-- Mode Edit Modal -->
  <ExpertModeModal
    :show="showModeModal"
    :expert="expert"
    :mode="editingMode"
    @close="closeModeModal"
    @saved="onModeSaved"
  />
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { XMarkIcon, ChevronDownIcon, PlusIcon, TrashIcon, EyeIcon, DocumentArrowUpIcon, PencilIcon, GlobeAltIcon, DocumentMagnifyingGlassIcon, PhotoIcon, ArrowUpTrayIcon } from '@heroicons/vue/24/outline'
import axios from 'axios'
import api from '../services/api'
import { useChatStore } from '../stores/chatStore'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import { secureFetch } from '../utils/secureFetch'
import ExpertModeModal from './ExpertModeModal.vue'

const { t } = useI18n()
const { confirmDelete: showDeleteConfirm } = useConfirmDialog()

const props = defineProps({
  show: Boolean,
  expert: Object,
  availableModels: Array
})

const emit = defineEmits(['close', 'saved'])

// DEBUG: Log when props change
watch(() => props.availableModels, (newModels) => {
  console.log('CreateExpertModal: availableModels received:', newModels?.length, newModels)
}, { immediate: true })

watch(() => props.show, (isShown) => {
  if (isShown) {
    console.log('CreateExpertModal OPENED - availableModels:', props.availableModels?.length, props.availableModels)
  }
})
const { success, error } = useToast()
const chatStore = useChatStore()

const form = ref({
  name: '',
  role: '',
  description: '',
  avatarUrl: '',
  voice: 'de_DE-thorsten-medium', // TTS Stimme
  basePrompt: '',
  personalityPrompt: '',
  baseModel: '',
  providerType: '',
  ggufModel: '',
  defaultTemperature: 0.7,
  defaultTopP: 0.9,
  defaultNumCtx: 65536,  // 64K Default
  defaultMaxTokens: 4096,
  autoWebSearch: false,
  webSearchShowLinks: true, // Links in Antwort anzeigen (Default: true)
  searchDomains: '',
  maxSearchResults: 5,
  autoFileSearch: false,
  documentDirectory: '',
  showSwitchNotification: true,
  // RAG Vector Settings
  ragEnabled: false,
  ragEmbeddingModel: 'nomic-embed-text',
  ragTopK: 5,
  ragThreshold: 0.7
})

// Verfügbare TTS Stimmen mit Beispielsätzen
const availableVoices = [
  // Empfohlene Stimmen zuerst
  { id: 'de_DE-kerstin-low', name: 'Kerstin', description: 'Weiblich, klar ⭐', gender: 'female', sample: 'Mein Name ist Kerstin. Ich stehe Ihnen gerne zur Verfügung.' },
  { id: 'de_DE-thorsten-high', name: 'Thorsten HD', description: 'Männlich, beste Qualität ⭐', gender: 'male', sample: 'Guten Tag, mein Name ist Thorsten. Ich freue mich, Sie kennenzulernen.' },
  // Weitere Stimmen
  { id: 'de_DE-thorsten_emotional-medium', name: 'Thorsten Emotional', description: 'Männlich, expressiv', gender: 'male', sample: 'Das ist ja fantastisch! Ich bin wirklich begeistert von dieser Idee!' },
  { id: 'de_DE-ramona-low', name: 'Ramona', description: 'Weiblich, warm', gender: 'female', sample: 'Hallo, hier spricht Ramona. Wie geht es Ihnen heute?' },
  { id: 'de_DE-karlsson-low', name: 'Karlsson', description: 'Männlich, tief', gender: 'male', sample: 'Guten Tag, Karlsson hier. Womit kann ich Ihnen dienen?' },
  { id: 'de_DE-pavoque-low', name: 'Pavoque', description: 'Männlich, professionell', gender: 'male', sample: 'Willkommen, ich bin Pavoque. Lassen Sie uns beginnen.' },
  { id: 'de_DE-thorsten-medium', name: 'Thorsten', description: 'Männlich, neutral', gender: 'male', sample: 'Hallo, ich bin Thorsten. Wie kann ich Ihnen heute behilflich sein?' },
  { id: 'de_DE-eva_k-x_low', name: 'Eva K', description: 'Weiblich, kompakt', gender: 'female', sample: 'Hallo, ich bin Eva. Schön, dass Sie hier sind.' },
]

// Voice testing state
const isTestingVoice = ref(false)
let currentTestAudio = null

// Test voice with sample sentence
async function testVoice(voice) {
  if (isTestingVoice.value) {
    // Stop current playback
    if (currentTestAudio) {
      currentTestAudio.pause()
      currentTestAudio = null
    }
    isTestingVoice.value = false
    return
  }

  isTestingVoice.value = true

  try {
    const response = await fetch('/api/voice/tts', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: voice.sample,
        voice: voice.id
      })
    })

    if (!response.ok) {
      throw new Error('TTS fehlgeschlagen')
    }

    const audioBlob = await response.blob()
    const audioUrl = URL.createObjectURL(audioBlob)

    currentTestAudio = new Audio(audioUrl)
    currentTestAudio.onended = () => {
      isTestingVoice.value = false
      currentTestAudio = null
      URL.revokeObjectURL(audioUrl)
    }
    currentTestAudio.onerror = () => {
      isTestingVoice.value = false
      currentTestAudio = null
      URL.revokeObjectURL(audioUrl)
    }

    await currentTestAudio.play()
  } catch (err) {
    console.error('Voice test failed:', err)
    error(t('createExpertModal.toast.voiceTestFailed'))
    isTestingVoice.value = false
  }
}

// PostgreSQL Status for RAG
const postgresConnected = ref(false)

onMounted(async () => {
  await checkPostgresStatus()
})

async function checkPostgresStatus() {
  try {
    const response = await secureFetch('/api/database/status')
    if (response.ok) {
      const data = await response.json()
      postgresConnected.value = data.database === 'postgres'
    }
  } catch (error) {
    console.debug('Failed to check database status:', error)
    postgresConnected.value = false
  }
}

const showAdvanced = ref(false)
const showRAG = ref(false)
const showModes = ref(false)
const isSaving = ref(false)

// Auto-Context: Beim Modell-Wechsel automatisch max. Context-Größe laden
const isLoadingContext = ref(false)
watch(() => form.value.baseModel, async (newModel) => {
  if (!newModel) return
  try {
    isLoadingContext.value = true
    const data = await api.getModelDetails(newModel)
    if (data && data.context_length) {
      form.value.defaultNumCtx = data.context_length
    }
  } catch (err) {
    console.warn('Konnte Context-Größe nicht laden:', err.message)
  } finally {
    isLoadingContext.value = false
  }
})

// File input refs
const basePromptFileInput = ref(null)
const avatarFileInput = ref(null)
const isUploadingAvatar = ref(false)
const avatarUploadError = ref('')

// Modes management
const modes = ref([])
const showAddMode = ref(false)
const newMode = ref({ name: '', promptAddition: '', temperature: null })

// Mode editing modal
const showModeModal = ref(false)
const editingMode = ref(null)

const isEditing = computed(() => !!props.expert)

const canSave = computed(() => {
  return form.value.name?.trim() &&
         form.value.role?.trim() &&
         form.value.basePrompt?.trim() &&
         form.value.baseModel
})

// Watch for expert changes (edit mode)
watch(() => props.expert, (newExpert) => {
  if (newExpert) {
    form.value = {
      name: newExpert.name || '',
      role: newExpert.role || '',
      description: newExpert.description || '',
      avatarUrl: newExpert.avatarUrl || '',
      voice: newExpert.voice || 'de_DE-thorsten-medium', // TTS Stimme
      basePrompt: newExpert.basePrompt || '',
      personalityPrompt: newExpert.personalityPrompt || '',
      baseModel: newExpert.baseModel || newExpert.model || '',
      providerType: newExpert.providerType || '',
      ggufModel: newExpert.ggufModel || '',
      defaultTemperature: newExpert.defaultTemperature ?? 0.7,
      defaultTopP: newExpert.defaultTopP ?? 0.9,
      defaultNumCtx: newExpert.defaultNumCtx ?? 65536,  // 64K Default
      defaultMaxTokens: newExpert.defaultMaxTokens ?? 4096,
      autoWebSearch: newExpert.autoWebSearch ?? false,
      webSearchShowLinks: newExpert.webSearchShowLinks ?? true, // Default: Links anzeigen
      searchDomains: newExpert.searchDomains || '',
      maxSearchResults: newExpert.maxSearchResults ?? 5,
      autoFileSearch: newExpert.autoFileSearch ?? false,
      documentDirectory: newExpert.documentDirectory || '',
      showSwitchNotification: newExpert.showSwitchNotification ?? true,
      // RAG Vector Settings
      ragEnabled: newExpert.ragEnabled ?? false,
      ragEmbeddingModel: newExpert.ragEmbeddingModel || 'nomic-embed-text',
      ragTopK: newExpert.ragTopK ?? 5,
      ragThreshold: newExpert.ragThreshold ?? 0.7
    }
    modes.value = newExpert.modes ? [...newExpert.modes] : []
  } else {
    resetForm()
  }
}, { immediate: true })

watch(() => props.show, (show) => {
  if (!show && !props.expert) {
    resetForm()
  }
})

function resetForm() {
  form.value = {
    name: '',
    role: '',
    description: '',
    avatarUrl: '',
    voice: 'de_DE-thorsten-medium', // TTS Stimme
    basePrompt: '',
    personalityPrompt: '',
    baseModel: '',
    providerType: '',
    ggufModel: '',
    defaultTemperature: 0.7,
    defaultTopP: 0.9,
    defaultNumCtx: 65536,  // 64K Default
    defaultMaxTokens: 4096,
    autoWebSearch: false,
    webSearchShowLinks: true, // Default: Links anzeigen
    searchDomains: '',
    maxSearchResults: 5,
    autoFileSearch: false,
    documentDirectory: '',
    showSwitchNotification: true,
    // RAG Vector Settings
    ragEnabled: false,
    ragEmbeddingModel: 'nomic-embed-text',
    ragTopK: 5,
    ragThreshold: 0.7
  }
  modes.value = []
  showAdvanced.value = false
  showRAG.value = false
  showModes.value = false
  showAddMode.value = false
  resetNewMode()
}

function handleAvatarError(event) {
  console.warn('Avatar konnte nicht geladen werden:', form.value.avatarUrl)
}

function triggerAvatarFileInput() {
  avatarFileInput.value?.click()
}

async function handleAvatarUpload(event) {
  const file = event.target.files?.[0]
  if (!file) return

  avatarUploadError.value = ''

  if (file.size > 5 * 1024 * 1024) {
    avatarUploadError.value = t('createExpertModal.toast.fileTooLarge')
    event.target.value = ''
    return
  }

  try {
    isUploadingAvatar.value = true
    const formData = new FormData()
    formData.append('file', file)

    const response = await axios.post('/api/experts/avatar/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })

    if (response.data.success) {
      form.value.avatarUrl = response.data.avatarUrl
      success(t('createExpertModal.toast.avatarUploaded'))
    } else {
      avatarUploadError.value = response.data.error || t('createExpertModal.toast.uploadFailed')
    }
  } catch (err) {
    console.error('Avatar upload failed:', err)
    avatarUploadError.value = err.response?.data?.error || t('createExpertModal.toast.uploadFailed')
  } finally {
    isUploadingAvatar.value = false
    event.target.value = ''
  }
}

function resetNewMode() {
  newMode.value = { name: '', promptAddition: '', temperature: null }
}

async function addMode() {
  if (!newMode.value.name?.trim()) return
  if (!props.expert?.id) {
    error(t('createExpertModal.toast.saveExpertFirst'))
    return
  }

  try {
    const created = await api.createExpertMode(props.expert.id, newMode.value)
    modes.value.push(created)
    resetNewMode()
    showAddMode.value = false
    success(t('createExpertModal.toast.perspectiveAdded', { name: created.name }))
  } catch (err) {
    console.error('Failed to create mode:', err)
    error(err.response?.data?.error || t('createExpertModal.toast.createError'))
  }
}

async function deleteMode(mode) {
  const confirmed = await showDeleteConfirm(mode.name)
  if (!confirmed) return

  try {
    await api.deleteExpertMode(props.expert.id, mode.id)
    modes.value = modes.value.filter(m => m.id !== mode.id)
    success(t('createExpertModal.toast.perspectiveDeleted', { name: mode.name }))
  } catch (err) {
    console.error('Failed to delete mode:', err)
    error(t('createExpertModal.toast.deleteError'))
  }
}

function openEditMode(mode) {
  editingMode.value = mode
  showModeModal.value = true
}

function closeModeModal() {
  showModeModal.value = false
  editingMode.value = null
}

async function onModeSaved() {
  closeModeModal()
  if (props.expert?.id) {
    try {
      const updatedModes = await api.getExpertModes(props.expert.id)
      modes.value = updatedModes
    } catch (err) {
      console.error('Failed to reload modes:', err)
    }
  }
}

function triggerBasePromptFileInput() {
  basePromptFileInput.value?.click()
}

async function loadBasePromptFromFile(event) {
  const file = event.target.files?.[0]
  if (!file) return

  try {
    const text = await file.text()
    form.value.basePrompt = text.trim()
    success(t('createExpertModal.toast.promptLoaded', { filename: file.name }))
  } catch (err) {
    console.error('Failed to read file:', err)
    error(t('createExpertModal.toast.fileReadError'))
  }
  event.target.value = ''
}

async function save() {
  if (!canSave.value) return

  isSaving.value = true

  try {
    // Transform form fields to match backend expected field names
    const payload = {
      name: form.value.name,
      role: form.value.role,
      description: form.value.description,
      avatar: form.value.avatarUrl,
      voice: form.value.voice,  // TTS Stimme
      basePrompt: form.value.basePrompt,
      personalityPrompt: form.value.personalityPrompt,  // Kommunikationsstil
      model: form.value.baseModel,  // Backend expects "model", not "baseModel"
      defaultNumCtx: form.value.defaultNumCtx,  // Context-Größe
      defaultTemperature: form.value.defaultTemperature,
      defaultTopP: form.value.defaultTopP,
      defaultMaxTokens: form.value.defaultMaxTokens,
      is_active: true,
      auto_mode_switch: form.value.showSwitchNotification,
      // Web Search Settings
      autoWebSearch: form.value.autoWebSearch,
      webSearchShowLinks: form.value.webSearchShowLinks
    }

    let savedExpertId = null
    if (isEditing.value) {
      await api.updateExpert(props.expert.id, payload)
      savedExpertId = props.expert.id
      success(t('createExpertModal.toast.expertUpdated', { name: form.value.name }))
    } else {
      const result = await api.createExpert(payload)
      savedExpertId = result.id
      success(t('createExpertModal.toast.expertCreated', { name: form.value.name }))
    }

    // Wenn dieser Experte gerade aktiv ist, Context sofort aktualisieren
    if (savedExpertId && chatStore.selectedExpertId === savedExpertId) {
      const newContext = form.value.defaultNumCtx || 65536
      try {
        const result = await api.changeContextSize(newContext)
        if (result.restartNeeded) {
          console.log(`🔄 Context auf ${newContext} aktualisiert (Server neugestartet)`)
        }
        // Update contextUsage display
        chatStore.contextUsage.maxContextTokens = newContext
      } catch (ctxErr) {
        console.warn('Context-Update fehlgeschlagen:', ctxErr)
      }
    }

    emit('saved')
  } catch (err) {
    console.error('Failed to save expert:', err)
    error(err.response?.data?.error || t('createExpertModal.toast.saveError'))
  } finally {
    isSaving.value = false
  }
}
</script>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}
</style>
