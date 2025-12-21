<template>
  <Transition name="modal">
    <div v-if="show" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-[60] p-4">
      <div class="bg-white dark:bg-gray-800 rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <!-- Header -->
        <div class="flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700 bg-gradient-to-r from-purple-50 to-indigo-50 dark:from-purple-900/20 dark:to-indigo-900/20">
          <h3 class="text-xl font-bold text-gray-900 dark:text-white">
            {{ isEditing ? 'Experte bearbeiten' : 'Neuen Experten erstellen' }}
          </h3>
          <button
            @click="$emit('close')"
            class="p-2 rounded-lg text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          >
            <XMarkIcon class="w-5 h-5" />
          </button>
        </div>

        <!-- Form -->
        <div class="p-6 space-y-5">
          <!-- Name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Name *
            </label>
            <input
              v-model="form.name"
              type="text"
              placeholder="z.B. Roland, Ayşe, Dr. Schmidt"
              class="w-full px-4 py-2.5 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
          </div>

          <!-- Role -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Rolle/Beruf *
            </label>
            <input
              v-model="form.role"
              type="text"
              placeholder="z.B. Rechtsanwalt, Steuerberater, Marketing-Experte"
              class="w-full px-4 py-2.5 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
          </div>

          <!-- Description -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Beschreibung
            </label>
            <input
              v-model="form.description"
              type="text"
              placeholder="Kurze Beschreibung des Experten"
              class="w-full px-4 py-2.5 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            />
          </div>

          <!-- Base Model -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Basis-Modell *
            </label>
            <select
              v-model="form.baseModel"
              class="w-full px-4 py-2.5 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
            >
              <option value="">-- Bitte wählen --</option>
              <option v-for="model in availableModels" :key="model.name" :value="model.name">
                {{ model.name }}
              </option>
            </select>
          </div>

          <!-- Base Prompt -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Basis-Prompt (Persönlichkeit) *
            </label>
            <textarea
              v-model="form.basePrompt"
              rows="6"
              placeholder="Du bist [Name], ein/e erfahrene/r [Rolle].

DEINE EXPERTISE:
- ...

DEIN STIL:
- ..."
              class="w-full px-4 py-2.5 bg-white dark:bg-gray-900 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none font-mono text-sm"
            ></textarea>
            <p class="text-xs text-gray-500 mt-1">
              Dieser Prompt definiert die Basis-Persönlichkeit und gilt für alle Modi.
            </p>
          </div>

          <!-- Advanced Settings -->
          <div class="border-t border-gray-200 dark:border-gray-700 pt-5">
            <button
              @click="showAdvanced = !showAdvanced"
              class="flex items-center gap-2 text-sm font-medium text-purple-600 dark:text-purple-400"
            >
              <ChevronDownIcon class="w-4 h-4 transition-transform" :class="{ 'rotate-180': showAdvanced }" />
              Erweiterte Einstellungen
            </button>

            <div v-if="showAdvanced" class="mt-4 grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Default Temperature: {{ form.defaultTemperature }}
                </label>
                <input
                  v-model.number="form.defaultTemperature"
                  type="range"
                  min="0"
                  max="2"
                  step="0.1"
                  class="w-full"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Default Top P: {{ form.defaultTopP }}
                </label>
                <input
                  v-model.number="form.defaultTopP"
                  type="range"
                  min="0"
                  max="1"
                  step="0.05"
                  class="w-full"
                />
              </div>
              <div class="col-span-2">
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Default Context Length: {{ form.defaultNumCtx?.toLocaleString() }}
                </label>
                <input
                  v-model.number="form.defaultNumCtx"
                  type="range"
                  min="2048"
                  max="131072"
                  step="2048"
                  class="w-full"
                />
              </div>
            </div>
          </div>

          <!-- Modes Management (only when editing) -->
          <div v-if="isEditing" class="border-t border-gray-200 dark:border-gray-700 pt-5">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <EyeIcon class="w-5 h-5 text-purple-500" />
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Blickwinkel (Modi)
                </span>
                <span class="px-2 py-0.5 text-xs rounded-full bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400">
                  {{ modes.length }}
                </span>
              </div>
              <button
                @click="showAddMode = !showAddMode"
                class="text-sm text-purple-600 dark:text-purple-400 hover:text-purple-700 dark:hover:text-purple-300 flex items-center gap-1"
              >
                <PlusIcon class="w-4 h-4" />
                Hinzufügen
              </button>
            </div>

            <!-- Existing Modes -->
            <div v-if="modes.length > 0" class="space-y-2 mb-3">
              <div
                v-for="mode in modes"
                :key="mode.id"
                class="flex items-center justify-between p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg border border-purple-200 dark:border-purple-700/50"
              >
                <div>
                  <div class="font-medium text-gray-900 dark:text-white text-sm">{{ mode.name }}</div>
                  <div v-if="mode.promptAddition" class="text-xs text-gray-500 dark:text-gray-400 truncate max-w-xs">
                    {{ mode.promptAddition.substring(0, 50) }}{{ mode.promptAddition.length > 50 ? '...' : '' }}
                  </div>
                </div>
                <button
                  @click="deleteMode(mode)"
                  class="p-1.5 rounded-lg text-red-500 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors"
                  title="Löschen"
                >
                  <TrashIcon class="w-4 h-4" />
                </button>
              </div>
            </div>
            <div v-else class="text-sm text-gray-500 dark:text-gray-400 italic mb-3">
              Noch keine Blickwinkel definiert
            </div>

            <!-- Add New Mode Form -->
            <div v-if="showAddMode" class="bg-gray-50 dark:bg-gray-900/50 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
              <div class="space-y-3">
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Name *</label>
                  <input
                    v-model="newMode.name"
                    type="text"
                    placeholder="z.B. Kritisch, Kreativ, Formal"
                    class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-700 dark:text-gray-300 mb-1">Prompt-Zusatz</label>
                  <textarea
                    v-model="newMode.promptAddition"
                    rows="3"
                    placeholder="Zusätzliche Anweisungen für diesen Modus..."
                    class="w-full px-3 py-2 text-sm bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent resize-none"
                  ></textarea>
                </div>
                <div class="flex justify-end gap-2">
                  <button
                    @click="showAddMode = false; resetNewMode()"
                    class="px-3 py-1.5 text-sm rounded-lg text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors"
                  >
                    Abbrechen
                  </button>
                  <button
                    @click="addMode"
                    :disabled="!newMode.name?.trim()"
                    class="px-3 py-1.5 text-sm rounded-lg bg-purple-500 hover:bg-purple-600 text-white font-medium transition-colors disabled:opacity-50"
                  >
                    Hinzufügen
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="flex justify-end gap-3 p-6 border-t border-gray-200 dark:border-gray-700">
          <button
            @click="$emit('close')"
            class="px-4 py-2 rounded-lg text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors"
          >
            Abbrechen
          </button>
          <button
            @click="save"
            :disabled="!canSave || isSaving"
            class="px-4 py-2 rounded-lg bg-gradient-to-r from-purple-500 to-indigo-500 hover:from-purple-600 hover:to-indigo-600 text-white font-medium transition-all disabled:opacity-50 flex items-center gap-2"
          >
            <span v-if="isSaving">Speichern...</span>
            <span v-else>{{ isEditing ? 'Speichern' : 'Erstellen' }}</span>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { XMarkIcon, ChevronDownIcon, PlusIcon, TrashIcon, EyeIcon } from '@heroicons/vue/24/outline'
import api from '../services/api'
import { useToast } from '../composables/useToast'

const props = defineProps({
  show: Boolean,
  expert: Object,
  availableModels: Array
})

const emit = defineEmits(['close', 'saved'])
const { success, error } = useToast()

const form = ref({
  name: '',
  role: '',
  description: '',
  basePrompt: '',
  baseModel: '',
  defaultTemperature: 0.7,
  defaultTopP: 0.9,
  defaultNumCtx: 8192
})

const showAdvanced = ref(false)
const isSaving = ref(false)

// Modes management
const modes = ref([])
const showAddMode = ref(false)
const newMode = ref({
  name: '',
  promptAddition: '',
  temperature: null
})

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
      basePrompt: newExpert.basePrompt || '',
      baseModel: newExpert.baseModel || '',
      defaultTemperature: newExpert.defaultTemperature ?? 0.7,
      defaultTopP: newExpert.defaultTopP ?? 0.9,
      defaultNumCtx: newExpert.defaultNumCtx ?? 8192
    }
    // Load modes
    modes.value = newExpert.modes ? [...newExpert.modes] : []
  } else {
    resetForm()
  }
}, { immediate: true })

// Watch show to reset form on close
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
    basePrompt: '',
    baseModel: '',
    defaultTemperature: 0.7,
    defaultTopP: 0.9,
    defaultNumCtx: 8192
  }
  modes.value = []
  showAdvanced.value = false
  showAddMode.value = false
  resetNewMode()
}

function resetNewMode() {
  newMode.value = {
    name: '',
    promptAddition: '',
    temperature: null
  }
}

async function addMode() {
  if (!newMode.value.name?.trim()) return
  if (!props.expert?.id) {
    error('Bitte speichere den Experten zuerst')
    return
  }

  try {
    const created = await api.createExpertMode(props.expert.id, newMode.value)
    modes.value.push(created)
    resetNewMode()
    showAddMode.value = false
    success(`Blickwinkel "${created.name}" hinzugefügt`)
  } catch (err) {
    console.error('Failed to create mode:', err)
    error(err.response?.data?.error || 'Fehler beim Erstellen')
  }
}

async function deleteMode(mode) {
  if (!confirm(`Blickwinkel "${mode.name}" wirklich löschen?`)) return

  try {
    await api.deleteExpertMode(props.expert.id, mode.id)
    modes.value = modes.value.filter(m => m.id !== mode.id)
    success(`Blickwinkel "${mode.name}" gelöscht`)
  } catch (err) {
    console.error('Failed to delete mode:', err)
    error('Fehler beim Löschen')
  }
}

async function save() {
  if (!canSave.value) return

  isSaving.value = true

  try {
    if (isEditing.value) {
      await api.updateExpert(props.expert.id, form.value)
      success(`Experte "${form.value.name}" aktualisiert`)
    } else {
      await api.createExpert(form.value)
      success(`Experte "${form.value.name}" erstellt`)
    }
    emit('saved')
  } catch (err) {
    console.error('Failed to save expert:', err)
    error(err.response?.data?.error || 'Fehler beim Speichern')
  } finally {
    isSaving.value = false
  }
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
</style>
