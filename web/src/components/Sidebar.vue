<template>
  <!-- Resizable Sidebar Container -->
  <div
    v-if="!settingsStore.settings.sidebarCollapsed"
    class="sidebar-nav flex flex-col h-full min-h-0 bg-gradient-to-b from-gray-900 to-gray-950 text-white border-r border-gray-700/50 transition-all duration-300 relative"
    :style="{ width: sidebarWidth + 'px', minWidth: '180px', maxWidth: '400px' }"
  >
    <!-- Resize Handle -->
    <div
      class="absolute top-0 right-0 w-1 h-full cursor-ew-resize hover:bg-gray-500/50 active:bg-gray-400/50 z-10 transition-colors"
      @mousedown="startResize"
    ></div>
    <!-- Header -->
    <div class="sidebar-header p-4 border-b border-gray-700/50 bg-gray-900/50 backdrop-blur-sm">
      <!-- New Buttons with Icons -->
      <div class="flex gap-2">
        <button
          @click="handleNewProject"
          class="
            flex-1 px-3 py-2.5 rounded-xl
            bg-gradient-to-br from-purple-600 to-purple-700
            hover:from-purple-500 hover:to-purple-600
            font-medium text-sm
            transition-all duration-200
            transform hover:scale-105 active:scale-95
            shadow-lg hover:shadow-purple-500/30
            flex items-center justify-center gap-1.5
          "
          :title="$t('sidebar.newProject')"
        >
          <FolderPlusIcon class="w-4 h-4" />
          <span>{{ $t('sidebar.project') }}</span>
        </button>
        <button
          @click="handleNewChat"
          class="
            flex-1 px-3 py-2.5 rounded-xl
            bg-gradient-to-br from-fleet-orange-500 to-fleet-orange-600
            hover:from-fleet-orange-400 hover:to-fleet-orange-500
            font-medium text-sm
            transition-all duration-200
            transform hover:scale-105 active:scale-95
            shadow-lg hover:shadow-orange-500/30
            flex items-center justify-center gap-1.5
          "
          :title="$t('sidebar.newChat')"
        >
          <PlusCircleIcon class="w-4 h-4" />
          <span>{{ $t('sidebar.chat') }}</span>
        </button>
      </div>
    </div>

    <!-- Unified List with Custom Scrollbar -->
    <div class="flex-1 overflow-y-auto p-2 custom-scrollbar">
      <!-- Projects Section -->
      <div v-if="sortedProjects.length > 0">
        <div
          v-for="project in sortedProjects"
          :key="'project-' + project.id"
          class="relative group mb-1"
        >
          <div
            @click="selectProject(project)"
            class="
              flex items-center justify-between p-3 rounded-xl cursor-pointer
              transition-all duration-200
              hover:bg-gray-800/80 hover:backdrop-blur-sm
              hover:scale-[1.02] hover:shadow-md
            "
            :class="{
              'bg-purple-900/30 border border-purple-500/30': selectedProject?.id === project.id,
              'hover:border hover:border-gray-700/50': selectedProject?.id !== project.id
            }"
          >
            <div class="flex-1 truncate">
              <div class="flex items-center space-x-2">
                <div class="p-1.5 rounded-lg bg-purple-500/20">
                  <FolderIcon class="w-4 h-4 text-purple-400" />
                </div>
                <span class="font-medium truncate">{{ project.name }}</span>
              </div>
              <div class="text-xs text-gray-400 ml-8 flex items-center gap-2 mt-1">
                <span class="flex items-center gap-1">
                  <DocumentTextIcon class="w-3 h-3" />
                  {{ project.contextFiles?.length || 0 }}
                </span>
                <span class="flex items-center gap-1">
                  <ChatBubbleLeftRightIcon class="w-3 h-3" />
                  {{ getProjectChatCount(project.id) }}
                </span>
              </div>
            </div>
            <!-- Context Menu Button -->
            <button
              @click.stop="toggleProjectMenu(project.id)"
              class="
                opacity-0 group-hover:opacity-100
                p-1.5 rounded-lg
                text-gray-400 hover:text-white hover:bg-gray-700
                transition-all duration-200
              "
            >
              <EllipsisVerticalIcon class="w-5 h-5" />
            </button>
          </div>

          <!-- Project Dropdown Menu with Glassmorphism -->
          <Transition name="fade-scale">
            <div
              v-if="openProjectMenuId === project.id"
              class="
                context-dropdown-menu
                absolute right-2 top-12 z-10 w-48
                bg-gray-800/95 backdrop-blur-xl
                border border-gray-700/50
                rounded-xl shadow-2xl
                overflow-hidden
              "
            >
              <button
                @click="startRenameProject(project)"
                class="w-full text-left px-4 py-2.5 hover:bg-gray-700/50 transition-colors flex items-center gap-2"
              >
                <PencilIcon class="w-4 h-4 text-blue-400" />
                <span>{{ $t('sidebar.rename') }}</span>
              </button>
              <button
                @click="openManageFiles(project)"
                class="w-full text-left px-4 py-2.5 hover:bg-gray-700/50 transition-colors flex items-center gap-2"
              >
                <DocumentPlusIcon class="w-4 h-4 text-green-400" />
                <span>{{ $t('sidebar.manageFiles') }}</span>
              </button>
              <button
                @click="deleteProject(project.id)"
                class="w-full text-left px-4 py-2.5 hover:bg-red-900/30 text-red-400 hover:text-red-300 transition-colors flex items-center gap-2 border-t border-gray-700/50"
              >
                <TrashIcon class="w-4 h-4" />
                <span>{{ $t('sidebar.delete') }}</span>
              </button>
            </div>
          </Transition>
        </div>

        <!-- Divider -->
        <div class="my-3 border-t border-gray-700/50"></div>
      </div>

      <!-- Empty State for Projects and Chats -->
      <div v-if="sortedUnassignedChats.length === 0 && sortedProjects.length === 0" class="px-4 py-12 text-center">
        <div class="mb-4 flex justify-center">
          <div class="p-4 rounded-2xl bg-gray-800/50">
            <ChatBubbleLeftRightIcon class="w-12 h-12 text-gray-600" />
          </div>
        </div>
        <h3 class="text-sm font-semibold text-gray-400 mb-2">{{ $t('sidebar.noChats') }}</h3>
        <p class="text-xs text-gray-500 mb-4">
          {{ $t('sidebar.noChatsDesc') }}
        </p>
        <div class="flex flex-col gap-2">
          <button
            @click="handleNewChat"
            class="
              px-4 py-2 rounded-lg
              bg-fleet-orange-600/20 hover:bg-fleet-orange-600/30
              text-fleet-orange-400 text-sm font-medium
              transition-colors
              flex items-center justify-center gap-2
            "
          >
            <PlusCircleIcon class="w-4 h-4" />
            {{ $t('sidebar.newChat') }}
          </button>
        </div>
      </div>

      <!-- Chats Section (unassigned) -->
      <div
        v-for="chat in sortedUnassignedChats"
        :key="'chat-' + chat.id"
        class="relative group mb-1"
      >
        <div
          @click="selectChat(chat)"
          class="
            flex items-center justify-between p-3 rounded-xl cursor-pointer
            transition-all duration-200
            hover:bg-gray-800/80 hover:backdrop-blur-sm
            hover:scale-[1.02] hover:shadow-md
          "
          :class="{
            'bg-fleet-orange-900/30 border border-fleet-orange-500/30': chatStore.currentChat?.id === chat.id,
            'hover:border hover:border-gray-700/50': chatStore.currentChat?.id !== chat.id
          }"
        >
          <div class="flex-1 truncate">
            <div class="flex items-center gap-2">
              <ChatBubbleLeftIcon class="w-4 h-4 text-fleet-orange-400 flex-shrink-0" />
              <span class="font-medium truncate">{{ chat.title }}</span>
            </div>
            <div class="text-xs text-gray-400 ml-6 mt-1 flex items-center gap-1">
              <ClockIcon class="w-3 h-3" />
              {{ formatDate(chat.updatedAt) }}
            </div>
            <div v-if="chat.projectName" class="text-xs text-purple-400 mt-1 ml-6 flex items-center gap-1">
              <FolderIcon class="w-3 h-3" />
              {{ chat.projectName }}
            </div>
          </div>
          <!-- Context Menu Button -->
          <button
            @click.stop="toggleChatMenu(chat.id)"
            class="
              opacity-0 group-hover:opacity-100
              p-1.5 rounded-lg
              text-gray-400 hover:text-white hover:bg-gray-700
              transition-all duration-200
            "
          >
            <EllipsisVerticalIcon class="w-5 h-5" />
          </button>
        </div>

        <!-- Chat Dropdown Menu -->
        <Transition name="fade-scale">
          <div
            v-if="openChatMenuId === chat.id"
            class="
              context-dropdown-menu
              absolute right-2 top-12 z-[100] w-48
              bg-gray-800/95 backdrop-blur-xl
              border border-gray-700/50
              rounded-xl shadow-2xl
              overflow-hidden
            "
          >
            <button
              @click="startRenameChat(chat)"
              class="w-full text-left px-4 py-2.5 hover:bg-gray-700/50 transition-colors flex items-center gap-2"
            >
              <PencilIcon class="w-4 h-4 text-blue-400" />
              <span>{{ $t('sidebar.rename') }}</span>
            </button>
            <button
              @click="openAssignProject(chat)"
              class="w-full text-left px-4 py-2.5 hover:bg-gray-700/50 transition-colors flex items-center gap-2"
            >
              <FolderIcon class="w-4 h-4 text-purple-400" />
              <span>{{ $t('sidebar.assignProject') }}</span>
            </button>
            <button
              @click="deleteChat(chat.id)"
              class="w-full text-left px-4 py-2.5 hover:bg-red-900/30 text-red-400 hover:text-red-300 transition-colors flex items-center gap-2 border-t border-gray-700/50"
            >
              <TrashIcon class="w-4 h-4" />
              <span>{{ $t('sidebar.delete') }}</span>
            </button>
          </div>
        </Transition>
      </div>
    </div>

    <!-- Footer: Stats + Server/User Status + Context -->
    <div class="sidebar-footer p-3 border-t border-gray-700/50 bg-gray-900/50 backdrop-blur-sm flex-shrink-0">
      <div class="text-xs space-y-1.5">
        <!-- Stats Row -->
        <div class="flex justify-between items-center text-gray-400">
          <span class="flex items-center gap-1">
            <FolderIcon class="w-3 h-3" />
            <span>{{ sortedProjects.length }}</span>
          </span>
          <span class="flex items-center gap-1">
            <ChatBubbleLeftRightIcon class="w-3 h-3" />
            <span>{{ chatStore.globalStats.chatCount }}</span>
          </span>
          <span class="flex items-center gap-1">
            <CpuChipIcon class="w-3 h-3" />
            <span>{{ formatNumber(chatStore.globalStats.totalTokens) }}</span>
          </span>
        </div>
        <!-- Server + User: 2-Spalten Layout -->
        <div class="grid grid-cols-2 gap-2 pt-1.5 border-t border-gray-700/30">
          <!-- Server Status -->
          <div class="flex items-center gap-1.5">
            <ServerStackIcon class="w-3.5 h-3.5 text-gray-400" />
            <span
              class="flex items-center gap-1 font-semibold"
              :class="llamaServerOnline ? 'text-green-400' : 'text-red-400'"
              :title="llamaServerOnline ? $t('sidebar.serverRunning') : $t('sidebar.serverNotReachable')"
            >
              <span
                class="w-1.5 h-1.5 rounded-full"
                :class="llamaServerOnline ? 'bg-green-400 animate-pulse' : 'bg-red-400'"
              ></span>
              {{ llamaServerOnline ? $t('sidebar.online') : $t('sidebar.offline') }}
            </span>
          </div>
          <!-- User + Logout -->
          <div class="flex items-center justify-end gap-1">
            <span class="text-gray-400 truncate text-[10px]">{{ authStore.user?.username || '?' }}</span>
            <button
              @click="handleLogout"
              class="p-1 text-gray-500 hover:text-red-400 hover:bg-red-500/10 rounded transition-all flex-shrink-0"
              :title="$t('sidebar.logout')"
            >
              <PowerIcon class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>
        <!-- Model/Expert Context Usage -->
        <div class="pt-1.5 border-t border-gray-700/30">
          <div class="flex items-center justify-between mb-1">
            <span class="text-gray-500 flex items-center gap-1" :title="currentExpert ? `${currentExpert.name} (${currentModelName})` : currentModelName">
              <template v-if="currentExpert">
                <span class="text-sm">{{ currentExpert.avatar || '🤖' }}</span>
                <span class="truncate max-w-[100px]">{{ currentExpert.name }}</span>
              </template>
              <template v-else>
                <DocumentTextIcon class="w-3 h-3" />
                <span class="truncate">{{ truncateModelName(currentModelName) }}</span>
              </template>
            </span>
            <span class="text-gray-400 font-mono text-[10px]">{{ maxContextDisplay }}</span>
          </div>
          <!-- Progress Bar -->
          <div class="w-full h-1.5 bg-gray-700 rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-300"
              :class="contextUsageColor"
              :style="{ width: contextUsagePercent + '%' }"
            ></div>
          </div>
          <div class="flex justify-between text-[10px] text-gray-500 mt-0.5">
            <span>{{ formatNumber(chatStore.contextUsage.totalChatTokens) }} {{ $t('sidebar.tokens') }}</span>
            <span class="font-mono">{{ contextUsagePercent }}%</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Rename Chat Modal -->
    <Transition name="modal">
      <div
        v-if="renamingChat"
        @click="cancelRenameChat"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
      >
        <div @click.stop class="bg-gray-800/95 backdrop-blur-xl border border-gray-700/50 rounded-2xl p-6 w-96 shadow-2xl">
          <h3 class="text-lg font-bold mb-4 flex items-center gap-2">
            <PencilIcon class="w-5 h-5 text-blue-400" />
            {{ $t('sidebar.renameChat') }}
          </h3>
          <input
            v-model="newChatTitle"
            @keydown.enter="confirmRenameChat"
            @keydown.esc="cancelRenameChat"
            type="text"
            class="w-full px-4 py-2.5 bg-gray-700/50 border border-gray-600/50 rounded-xl focus:outline-none focus:ring-2 focus:ring-fleet-orange-500 focus:border-transparent mb-4 transition-all"
            :placeholder="$t('sidebar.newTitle')"
            ref="renameChatInput"
          />
          <div class="flex justify-end gap-2">
            <button
              @click="cancelRenameChat"
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-xl transition-all transform hover:scale-105"
            >
              {{ $t('sidebar.cancel') }}
            </button>
            <button
              @click="confirmRenameChat"
              class="px-4 py-2 bg-gradient-to-r from-fleet-orange-500 to-fleet-orange-600 hover:from-fleet-orange-400 hover:to-fleet-orange-500 rounded-xl transition-all transform hover:scale-105 shadow-lg"
            >
              {{ $t('sidebar.save') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Create/Rename Project Modal -->
    <Transition name="modal">
      <div
        v-if="showProjectModal"
        @click="cancelProjectModal"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
      >
        <div @click.stop class="bg-gray-800/95 backdrop-blur-xl border border-gray-700/50 rounded-2xl p-6 w-96 shadow-2xl">
          <h3 class="text-lg font-bold mb-4 flex items-center gap-2">
            <FolderPlusIcon v-if="!editingProject" class="w-5 h-5 text-purple-400" />
            <PencilIcon v-else class="w-5 h-5 text-blue-400" />
            {{ editingProject ? $t('sidebar.renameProject') : $t('sidebar.newProject') }}
          </h3>
          <input
            v-model="projectName"
            @keydown.enter="confirmProjectModal"
            @keydown.esc="cancelProjectModal"
            type="text"
            class="w-full px-4 py-2.5 bg-gray-700/50 border border-gray-600/50 rounded-xl focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent mb-3 transition-all"
            :placeholder="$t('sidebar.projectName')"
            ref="projectNameInput"
          />
          <textarea
            v-model="projectDescription"
            rows="3"
            class="w-full px-4 py-2.5 bg-gray-700/50 border border-gray-600/50 rounded-xl focus:outline-none focus:ring-2 focus:ring-purple-500 focus:border-transparent mb-4 transition-all resize-none"
            :placeholder="$t('sidebar.projectDesc')"
          ></textarea>
          <div class="flex justify-end gap-2">
            <button
              @click="cancelProjectModal"
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-xl transition-all transform hover:scale-105"
            >
              {{ $t('sidebar.cancel') }}
            </button>
            <button
              @click="confirmProjectModal"
              class="px-4 py-2 bg-gradient-to-r from-purple-600 to-purple-700 hover:from-purple-500 hover:to-purple-600 rounded-xl transition-all transform hover:scale-105 shadow-lg"
            >
              {{ editingProject ? $t('sidebar.save') : $t('sidebar.create') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Assign Project Modal -->
    <Transition name="modal">
      <div
        v-if="showAssignProjectModal"
        @click="showAssignProjectModal = false"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50"
      >
        <div @click.stop class="bg-gray-800/95 backdrop-blur-xl border border-gray-700/50 rounded-2xl p-6 w-[480px] shadow-2xl">
          <h3 class="text-lg font-bold mb-4 flex items-center gap-2">
            <FolderIcon class="w-5 h-5 text-purple-400" />
            {{ $t('sidebar.assignProject') }}
          </h3>
          <div class="mb-4">
            <label class="block text-sm text-gray-400 mb-3">{{ $t('sidebar.chat') }}: <span class="text-white font-medium">{{ assigningChat?.title }}</span></label>

            <!-- ListBox with Projects -->
            <div class="space-y-2 max-h-[400px] overflow-y-auto pr-2 custom-scrollbar">
              <!-- "Kein Projekt" Option -->
              <button
                @click="selectedProjectId = null"
                class="
                  w-full text-left px-4 py-3 rounded-xl
                  transition-all duration-200
                  flex items-center gap-3
                  border-2
                "
                :class="selectedProjectId === null
                  ? 'bg-purple-900/40 border-purple-500/50 shadow-lg shadow-purple-500/20'
                  : 'bg-gray-700/30 border-gray-600/30 hover:bg-gray-700/50 hover:border-gray-500/50'"
              >
                <div class="flex-shrink-0 p-2 rounded-lg bg-gray-600/50">
                  <XMarkIcon class="w-5 h-5 text-gray-400" />
                </div>
                <div class="flex-1">
                  <div class="font-medium text-white">{{ $t('sidebar.noProject') }}</div>
                  <div class="text-xs text-gray-400">{{ $t('sidebar.noProjectDesc') }}</div>
                </div>
                <div v-if="selectedProjectId === null" class="flex-shrink-0">
                  <div class="w-5 h-5 rounded-full bg-purple-500 flex items-center justify-center">
                    <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                  </div>
                </div>
              </button>

              <!-- Project Options -->
              <button
                v-for="project in sortedProjects"
                :key="project.id"
                @click="selectedProjectId = project.id"
                class="
                  w-full text-left px-4 py-3 rounded-xl
                  transition-all duration-200
                  flex items-center gap-3
                  border-2
                "
                :class="selectedProjectId === project.id
                  ? 'bg-purple-900/40 border-purple-500/50 shadow-lg shadow-purple-500/20'
                  : 'bg-gray-700/30 border-gray-600/30 hover:bg-gray-700/50 hover:border-gray-500/50'"
              >
                <div class="flex-shrink-0 p-2 rounded-lg bg-gradient-to-br from-purple-500/20 to-indigo-500/20">
                  <FolderIcon class="w-5 h-5 text-purple-400" />
                </div>
                <div class="flex-1 min-w-0">
                  <div class="font-medium text-white truncate">{{ project.name }}</div>
                  <div v-if="project.description" class="text-xs text-gray-400 truncate">{{ project.description }}</div>
                  <div class="flex items-center gap-3 mt-1 text-xs text-gray-500">
                    <span class="flex items-center gap-1">
                      <ChatBubbleLeftRightIcon class="w-3 h-3" />
                      {{ getProjectChatCount(project.id) }} {{ $t('sidebar.chats') }}
                    </span>
                    <span class="flex items-center gap-1">
                      <DocumentTextIcon class="w-3 h-3" />
                      {{ project.contextFiles?.length || 0 }} {{ $t('sidebar.files') }}
                    </span>
                  </div>
                </div>
                <div v-if="selectedProjectId === project.id" class="flex-shrink-0">
                  <div class="w-5 h-5 rounded-full bg-purple-500 flex items-center justify-center">
                    <svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                  </div>
                </div>
              </button>
            </div>
          </div>
          <div class="flex justify-end gap-2 pt-4 border-t border-gray-700/50">
            <button
              @click="showAssignProjectModal = false"
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-xl transition-all transform hover:scale-105"
            >
              {{ $t('sidebar.cancel') }}
            </button>
            <button
              @click="confirmAssignProject"
              class="px-4 py-2 bg-gradient-to-r from-fleet-orange-500 to-fleet-orange-600 hover:from-fleet-orange-400 hover:to-fleet-orange-500 rounded-xl transition-all transform hover:scale-105 shadow-lg"
            >
              {{ $t('sidebar.assign') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Manage Files Modal -->
    <Transition name="modal">
      <div
        v-if="showManageFilesModal"
        @click="showManageFilesModal = false"
        class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4"
      >
        <div @click.stop class="bg-gray-800/95 backdrop-blur-xl border border-gray-700/50 rounded-2xl p-6 w-[600px] max-h-[80vh] overflow-y-auto shadow-2xl custom-scrollbar">
          <h3 class="text-lg font-bold mb-4 flex items-center gap-2">
            <DocumentTextIcon class="w-5 h-5 text-green-400" />
            {{ $t('sidebar.contextFiles') }}: {{ managingFilesProject?.name }}
          </h3>

          <!-- Upload Area -->
          <div class="mb-4">
            <input
              type="file"
              @change="handleFileUpload"
              accept=".txt,.md,.java,.js,.py,.json,.xml,.yaml,.yml"
              class="hidden"
              ref="fileInput"
            />
            <button
              @click="$refs.fileInput.click()"
              class="
                w-full px-4 py-3 rounded-xl
                bg-gradient-to-r from-purple-600 to-purple-700
                hover:from-purple-500 hover:to-purple-600
                transition-all transform hover:scale-[1.02]
                shadow-lg
                flex items-center justify-center gap-2
              "
            >
              <ArrowUpTrayIcon class="w-5 h-5" />
              {{ $t('sidebar.uploadFile') }}
            </button>
          </div>

          <!-- File List -->
          <div v-if="managingFilesProject?.contextFiles?.length > 0" class="space-y-2">
            <div
              v-for="file in managingFilesProject.contextFiles"
              :key="file.id"
              class="bg-gray-700/50 rounded-xl p-3 flex items-center justify-between hover:bg-gray-700/70 transition-all group"
            >
              <div class="flex-1 truncate flex items-center gap-3">
                <div class="p-2 rounded-lg bg-green-500/20">
                  <DocumentTextIcon class="w-4 h-4 text-green-400" />
                </div>
                <div class="flex-1 truncate">
                  <div class="font-medium truncate">{{ file.filename }}</div>
                  <div class="text-xs text-gray-400 flex items-center gap-2">
                    <span>{{ formatFileSize(file.size) }}</span>
                    <span>·</span>
                    <span class="flex items-center gap-1">
                      <CpuChipIcon class="w-3 h-3" />
                      ~{{ file.estimatedTokens || Math.floor(file.size / 4) }} tokens
                    </span>
                  </div>
                </div>
              </div>
              <button
                @click="deleteContextFile(file.id)"
                class="p-2 text-red-400 hover:text-red-300 hover:bg-red-900/30 rounded-lg transition-all opacity-0 group-hover:opacity-100"
              >
                <TrashIcon class="w-4 h-4" />
              </button>
            </div>
          </div>
          <div v-else class="text-center py-12">
            <div class="mb-4 flex justify-center">
              <div class="p-4 rounded-2xl bg-gray-700/30">
                <DocumentTextIcon class="w-12 h-12 text-gray-600" />
              </div>
            </div>
            <p class="text-gray-400 text-sm">{{ $t('sidebar.noFilesUploaded') }}</p>
          </div>

          <div class="flex justify-end mt-4">
            <button
              @click="showManageFilesModal = false"
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-xl transition-all transform hover:scale-105"
            >
              {{ $t('sidebar.close') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { useChatStore } from '../stores/chatStore'
import { useSettingsStore } from '../stores/settingsStore'
import { ref, computed, onMounted, nextTick, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
import {
  FolderIcon,
  FolderPlusIcon,
  PlusCircleIcon,
  ChatBubbleLeftRightIcon,
  ChatBubbleLeftIcon,
  DocumentTextIcon,
  DocumentPlusIcon,
  EllipsisVerticalIcon,
  PencilIcon,
  TrashIcon,
  ClockIcon,
  CpuChipIcon,
  ArrowUpTrayIcon,
  XMarkIcon,
  ServerStackIcon,
  PowerIcon
} from '@heroicons/vue/24/outline'
import { useAuthStore } from '../stores/authStore'
import api from '../services/api'
import { useToast } from '../composables/useToast'
import { useConfirmDialog } from '../composables/useConfirmDialog'
import { formatDate, formatNumber, formatFileSize } from '../composables/useFormatters'

const { success, error: errorToast } = useToast()
const { confirm, confirmDelete } = useConfirmDialog()
const router = useRouter()
const chatStore = useChatStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()

// Sidebar Resize
const SIDEBAR_WIDTH_KEY = 'fleet-navigator-sidebar-width'
const DEFAULT_SIDEBAR_WIDTH = 256  // 16rem = 256px
const MIN_SIDEBAR_WIDTH = 180
const MAX_SIDEBAR_WIDTH = 400

const sidebarWidth = ref(loadSavedWidth())
const isResizing = ref(false)

function loadSavedWidth() {
  try {
    const saved = localStorage.getItem(SIDEBAR_WIDTH_KEY)
    if (saved) {
      const width = parseInt(saved, 10)
      if (width >= MIN_SIDEBAR_WIDTH && width <= MAX_SIDEBAR_WIDTH) {
        return width
      }
    }
  } catch (e) {}
  return DEFAULT_SIDEBAR_WIDTH
}

function saveSidebarWidth(width) {
  try {
    localStorage.setItem(SIDEBAR_WIDTH_KEY, String(width))
  } catch (e) {}
}

function startResize(event) {
  event.preventDefault()
  isResizing.value = true
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', stopResize)
  document.body.style.cursor = 'ew-resize'
  document.body.style.userSelect = 'none'
}

function onResize(event) {
  if (!isResizing.value) return

  let newWidth = event.clientX

  // Clamp to min/max
  if (newWidth < MIN_SIDEBAR_WIDTH) {
    newWidth = MIN_SIDEBAR_WIDTH
  } else if (newWidth > MAX_SIDEBAR_WIDTH) {
    newWidth = MAX_SIDEBAR_WIDTH
  }

  sidebarWidth.value = newWidth
}

function stopResize() {
  isResizing.value = false
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', stopResize)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''

  // Save width to localStorage
  saveSidebarWidth(sidebarWidth.value)
}

// Chat-related
const openChatMenuId = ref(null)
const renamingChat = ref(null)
const newChatTitle = ref('')
const renameChatInput = ref(null)

// Project-related
const projects = ref([])
const selectedProject = ref(null)
const openProjectMenuId = ref(null)
const showProjectModal = ref(false)
const editingProject = ref(null)
const projectName = ref('')
const projectDescription = ref('')
const projectNameInput = ref(null)

// Assign project
const showAssignProjectModal = ref(false)
const assigningChat = ref(null)
const selectedProjectId = ref(null)

// Manage files
const showManageFilesModal = ref(false)
const managingFilesProject = ref(null)
const fileInput = ref(null)

// llama-server Status
const llamaServerOnline = ref(false)
let healthCheckInterval = null

// Sorted projects by last update
const sortedProjects = computed(() => {
  return [...projects.value].sort((a, b) => {
    const dateA = new Date(a.updatedAt || a.createdAt)
    const dateB = new Date(b.updatedAt || b.createdAt)
    return dateB - dateA
  })
})

// Only show chats that are NOT assigned to a project
const unassignedChats = computed(() => {
  return (chatStore.chats || []).filter(chat => !chat.projectId)
})

// Sort unassigned chats in reverse order (newest first)
const sortedUnassignedChats = computed(() => {
  return [...unassignedChats.value].sort((a, b) => {
    const dateA = new Date(a.updatedAt)
    const dateB = new Date(b.updatedAt)
    return dateB - dateA
  })
})

// Context Usage computed properties
const contextUsagePercent = computed(() => {
  const { totalChatTokens, maxContextTokens } = chatStore.contextUsage
  if (!maxContextTokens || maxContextTokens === 0) return 0
  const percent = Math.round((totalChatTokens / maxContextTokens) * 100)
  return Math.min(percent, 100) // Cap at 100%
})

const contextUsageColor = computed(() => {
  const percent = contextUsagePercent.value
  if (percent >= 90) return 'bg-red-500'
  if (percent >= 70) return 'bg-yellow-500'
  if (percent >= 50) return 'bg-fleet-orange-500'
  return 'bg-green-500'
})

// Display max context size (from backend or default 4096)
const maxContextDisplay = computed(() => {
  const max = chatStore.contextUsage.maxContextTokens
  if (max && max > 0) {
    // Format as "4K", "8K", "32K", "128K" etc.
    if (max >= 1000) {
      return Math.round(max / 1000) + 'K ctx'
    }
    return max + ' ctx'
  }
  // Default: llama-server uses 65536 (64K)
  return '64K ctx'
})

// Get current expert info (if selected)
const currentExpert = computed(() => {
  if (chatStore.selectedExpertId) {
    return chatStore.getExpertById(chatStore.selectedExpertId)
  }
  return null
})

// Display name: Expert name if selected, otherwise model name
const displayName = computed(() => {
  const expert = currentExpert.value
  if (expert) {
    return `${expert.avatar || '🤖'} ${expert.name}`
  }
  return chatStore.selectedModel || t('sidebar.noModel')
})

// Get current model name (from expert or direct selection)
const currentModelName = computed(() => {
  // If expert is selected, get model from expert
  if (chatStore.selectedExpertId) {
    const expert = chatStore.getExpertById(chatStore.selectedExpertId)
    // API returns "model" field, not "baseModel"
    if (expert && (expert.model || expert.baseModel)) {
      return expert.model || expert.baseModel
    }
  }
  // Otherwise use selected model
  return chatStore.selectedModel || t('sidebar.noModel')
})

// Truncate model name for display
function truncateModelName(name) {
  if (!name) return t('sidebar.noModel')
  // Remove common suffixes and truncate
  const cleanName = name.replace(/:latest$/, '')
  if (cleanName.length > 16) {
    return cleanName.substring(0, 14) + '...'
  }
  return cleanName
}

const emit = defineEmits(['select-project', 'new-chat'])

onMounted(async () => {
  await chatStore.loadChats()
  await chatStore.loadGlobalStats()
  await loadProjects()
  document.addEventListener('click', closeMenus)
  document.addEventListener('keydown', handleKeyDown)

  // Initial llama-server health check
  checkLlamaServerHealth()
  // Check every 10 seconds
  healthCheckInterval = setInterval(checkLlamaServerHealth, 10000)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenus)
  document.removeEventListener('keydown', handleKeyDown)
  // Stop health check interval
  if (healthCheckInterval) {
    clearInterval(healthCheckInterval)
    healthCheckInterval = null
  }
})

// llama-server health check
async function checkLlamaServerHealth() {
  try {
    const result = await api.checkLlamaServerHealth(2026)
    llamaServerOnline.value = result.online === true
  } catch (err) {
    // Bei Netzwerkfehler: offline
    llamaServerOnline.value = false
  }
}

// Handle keyboard shortcuts
function handleKeyDown(event) {
  // Check if Delete or Entf key is pressed
  if (event.key === 'Delete' || event.key === 'Entf') {
    // Don't trigger if user is typing in an input field or textarea
    const activeElement = document.activeElement
    const isInInputField = activeElement && (
      activeElement.tagName === 'INPUT' ||
      activeElement.tagName === 'TEXTAREA' ||
      activeElement.isContentEditable ||
      activeElement.closest('[role="dialog"]') ||
      activeElement.closest('.modal') ||
      activeElement.closest('[class*="Modal"]')
    )

    if (isInInputField) {
      return // Let the default delete behavior happen in input fields
    }

    // Only trigger if a chat is currently selected
    if (chatStore.currentChat) {
      event.preventDefault()
      deleteChat(chatStore.currentChat.id)
    }
  }
}

function closeMenus() {
  openChatMenuId.value = null
  openProjectMenuId.value = null
}

function toggleChatMenu(chatId) {
  openChatMenuId.value = openChatMenuId.value === chatId ? null : chatId
}

function toggleProjectMenu(projectId) {
  openProjectMenuId.value = openProjectMenuId.value === projectId ? null : projectId
}

// Get chat count for a project
function getProjectChatCount(projectId) {
  return (chatStore.chats || []).filter(chat => chat.projectId === projectId).length
}

// Handle new chat button
function handleNewChat() {
  emit('new-chat')
}

// Handle new project button
function handleNewProject() {
  showProjectModal.value = true
  editingProject.value = null
  projectName.value = ''
  projectDescription.value = ''
  nextTick(() => {
    if (projectNameInput.value) {
      projectNameInput.value.focus()
    }
  })
}

// Chat functions
async function selectChat(chat) {
  await chatStore.loadChatHistory(chat.id)
}

async function startRenameChat(chat) {
  openChatMenuId.value = null
  renamingChat.value = chat
  newChatTitle.value = chat.title
  await nextTick()
  if (renameChatInput.value) {
    renameChatInput.value.focus()
    renameChatInput.value.select()
  }
}

async function confirmRenameChat() {
  if (!renamingChat.value || !newChatTitle.value.trim()) return

  try {
    await chatStore.renameChat(renamingChat.value.id, newChatTitle.value.trim())
    success(t('success.saved'))
    renamingChat.value = null
    newChatTitle.value = ''
  } catch (err) {
    errorToast(t('errors.saveFailed'))
  }
}

function cancelRenameChat() {
  renamingChat.value = null
  newChatTitle.value = ''
}

async function deleteChat(chatId) {
  openChatMenuId.value = null

  // Find the chat to get its title
  const chat = (chatStore.chats || []).find(c => c.id === chatId)
  const chatTitle = chat ? chat.title : 'Diesen Chat'

  const confirmed = await confirmDelete(chatTitle, 'Diese Aktion kann nicht rückgängig gemacht werden.')
  if (!confirmed) return

  try {
    console.log('[Sidebar] Deleting chat:', chatId)
    await chatStore.deleteChat(chatId)
    success(t('success.deleted'))
    console.log('[Sidebar] Chat deleted successfully, remaining chats:', (chatStore.chats || []).length)

    // Force reload the chat list from database to ensure sync
    await chatStore.loadChats()
    console.log('[Sidebar] Chats reloaded from DB, count:', (chatStore.chats || []).length)
  } catch (err) {
    console.error('[Sidebar] Delete failed:', err)
    errorToast(t('errors.deleteFailed'))
  }
}

// Assign project to chat
function openAssignProject(chat) {
  openChatMenuId.value = null
  assigningChat.value = chat
  selectedProjectId.value = chat.projectId || null
  showAssignProjectModal.value = true
}

async function confirmAssignProject() {
  if (!assigningChat.value) return

  try {
    if (selectedProjectId.value) {
      await api.assignChatToProject(assigningChat.value.id, selectedProjectId.value)
      success(t('success.saved'))
    } else {
      await api.unassignChatFromProject(assigningChat.value.id)
      success(t('success.saved'))
    }
    await chatStore.loadChats()
    await loadProjects()
    showAssignProjectModal.value = false
    assigningChat.value = null
  } catch (err) {
    errorToast(t('errors.saveFailed'))
    console.error(err)
  }
}

// Project functions
async function loadProjects() {
  try {
    projects.value = await api.getAllProjects()
  } catch (err) {
    console.error('Failed to load projects:', err)
  }
}

function selectProject(project) {
  selectedProject.value = project
  emit('select-project', project)
}

async function startRenameProject(project) {
  openProjectMenuId.value = null
  editingProject.value = project
  projectName.value = project.name
  projectDescription.value = project.description || ''
  showProjectModal.value = true
  await nextTick()
  if (projectNameInput.value) {
    projectNameInput.value.focus()
    projectNameInput.value.select()
  }
}

async function confirmProjectModal() {
  if (!projectName.value.trim()) return

  try {
    if (editingProject.value) {
      await api.updateProject(editingProject.value.id, {
        name: projectName.value.trim(),
        description: projectDescription.value.trim()
      })
      success(t('success.saved'))
    } else {
      await api.createProject({
        name: projectName.value.trim(),
        description: projectDescription.value.trim()
      })
      success(t('success.saved'))
    }
    await loadProjects()
    showProjectModal.value = false
    editingProject.value = null
    projectName.value = ''
    projectDescription.value = ''
  } catch (err) {
    errorToast(t('errors.saveFailed'))
    console.error(err)
  }
}

function cancelProjectModal() {
  showProjectModal.value = false
  editingProject.value = null
  projectName.value = ''
  projectDescription.value = ''
}

async function deleteProject(projectId) {
  openProjectMenuId.value = null
  const project = projects.value.find(p => p.id === projectId)
  const confirmed = await confirmDelete(project?.name || 'Dieses Projekt', 'Alle Chats werden vom Projekt getrennt.')
  if (!confirmed) return

  try {
    await api.deleteProject(projectId)
    await loadProjects()
    await chatStore.loadChats()
    success(t('success.deleted'))
  } catch (err) {
    errorToast(t('errors.deleteFailed'))
    console.error(err)
  }
}

// File management
function openManageFiles(project) {
  openProjectMenuId.value = null
  managingFilesProject.value = project
  showManageFilesModal.value = true
}

async function handleFileUpload(event) {
  const file = event.target.files[0]
  if (!file) return

  try {
    const reader = new FileReader()
    reader.onload = async (e) => {
      const content = e.target.result

      await api.uploadContextFile({
        projectId: managingFilesProject.value.id,
        filename: file.name,
        content: content,
        fileType: file.type || 'text/plain',
        size: file.size
      })

      await loadProjects()
      managingFilesProject.value = projects.value.find(p => p.id === managingFilesProject.value.id)
      success(t('success.uploaded'))
    }
    reader.readAsText(file)
  } catch (err) {
    errorToast(t('errors.uploadFailed'))
    console.error(err)
  }

  event.target.value = ''
}

async function deleteContextFile(fileId) {
  const confirmed = await confirmDelete('Diese Datei')
  if (!confirmed) return

  try {
    await api.deleteContextFile(fileId)
    await loadProjects()
    managingFilesProject.value = projects.value.find(p => p.id === managingFilesProject.value.id)
    success(t('success.deleted'))
  } catch (err) {
    errorToast(t('errors.deleteFailed'))
    console.error(err)
  }
}

// Logout-Funktion
async function handleLogout() {
  const confirmed = await confirm({
    title: t('sidebar.logout'),
    message: t('sidebar.logoutConfirm'),
    type: 'question',
    confirmText: t('sidebar.logout'),
    cancelText: t('sidebar.cancel')
  })
  if (confirmed) {
    await authStore.logout()
    // Nach Logout zur Login-Seite weiterleiten
    router.push({ name: 'login' })
  }
}

// Utility functions importiert aus useFormatters.js
</script>

<style scoped>
/* Custom Scrollbar - Grau */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #5A6268;
  border-radius: 3px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #6C757D;
}

/* Resize Handle Indicator */
.sidebar-nav:hover .resize-handle {
  opacity: 1;
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

/* Fade Scale Transition for Dropdowns */
.fade-scale-enter-active,
.fade-scale-leave-active {
  transition: all 0.2s ease;
}

.fade-scale-enter-from,
.fade-scale-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-5px);
}
</style>
