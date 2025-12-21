<template>
  <div class="expert-system min-h-screen p-6 bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 overflow-y-auto">
    <!-- Header -->
    <div class="mb-8">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <div class="p-3 rounded-xl bg-gradient-to-br from-purple-500 to-indigo-600 shadow-lg">
            <UserGroupIcon class="w-8 h-8 text-white" />
          </div>
          <div>
            <h1 class="text-3xl font-bold text-white">{{ $t('expertManager.title') }}</h1>
            <p class="text-gray-400 mt-1">{{ $t('expertManager.subtitle') }}</p>
          </div>
        </div>
        <button
          @click="showCreateExpertModal = true"
          class="
            px-4 py-2.5 rounded-xl
            bg-gradient-to-r from-purple-500 to-indigo-500
            hover:from-purple-600 hover:to-indigo-600
            text-white font-medium
            shadow-lg hover:shadow-xl hover:shadow-purple-500/20
            transition-all duration-200
            flex items-center gap-2
          "
        >
          <PlusIcon class="w-5 h-5" />
          {{ $t('expertManager.createNewExpert') }}
        </button>
      </div>

      <!-- Summary Cards -->
      <div class="flex items-center justify-end mt-4">
        <div class="flex gap-4">
          <div class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 backdrop-blur-sm px-6 py-3 rounded-xl border border-gray-700/50">
            <div class="text-xs text-gray-400 mb-1">{{ $t('expertManager.total') }}</div>
            <div class="text-2xl font-bold text-white">{{ experts?.length || 0 }}</div>
          </div>
          <div class="bg-gradient-to-br from-purple-500/20 to-indigo-500/20 backdrop-blur-sm px-6 py-3 rounded-xl border border-purple-500/30">
            <div class="text-xs text-purple-400 mb-1">{{ $t('expertManager.modes') }}</div>
            <div class="text-2xl font-bold text-purple-400">{{ totalModes }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-purple-500"></div>
    </div>

    <!-- Empty State -->
    <div v-else-if="!experts?.length" class="
      bg-gradient-to-br from-gray-800/30 to-gray-900/30
      backdrop-blur-sm
      p-12 rounded-2xl
      border border-gray-700/30 border-dashed
      text-center
    ">
      <UserGroupIcon class="w-20 h-20 text-gray-600 mx-auto mb-4" />
      <h3 class="text-xl font-semibold text-gray-400 mb-2">{{ $t('expertManager.noExpertsTitle') }}</h3>
      <p class="text-sm text-gray-500 max-w-md mx-auto mb-6">
        {{ $t('expertManager.noExpertsDesc') }}
      </p>
      <button
        @click="showCreateExpertModal = true"
        class="
          px-6 py-3 rounded-xl
          bg-gradient-to-r from-purple-500 to-indigo-500
          text-white font-medium
          shadow-lg hover:shadow-xl
          transition-all
          inline-flex items-center gap-2
        "
      >
        <PlusIcon class="w-5 h-5" />
        {{ $t('expertManager.createFirst') }}
      </button>
    </div>

    <!-- Expert Cards Grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div
        v-for="expert in experts"
        :key="expert.id"
        class="
          group
          bg-gradient-to-br from-gray-800/50 to-gray-900/50
          backdrop-blur-sm
          rounded-2xl
          border border-gray-700/50
          overflow-hidden
          transition-all duration-300
          hover:shadow-2xl
          hover:shadow-purple-500/20
          hover:border-purple-500/50
        "
      >
        <!-- Card Header -->
        <div class="p-5 border-b border-gray-700/50 bg-gradient-to-r from-purple-900/20 to-indigo-900/20">
          <div class="flex items-start justify-between">
            <div class="flex items-center gap-3">
              <!-- Avatar oder Fallback-Icon -->
              <div
                v-if="expert.avatarUrl"
                class="w-12 h-12 rounded-lg overflow-hidden border-2 border-purple-500/50 group-hover:scale-110 transition-transform duration-300"
              >
                <img
                  :src="expert.avatarUrl"
                  :alt="expert.name"
                  class="w-full h-full object-cover"
                  @error="(e) => e.target.style.display = 'none'"
                />
              </div>
              <div
                v-else
                class="p-2 rounded-lg bg-gradient-to-br from-purple-500 to-indigo-600 group-hover:scale-110 transition-transform duration-300"
              >
                <UserGroupIcon class="w-5 h-5 text-white" />
              </div>
              <div>
                <h3 class="text-xl font-bold text-white">{{ expert.name }}</h3>
                <p class="text-sm text-purple-400 font-medium">{{ expert.role }}</p>
              </div>
            </div>
            <div class="flex items-center gap-1">
              <button
                @click="editExpert(expert)"
                class="p-2 rounded-lg text-gray-400 hover:text-amber-400 hover:bg-amber-500/20 transition-colors"
                :title="$t('common.edit')"
              >
                <PencilIcon class="w-4 h-4" />
              </button>
              <button
                @click="confirmDeleteExpert(expert)"
                class="p-2 rounded-lg text-gray-400 hover:text-red-400 hover:bg-red-500/20 transition-colors"
                :title="$t('common.delete')"
              >
                <TrashIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <!-- Card Body -->
        <div class="p-5">
          <p class="text-sm text-gray-400 mb-4 line-clamp-2">
            {{ expert.description || $t('expertManager.noDescription') }}
          </p>

          <!-- Model Info -->
          <div class="flex items-center gap-2 text-xs text-gray-500 mb-4">
            <CpuChipIcon class="w-4 h-4" />
            <span>{{ expert.baseModel || expert.model }}</span>
          </div>

          <!-- Modes -->
          <div class="mb-4">
            <div class="flex items-center justify-between mb-2">
              <span class="text-sm font-medium text-gray-300">
                {{ $t('expertManager.perspectives') }}
              </span>
              <button
                @click="openAddModeModal(expert)"
                class="text-xs text-purple-400 hover:text-purple-300 flex items-center gap-1"
              >
                <PlusIcon class="w-3 h-3" />
                {{ $t('expertManager.add') }}
              </button>
            </div>

            <div v-if="expert.modes && expert.modes.length > 0" class="flex flex-wrap gap-2">
              <span
                v-for="mode in expert.modes"
                :key="mode.id"
                @click="editMode(expert, mode)"
                class="
                  px-2.5 py-1 rounded-full text-xs font-medium
                  bg-purple-500/20 border border-purple-500/30
                  text-purple-300
                  cursor-pointer hover:bg-purple-500/30
                  transition-colors
                "
              >
                {{ mode.name }}
              </span>
            </div>
            <p v-else class="text-xs text-gray-500 italic">
              {{ $t('expertManager.noModesYet') }}
            </p>
          </div>

          <!-- Actions -->
          <div class="pt-4 border-t border-gray-700/50">
            <button
              @click="testExpert(expert)"
              class="
                w-full py-2.5 px-4 rounded-lg
                bg-gradient-to-r from-purple-500 to-indigo-500
                hover:from-purple-600 hover:to-indigo-600
                text-white text-sm font-medium
                transition-all
                flex items-center justify-center gap-2
              "
            >
              <ChatBubbleLeftRightIcon class="w-4 h-4" />
              {{ $t('expertManager.test') }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Expert Modal -->
    <CreateExpertModal
      :show="showCreateExpertModal"
      :expert="editingExpert"
      :available-models="availableModels"
      @close="closeCreateExpertModal"
      @saved="onExpertSaved"
    />

    <!-- Add/Edit Mode Modal -->
    <ExpertModeModal
      :show="showModeModal"
      :expert="selectedExpert"
      :mode="editingMode"
      @close="closeModeModal"
      @saved="onModeSaved"
    />

    <!-- Test Expert Modal -->
    <TestExpertModal
      :show="showTestModal"
      :expert="testingExpert"
      @close="showTestModal = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  UserGroupIcon,
  PlusIcon,
  PencilIcon,
  TrashIcon,
  CpuChipIcon,
  ChatBubbleLeftRightIcon
} from '@heroicons/vue/24/outline'
import api from '../services/api'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import CreateExpertModal from './CreateExpertModal.vue'
import ExpertModeModal from './ExpertModeModal.vue'
import TestExpertModal from './TestExpertModal.vue'

const { t } = useI18n()
const { success, error } = useToast()
const { confirmDelete } = useConfirmDialog()

// State
const experts = ref([])
const availableModels = ref([])
const isLoading = ref(true)

// Computed
const totalModes = computed(() => {
  return experts.value.reduce((sum, expert) => sum + (expert.modes?.length || 0), 0)
})

// Modals
const showCreateExpertModal = ref(false)
const editingExpert = ref(null)

const showModeModal = ref(false)
const selectedExpert = ref(null)
const editingMode = ref(null)

const showTestModal = ref(false)
const testingExpert = ref(null)

// Load data
onMounted(async () => {
  await loadExperts()
  await loadModels()
})

async function loadExperts() {
  isLoading.value = true
  try {
    experts.value = await api.getAllExperts()
  } catch (err) {
    console.error('Failed to load experts:', err)
    error(t('expertManager.loadError'))
  } finally {
    isLoading.value = false
  }
}

async function loadModels() {
  try {
    const response = await api.getAvailableModels()
    console.log('ExpertManager loadModels - RAW API response:', response)
    // API gibt {current_model, models: [string]} zurück
    // Frontend erwartet [{name, displayName}]
    const modelList = response.models || response || []
    console.log('ExpertManager loadModels - modelList:', modelList)
    if (Array.isArray(modelList)) {
      availableModels.value = modelList.map(m => {
        if (typeof m === 'object' && m.name) return m
        return { name: m, displayName: m }
      })
    }
    console.log('ExpertManager loadModels - FINAL availableModels:', availableModels.value.length, availableModels.value.slice(0, 3))
  } catch (err) {
    console.error('Failed to load models:', err)
    availableModels.value = []
  }
}

// Expert actions
function editExpert(expert) {
  editingExpert.value = expert
  showCreateExpertModal.value = true
}

function closeCreateExpertModal() {
  showCreateExpertModal.value = false
  editingExpert.value = null
}

async function onExpertSaved() {
  closeCreateExpertModal()
  await loadExperts()
}

async function confirmDeleteExpert(expert) {
  const confirmed = await confirmDelete(expert.name, t('expertManager.deleteConfirmDesc'))
  if (!confirmed) return

  try {
    await api.deleteExpert(expert.id)
    success(t('expertManager.deleted', { name: expert.name }))
    await loadExperts()
  } catch (err) {
    console.error('Failed to delete expert:', err)
    error(t('expertManager.deleteError'))
  }
}

// Mode actions
function openAddModeModal(expert) {
  selectedExpert.value = expert
  editingMode.value = null
  showModeModal.value = true
}

function editMode(expert, mode) {
  selectedExpert.value = expert
  editingMode.value = mode
  showModeModal.value = true
}

function closeModeModal() {
  showModeModal.value = false
  selectedExpert.value = null
  editingMode.value = null
}

async function onModeSaved() {
  closeModeModal()
  await loadExperts()
}

// Test expert
function testExpert(expert) {
  testingExpert.value = expert
  showTestModal.value = true
}
</script>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
