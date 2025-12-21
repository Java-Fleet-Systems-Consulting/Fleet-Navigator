<template>
  <Transition name="modal">
    <div v-if="mate" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div class="
        bg-gray-900
        rounded-2xl shadow-2xl
        w-full max-w-6xl max-h-[90vh]
        border border-gray-700/50
        flex flex-col
        transform transition-all duration-300
      ">
        <!-- Header -->
        <div class="
          flex items-center justify-between p-6
          bg-gradient-to-r from-fleet-orange-500/20 to-orange-600/20
          border-b border-gray-700/50
          rounded-t-2xl
        ">
          <div class="flex items-center gap-4">
            <div class="p-3 rounded-xl bg-gradient-to-br from-fleet-orange-500 to-orange-600 shadow-lg">
              <ServerIcon class="w-7 h-7 text-white" />
            </div>
            <div>
              <h2 class="text-2xl font-bold text-white">{{ mate.name }}</h2>
              <div class="flex items-center gap-3 mt-1">
                <p class="text-sm text-gray-400">{{ mate.description }}</p>
                <span
                  class="px-2 py-1 rounded-full text-xs font-semibold flex items-center gap-1"
                  :class="mate.status === 'ONLINE'
                    ? 'bg-green-500/20 text-green-400 border border-green-500/30'
                    : 'bg-red-500/20 text-red-400 border border-red-500/30'"
                >
                  <component :is="mate.status === 'ONLINE' ? CheckCircleIcon : XCircleIcon" class="w-3 h-3" />
                  {{ mate.status }}
                </span>
              </div>
            </div>
          </div>
          <button
            @click="$emit('close')"
            class="
              p-2 rounded-lg
              text-gray-400 hover:text-gray-300
              hover:bg-gray-800
              transition-all duration-200
              transform hover:scale-110 active:scale-95
            "
          >
            <XMarkIcon class="w-6 h-6" />
          </button>
        </div>

        <!-- Tabs -->
        <div class="flex gap-2 px-6 pt-4 border-b border-gray-700/50 bg-gray-900/50">
          <!-- OS-Mate Tabs: Hardware, Terminal, Remote -->
          <button
            v-if="isOsMate"
            @click="activeTab = 'hardware'"
            class="
              px-4 py-3 rounded-t-lg font-semibold text-sm
              transition-all duration-200
              flex items-center gap-2
            "
            :class="activeTab === 'hardware'
              ? 'bg-gray-800 text-white border-t-2 border-fleet-orange-500'
              : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800/50'"
          >
            <CpuChipIcon class="w-5 h-5" />
            {{ $t('mateDetail.tabs.hardware') }}
          </button>
          <button
            v-if="isOsMate"
            @click="activeTab = 'terminal'"
            class="
              px-4 py-3 rounded-t-lg font-semibold text-sm
              transition-all duration-200
              flex items-center gap-2
            "
            :class="activeTab === 'terminal'
              ? 'bg-gray-800 text-white border-t-2 border-fleet-orange-500'
              : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800/50'"
          >
            <SparklesIcon class="w-5 h-5" />
            {{ $t('mateDetail.tabs.logAnalysis') }}
          </button>
          <button
            v-if="isOsMate"
            @click="activeTab = 'remote'"
            class="
              px-4 py-3 rounded-t-lg font-semibold text-sm
              transition-all duration-200
              flex items-center gap-2
            "
            :class="activeTab === 'remote'
              ? 'bg-gray-800 text-white border-t-2 border-fleet-orange-500'
              : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800/50'"
          >
            <CommandLineIcon class="w-5 h-5" />
            {{ $t('mateDetail.tabs.remoteTerminal') }}
          </button>
          <!-- KI-Einstellungen Tab: Immer sichtbar -->
          <button
            @click="activeTab = 'ai-config'"
            class="
              px-4 py-3 rounded-t-lg font-semibold text-sm
              transition-all duration-200
              flex items-center gap-2
            "
            :class="activeTab === 'ai-config'
              ? 'bg-gray-800 text-white border-t-2 border-fleet-orange-500'
              : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800/50'"
          >
            <CogIcon class="w-5 h-5" />
            {{ $t('mateDetail.tabs.aiSettings') }}
          </button>
        </div>

        <!-- Content Area -->
        <div class="flex-1 overflow-y-auto p-6">
          <!-- Hardware Tab (nur OS-Mates) -->
          <div v-if="isOsMate && activeTab === 'hardware'">
            <div v-if="stats && mate.status === 'ONLINE'" class="space-y-4">
              <!-- System Info -->
              <div class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 p-5 rounded-xl border border-gray-700/50">
                <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
                  <ComputerDesktopIcon class="w-5 h-5 text-blue-400" />
                  {{ $t('mateDetail.hardware.systemInfo') }}
                </h3>
                <div class="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.hostname') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ stats.system?.hostname || 'N/A' }}</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.os') }}:</span>
                    <span class="text-white ml-2 font-medium">
                      {{ (stats.system?.platform || 'Linux').charAt(0).toUpperCase() + (stats.system?.platform || 'Linux').slice(1) }}
                      {{ stats.system?.platform_version || '' }}
                    </span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.kernel') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ stats.system?.kernel_version || 'N/A' }}</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.uptime') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ formatUptime(stats.system?.uptime) }}</span>
                  </div>
                </div>
              </div>

              <!-- CPU -->
              <div class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 p-5 rounded-xl border border-gray-700/50">
                <h3 class="text-lg font-semibold text-white mb-4">{{ $t('mateDetail.hardware.cpu') }}</h3>
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4 text-sm">
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.model') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ shortenCPUModel(stats.cpu?.model) }}</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.cores') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ stats.cpu?.cores || 0 }}</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.clock') }}:</span>
                    <span class="text-white ml-2 font-medium">{{ stats.cpu?.mhz?.toFixed(0) || 0 }} MHz</span>
                  </div>
                  <div>
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.usage') }}:</span>
                    <span class="text-white ml-2 font-bold" :class="getCPUColor(stats.cpu?.usage_percent)">
                      {{ stats.cpu?.usage_percent?.toFixed(1) || 0 }}%
                    </span>
                  </div>
                </div>

                <!-- Per-Core CPU Usage -->
                <div v-if="stats.cpu?.per_core && stats.cpu.per_core.length > 0" class="space-y-2">
                  <h4 class="text-sm font-semibold text-gray-300 mb-2">{{ $t('mateDetail.hardware.perCoreUsage') }}</h4>
                  <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
                    <div v-for="(usage, index) in stats.cpu.per_core" :key="index" class="flex items-center gap-2">
                      <span class="text-xs text-gray-400 w-16">Core {{ index }}</span>
                      <div class="flex-1 h-6 bg-gray-700 rounded-lg overflow-hidden">
                        <div
                          class="h-full transition-all duration-300"
                          :class="getCPUBarColor(usage)"
                          :style="{ width: usage + '%' }"
                        />
                      </div>
                      <span class="text-xs font-semibold w-12 text-right" :class="getCPUColor(usage)">
                        {{ usage?.toFixed(1) }}%
                      </span>
                      <span
                        v-if="getCoreTemp(index)"
                        class="text-xs font-semibold px-2 py-0.5 rounded-full w-12 text-center"
                        :class="getTempBadgeColor(getCoreTemp(index))"
                      >
                        {{ getCoreTemp(index)?.toFixed(0) }}°C
                      </span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Memory -->
              <div class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 p-5 rounded-xl border border-gray-700/50">
                <h3 class="text-lg font-semibold text-white mb-4">{{ $t('mateDetail.hardware.ram') }}</h3>
                <div class="space-y-3">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-gray-400">
                      {{ $t('mateDetail.hardware.used') }}: {{ (stats.memory?.used / 1024 / 1024 / 1024).toFixed(1) }} GB /
                      {{ (stats.memory?.total / 1024 / 1024 / 1024).toFixed(1) }} GB
                    </span>
                    <span class="font-bold" :class="getRAMColor(stats.memory?.used_percent)">
                      {{ stats.memory?.used_percent?.toFixed(1) }}%
                    </span>
                  </div>
                  <div class="h-6 bg-gray-700 rounded-lg overflow-hidden">
                    <div
                      class="h-full transition-all duration-300"
                      :class="getRAMBarColor(stats.memory?.used_percent)"
                      :style="{ width: stats.memory?.used_percent + '%' }"
                    />
                  </div>
                </div>
              </div>

              <!-- GPU -->
              <div v-if="stats.gpu && stats.gpu.length > 0" v-for="gpu in stats.gpu" :key="gpu.index"
                   class="bg-gradient-to-br from-purple-900/30 to-pink-900/30 p-5 rounded-xl border border-purple-500/30">
                <div class="flex items-center justify-between mb-4">
                  <h3 class="text-lg font-semibold text-white flex items-center gap-2">
                    <svg class="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                    </svg>
                    GPU {{ gpu.index }}
                  </h3>
                  <span class="text-xs px-2 py-1 rounded-full bg-purple-500/20 text-purple-300 border border-purple-500/30">
                    {{ gpu.temperature?.toFixed(0) }}°C
                  </span>
                </div>

                <!-- GPU Model -->
                <div class="text-sm text-gray-400 mb-4">
                  {{ gpu.name }}
                </div>

                <!-- GPU Utilization -->
                <div class="space-y-3 mb-4">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-gray-400">{{ $t('mateDetail.hardware.gpuUsage') }}</span>
                    <span class="font-bold" :class="getGPUColor(gpu.utilization_gpu)">
                      {{ gpu.utilization_gpu?.toFixed(1) }}%
                    </span>
                  </div>
                  <div class="h-6 bg-gray-700 rounded-lg overflow-hidden">
                    <div
                      class="h-full transition-all duration-300"
                      :class="getGPUBarColor(gpu.utilization_gpu)"
                      :style="{ width: gpu.utilization_gpu + '%' }"
                    />
                  </div>
                </div>

                <!-- VRAM -->
                <div class="space-y-3">
                  <div class="flex items-center justify-between text-sm">
                    <span class="text-gray-400">
                      VRAM: {{ (gpu.memory_used / 1024).toFixed(1) }} GB / {{ (gpu.memory_total / 1024).toFixed(1) }} GB
                    </span>
                    <span class="font-bold" :class="getVRAMColor(gpu.memory_used_percent)">
                      {{ gpu.memory_used_percent?.toFixed(1) }}%
                    </span>
                  </div>
                  <div class="h-6 bg-gray-700 rounded-lg overflow-hidden">
                    <div
                      class="h-full transition-all duration-300"
                      :class="getVRAMBarColor(gpu.memory_used_percent)"
                      :style="{ width: gpu.memory_used_percent + '%' }"
                    />
                  </div>
                </div>
              </div>

              <!-- Temperature -->
              <div v-if="stats.temperature?.cpu_package" class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 p-5 rounded-xl border border-gray-700/50">
                <h3 class="text-lg font-semibold text-white mb-4">{{ $t('mateDetail.hardware.cpuTemp') }}</h3>
                <div class="flex items-center gap-3">
                  <FireIcon class="w-6 h-6" :class="getTempIconColor(stats.temperature.cpu_package)" />
                  <span
                    class="text-3xl font-bold px-4 py-2 rounded-lg"
                    :class="getTempBadgeColor(stats.temperature.cpu_package)"
                  >
                    {{ stats.temperature.cpu_package?.toFixed(1) }}°C
                  </span>
                </div>
              </div>
            </div>
            <div v-else-if="mate.status === 'OFFLINE'" class="
              bg-gradient-to-br from-red-500/10 to-rose-500/10
              border border-red-500/30
              p-12 rounded-xl
              text-center
            ">
              <XCircleIcon class="w-16 h-16 text-red-400 mx-auto mb-4" />
              <h3 class="text-xl font-semibold text-red-400 mb-2">{{ $t('mateDetail.hardware.mateOffline') }}</h3>
              <p class="text-sm text-gray-400">
                {{ $t('mateDetail.hardware.noDataAvailable') }}
              </p>
            </div>
            <div v-else class="
              bg-gradient-to-br from-gray-800/50 to-gray-900/50
              p-12 rounded-xl
              text-center
            ">
              <ArrowPathIcon class="w-12 h-12 text-gray-500 mx-auto mb-4 animate-spin" />
              <p class="text-sm text-gray-400">{{ $t('mateDetail.hardware.loadingData') }}</p>
            </div>
          </div>

          <!-- Terminal Tab (AI Log Analysis) - nur OS-Mates -->
          <div v-if="isOsMate && activeTab === 'terminal'" class="space-y-4">
            <!-- Log Analysis Form -->
            <div class="bg-gray-800/50 p-4 rounded-xl border border-gray-700/50">
              <h4 class="text-sm font-semibold text-gray-300 mb-3 flex items-center gap-2">
                <SparklesIcon class="w-4 h-4 text-yellow-400" />
                {{ $t('mateDetail.logAnalysis.title') }}
              </h4>

              <div class="space-y-3">
                <!-- Log Path -->
                <div>
                  <label class="text-xs text-gray-400 block mb-1">{{ $t('mateDetail.logAnalysis.logFile') }}</label>
                  <select v-model="logAnalysis.path" class="w-full px-3 py-2 bg-gray-700 text-white rounded-lg text-sm border border-gray-600 focus:border-fleet-orange-500 focus:outline-none">
                    <option value="/var/log/syslog">{{ $t('mateDetail.logAnalysis.systemLog') }}</option>
                    <option value="/var/log/auth.log">{{ $t('mateDetail.logAnalysis.authLog') }}</option>
                    <option value="/var/log/kern.log">{{ $t('mateDetail.logAnalysis.kernelLog') }}</option>
                  </select>
                </div>

                <!-- Mode -->
                <div>
                  <label class="text-xs text-gray-400 block mb-1">{{ $t('mateDetail.logAnalysis.analysisMode') }}</label>
                  <select v-model="logAnalysis.mode" class="w-full px-3 py-2 bg-gray-700 text-white rounded-lg text-sm border border-gray-600 focus:border-fleet-orange-500 focus:outline-none">
                    <option value="smart">{{ $t('mateDetail.logAnalysis.modeSmart') }}</option>
                    <option value="full">{{ $t('mateDetail.logAnalysis.modeFull') }}</option>
                    <option value="errors-only">{{ $t('mateDetail.logAnalysis.modeErrorsOnly') }}</option>
                  </select>
                </div>

                <!-- Model -->
                <div>
                  <label class="text-xs text-gray-400 block mb-1">{{ $t('mateDetail.logAnalysis.aiModel') }}</label>
                  <select
                    v-model="logAnalysis.model"
                    :disabled="loadingModels"
                    class="w-full px-3 py-2 bg-gray-700 text-white rounded-lg text-sm border border-gray-600 focus:border-fleet-orange-500 focus:outline-none disabled:opacity-50">
                    <option v-if="loadingModels" value="">{{ $t('mateDetail.logAnalysis.loadingModels') }}</option>
                    <option v-for="model in availableModels" :key="model.name" :value="model.name">
                      {{ model.name }} {{ model.size ? `(${model.size})` : '' }}
                    </option>
                    <option v-if="!loadingModels && availableModels.length === 0" value="">{{ $t('mateDetail.logAnalysis.noModels') }}</option>
                  </select>
                </div>

                <!-- Enhanced Dual-Phase Progress Display -->
                <div v-if="analyzing && readingProgress < 100" class="space-y-3 p-4 bg-gray-800 rounded-lg border-2"
                     :class="progressPhase === 'reading' ? 'border-blue-500' : 'border-orange-500'">

                  <!-- Phase Indicator with Animation -->
                  <div class="flex items-center gap-3">
                    <div class="text-3xl animate-bounce">
                      {{ progressPhase === 'reading' ? '📖' : '🤖' }}
                    </div>
                    <div class="flex-1">
                      <div class="text-sm font-bold"
                           :class="progressPhase === 'reading' ? 'text-blue-400' : 'text-orange-400'">
                        {{ progressPhase === 'reading' ? $t('mateDetail.logAnalysis.phase1') : $t('mateDetail.logAnalysis.phase2') }}
                      </div>
                      <div class="text-xs text-gray-400 mt-1">
                        {{ progressPhase === 'reading'
                          ? $t('mateDetail.logAnalysis.phase1Desc')
                          : $t('mateDetail.logAnalysis.phase2Desc') }}
                      </div>
                    </div>
                    <div class="text-2xl font-bold font-mono"
                         :class="progressPhase === 'reading' ? 'text-blue-400' : 'text-orange-400'">
                      {{ readingProgress }}%
                    </div>
                  </div>

                  <!-- Enhanced Progress Bar -->
                  <div class="w-full bg-gray-900 rounded-full h-4 overflow-hidden shadow-inner">
                    <div
                      class="h-full transition-all duration-300 relative"
                      :class="progressPhase === 'reading'
                        ? 'bg-gradient-to-r from-blue-500 via-blue-400 to-blue-600'
                        : 'bg-gradient-to-r from-fleet-orange-500 via-orange-400 to-orange-600'"
                      :style="{ width: readingProgress + '%' }">
                      <!-- Animated shine effect -->
                      <div class="absolute inset-0 bg-gradient-to-r from-transparent via-white to-transparent opacity-30 animate-pulse"></div>
                    </div>
                  </div>

                  <!-- Activity Indicator -->
                  <div class="flex items-center gap-2 text-xs text-gray-500">
                    <div class="flex gap-1">
                      <div class="w-2 h-2 rounded-full bg-current animate-pulse" style="animation-delay: 0ms"></div>
                      <div class="w-2 h-2 rounded-full bg-current animate-pulse" style="animation-delay: 150ms"></div>
                      <div class="w-2 h-2 rounded-full bg-current animate-pulse" style="animation-delay: 300ms"></div>
                    </div>
                    <span>{{ $t('mateDetail.logAnalysis.systemWorking') }}</span>
                  </div>
                </div>

                <!-- Start Button -->
                <button
                  @click="startLogAnalysis"
                  :disabled="analyzing || mate.status !== 'ONLINE'"
                  class="w-full px-4 py-2 rounded-lg bg-gradient-to-r from-fleet-orange-500 to-orange-600
                         hover:from-fleet-orange-400 hover:to-orange-500
                         text-white font-semibold text-sm
                         disabled:opacity-50 disabled:cursor-not-allowed
                         transition-all duration-200 transform hover:scale-105 active:scale-95
                         flex items-center justify-center gap-2"
                >
                  <SparklesIcon class="w-4 h-4" :class="{ 'animate-spin': analyzing }" />
                  {{ analyzing ? $t('mateDetail.logAnalysis.analyzing') : $t('mateDetail.logAnalysis.startAnalysis') }}
                </button>
              </div>
            </div>

            <!-- Terminal Output -->
            <div class="bg-black/70 p-4 rounded-xl border border-gray-700/50 font-mono text-xs min-h-[300px] max-h-[500px] overflow-y-auto">
              <div class="flex items-center justify-between mb-3 pb-2 border-b border-gray-700">
                <span class="text-green-400">fleet-mate@{{ mate.mateId }}</span>
                <span class="text-gray-500 text-xs">{{ new Date().toLocaleTimeString('de-DE') }}</span>
              </div>

              <!-- Output -->
              <div class="text-gray-300 whitespace-pre-wrap" v-html="analysisOutput"></div>

              <!-- Typing Cursor -->
              <span v-if="analyzing" class="inline-block w-2 h-4 bg-green-400 animate-pulse ml-1"></span>
            </div>

            <!-- PDF Export Button -->
            <button
              v-if="!analyzing && analysisOutput !== '$ Warte auf Analyse...\\n'"
              @click="exportAsPdf"
              class="mt-4 w-full px-4 py-2 rounded-lg bg-gradient-to-r from-red-600 to-red-700
                     hover:from-red-500 hover:to-red-600
                     text-white font-semibold text-sm
                     transition-all duration-200 transform hover:scale-105 active:scale-95
                     flex items-center justify-center gap-2"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              {{ $t('mateDetail.logAnalysis.exportPdf') }}
            </button>
          </div>

          <!-- Remote Terminal Tab - nur OS-Mates -->
          <div v-if="isOsMate && activeTab === 'remote'">
            <MateTerminal :mateId="mate.mateId" />
          </div>

          <!-- AI Config Tab -->
          <div v-if="activeTab === 'ai-config'" class="space-y-6">
            <div class="bg-gradient-to-br from-gray-800/50 to-gray-900/50 p-6 rounded-xl border border-gray-700/50">
              <h3 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">
                <CogIcon class="w-5 h-5 text-fleet-orange-400" />
                {{ $t('mateDetail.aiConfig.title', { name: mate.name }) }}
              </h3>
              <p class="text-sm text-gray-400 mb-6">
                {{ $t('mateDetail.aiConfig.description') }}
              </p>

              <!-- Coder Modes (nur für Coder-Mates) -->
              <div v-if="isCoderMate" class="mb-6">
                <label class="text-sm font-semibold text-gray-300 block mb-3">{{ $t('mateDetail.aiConfig.coderMode') }}</label>
                <div class="grid grid-cols-3 gap-3">
                  <button
                    v-for="mode in coderModes"
                    :key="mode.id"
                    @click="selectCoderMode(mode)"
                    class="
                      p-4 rounded-xl border-2 transition-all duration-200
                      flex flex-col items-center gap-2 text-center
                      hover:scale-105
                    "
                    :class="mateConfig.activeMode === mode.id
                      ? 'border-fleet-orange-500 bg-fleet-orange-500/20 text-white'
                      : 'border-gray-600 bg-gray-800/50 text-gray-400 hover:border-gray-500 hover:text-gray-300'"
                  >
                    <span class="text-2xl">{{ mode.icon }}</span>
                    <span class="text-sm font-medium">{{ mode.name }}</span>
                    <span class="text-xs opacity-70">{{ mode.language }}</span>
                  </button>
                </div>
                <p class="text-xs text-gray-500 mt-2">
                  {{ $t('mateDetail.aiConfig.coderModeHint') }}
                </p>
              </div>

              <!-- Model Selection -->
              <div class="mb-6">
                <label class="text-sm font-semibold text-gray-300 block mb-2">{{ $t('mateDetail.aiConfig.llmModel') }}</label>
                <select
                  v-model="mateConfig.model"
                  :disabled="loadingMateConfig"
                  class="w-full px-4 py-3 bg-gray-700 text-white rounded-lg text-sm border border-gray-600 focus:border-fleet-orange-500 focus:outline-none disabled:opacity-50"
                >
                  <option v-if="loadingModels" value="">{{ $t('mateDetail.logAnalysis.loadingModels') }}</option>
                  <option value="">{{ $t('mateDetail.aiConfig.useDefaultModel') }}</option>
                  <option v-for="model in filteredModels" :key="model.name" :value="model.name">
                    {{ model.name }} {{ model.size ? `(${model.size})` : '' }}
                  </option>
                </select>
                <p class="text-xs text-gray-500 mt-1">
                  {{ isCoderMate ? $t('mateDetail.aiConfig.coderModelsOnly') : $t('mateDetail.aiConfig.selectModelHint') }}
                </p>
              </div>

              <!-- System Prompt -->
              <div class="mb-6">
                <div class="flex items-center justify-between mb-2">
                  <label class="text-sm font-semibold text-gray-300">{{ $t('mateDetail.aiConfig.systemPrompt') }}</label>
                  <span v-if="isCoderMate && mateConfig.activeMode" class="text-xs text-fleet-orange-400">
                    {{ $t('mateDetail.aiConfig.mode') }}: {{ getModeName(mateConfig.activeMode) }}
                  </span>
                </div>
                <textarea
                  v-model="mateConfig.systemPrompt"
                  :disabled="loadingMateConfig"
                  rows="8"
                  :placeholder="$t('mateDetail.aiConfig.systemPromptPlaceholder')"
                  class="w-full px-4 py-3 bg-gray-700 text-white rounded-lg text-sm border border-gray-600 focus:border-fleet-orange-500 focus:outline-none resize-y disabled:opacity-50 font-mono"
                ></textarea>
                <p class="text-xs text-gray-500 mt-1">
                  {{ $t('mateDetail.aiConfig.systemPromptHint') }}
                </p>
              </div>

              <!-- Save Button -->
              <div class="flex items-center gap-4">
                <button
                  @click="saveMateConfig"
                  :disabled="savingMateConfig"
                  class="px-6 py-3 rounded-lg bg-gradient-to-r from-fleet-orange-500 to-orange-600
                         hover:from-fleet-orange-400 hover:to-orange-500
                         text-white font-semibold text-sm
                         disabled:opacity-50 disabled:cursor-not-allowed
                         transition-all duration-200 transform hover:scale-105 active:scale-95
                         flex items-center gap-2"
                >
                  <ArrowPathIcon v-if="savingMateConfig" class="w-4 h-4 animate-spin" />
                  <CheckCircleIcon v-else class="w-4 h-4" />
                  {{ savingMateConfig ? $t('mateDetail.aiConfig.saving') : $t('mateDetail.aiConfig.saveSettings') }}
                </button>
                <span v-if="configSaved" class="text-green-400 text-sm flex items-center gap-1">
                  <CheckCircleIcon class="w-4 h-4" />
                  {{ $t('mateDetail.aiConfig.saved') }}
                </span>
              </div>
            </div>

            <!-- Info Box -->
            <div class="bg-blue-500/10 border border-blue-500/30 p-4 rounded-xl">
              <div class="flex items-start gap-3">
                <SparklesIcon class="w-5 h-5 text-blue-400 mt-0.5" />
                <div>
                  <h4 class="text-sm font-semibold text-blue-400">{{ $t('mateDetail.aiConfig.hint') }}</h4>
                  <p class="text-xs text-gray-400 mt-1">
                    {{ $t('mateDetail.aiConfig.hintText') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="
          flex items-center justify-between gap-3 p-6
          bg-gray-900/50
          border-t border-gray-700/50
          rounded-b-2xl
        ">
          <div class="flex items-center gap-2 text-xs text-gray-500">
            <ClockIcon class="w-4 h-4" />
            <span v-if="mate.lastHeartbeat">
              {{ $t('mateDetail.footer.lastHeartbeat') }}: {{ formatTime(mate.lastHeartbeat) }}
            </span>
          </div>
          <div class="flex gap-3">
            <button
              @click="pingMate"
              :disabled="mate.status !== 'ONLINE' || pinging"
              class="
                px-4 py-2 rounded-lg
                bg-gray-800 hover:bg-gray-700
                text-gray-300 hover:text-white
                font-medium text-sm
                border border-gray-700
                disabled:opacity-50 disabled:cursor-not-allowed
                transition-all duration-200
                flex items-center gap-2
              "
            >
              <ArrowPathIcon class="w-4 h-4" :class="{ 'animate-spin': pinging }" />
              {{ $t('mateDetail.footer.ping') }}
            </button>
            <button
              @click="$emit('close')"
              class="
                px-6 py-2 rounded-lg
                bg-fleet-orange-500 hover:bg-fleet-orange-600
                text-white font-semibold text-sm
                shadow-lg
                transition-all duration-200
                transform hover:scale-105 active:scale-95
              "
            >
              {{ $t('mateDetail.footer.close') }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import {
  ServerIcon,
  XMarkIcon,
  CheckCircleIcon,
  XCircleIcon,
  CpuChipIcon,
  CommandLineIcon,
  ArrowPathIcon,
  ClockIcon,
  SparklesIcon,
  ComputerDesktopIcon,
  FireIcon,
  CogIcon
} from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import { useToast } from '../composables/useToast'
import MateTerminal from './MateTerminal.vue'

const { t } = useI18n()

const props = defineProps({
  mate: Object,
  stats: Object
})

const emit = defineEmits(['close'])

// Prüfe ob es ein OS-Mate ist (nur OS-Mates bekommen alle Tabs)
const isOsMate = computed(() => {
  const osTypes = ['linux', 'macos', 'windows', 'os']
  return osTypes.includes(props.mate?.mateType?.toLowerCase())
})

// Für Nicht-OS-Mates direkt KI-Einstellungen Tab anzeigen
const activeTab = ref('ai-config')

// Bei OS-Mates Hardware als Standard
watch(() => props.mate, (newMate) => {
  if (newMate) {
    const osTypes = ['linux', 'macos', 'windows', 'os']
    if (osTypes.includes(newMate.mateType?.toLowerCase())) {
      activeTab.value = 'hardware'
    } else {
      activeTab.value = 'ai-config'
    }
  }
}, { immediate: true })
const pinging = ref(false)
const analyzing = ref(false)
const readingProgress = ref(0)
const progressPhase = ref('reading') // 'reading' or 'analyzing'
const analysisOutput = ref('$ Warte auf Analyse...\n')
const availableModels = ref([])
const loadingModels = ref(false)
const logAnalysis = ref({
  path: '/var/log/syslog',
  mode: 'smart',
  model: 'mistral:latest'
})

// Mate AI Configuration
const mateConfig = ref({
  model: '',
  systemPrompt: '',
  activeMode: ''
})
const loadingMateConfig = ref(false)
const savingMateConfig = ref(false)
const configSaved = ref(false)

// Coder Modi - verschiedene Programmiersprachen
const coderModes = [
  {
    id: 'go',
    name: 'Go',
    language: 'Golang',
    icon: '🐹',
    prompt: `Du bist ein erfahrener Go-Entwickler. Du folgst den Go Best Practices:
- Idiomatisches Go (effective Go)
- Klare Fehlerbehandlung mit error returns
- Goroutines und Channels für Concurrency
- Interfaces für Abstraktion
- go fmt, go vet, golint Standards
- Kurze, prägnante Variablennamen in kleinem Scope
- Dokumentation mit GoDoc-Kommentaren
- Testing mit go test

Bevorzuge die Standardbibliothek wo möglich. Vermeide über-komplizierte Abstraktionen.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'java',
    name: 'Java',
    language: 'Spring Boot',
    icon: '☕',
    prompt: `Du bist ein erfahrener Java-Entwickler. Du folgst den Java Best Practices:
- Clean Code Prinzipien
- SOLID Design Patterns
- Spring Boot / Spring Framework
- JPA/Hibernate für Persistenz
- Maven/Gradle Build-Systeme
- JUnit 5 für Testing
- Lombok für Boilerplate-Reduktion
- Jakarta EE Standards

Bevorzuge moderne Java Features (Records, Pattern Matching, var). Achte auf Exception Handling.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'python',
    name: 'Python',
    language: 'Python 3',
    icon: '🐍',
    prompt: `Du bist ein erfahrener Python-Entwickler. Du folgst den Python Best Practices:
- PEP 8 Style Guide
- Type Hints (typing module)
- Virtual Environments (venv, poetry)
- pytest für Testing
- Docstrings (Google/NumPy Style)
- List/Dict Comprehensions wo sinnvoll
- Context Managers (with statement)
- async/await für I/O

Bevorzuge die Standardbibliothek. Nutze f-strings für Formatierung.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'javascript',
    name: 'JS/TS',
    language: 'TypeScript',
    icon: '🟨',
    prompt: `Du bist ein erfahrener JavaScript/TypeScript-Entwickler. Du folgst den Best Practices:
- TypeScript für Type Safety
- ESLint/Prettier für Code-Qualität
- Modern ES6+ Syntax (async/await, destructuring, spread)
- React/Vue/Angular Patterns
- Node.js für Backend
- npm/yarn/pnpm Package Management
- Jest/Vitest für Testing
- Functional Programming Patterns

Bevorzuge TypeScript über JavaScript. Vermeide any-Types.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'bash',
    name: 'Shell',
    language: 'Bash/Zsh',
    icon: '🐚',
    prompt: `Du bist ein erfahrener Shell/Bash-Entwickler. Du folgst den Best Practices:
- ShellCheck-kompatible Scripts
- Proper quoting ("$var" statt $var)
- Error handling (set -euo pipefail)
- Funktionen für Wiederverwendbarkeit
- Kommentare für komplexe Logik
- Portable POSIX sh wo möglich
- Vermeidung von bashisms wenn nicht nötig
- Proper exit codes

Bevorzuge einfache, lesbare Lösungen. Nutze GNU coreutils effektiv.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'rust',
    name: 'Rust',
    language: 'Rust',
    icon: '🦀',
    prompt: `Du bist ein erfahrener Rust-Entwickler. Du folgst den Rust Best Practices:
- Ownership und Borrowing korrekt nutzen
- Result/Option für Error Handling
- Traits für Polymorphismus
- Cargo für Build und Dependencies
- clippy für Linting
- rustfmt für Formatierung
- Dokumentation mit /// und //!
- Unit Tests in #[cfg(test)] Modulen

Vermeide unsafe wo möglich. Nutze die Standardbibliothek effektiv.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'sql',
    name: 'SQL',
    language: 'Database',
    icon: '🗃️',
    prompt: `Du bist ein erfahrener SQL/Datenbank-Entwickler. Du folgst den Best Practices:
- Normalisierte Datenbankdesigns (3NF)
- Indexierung für Performance
- Prepared Statements gegen SQL Injection
- Transaktionen für Datenkonsistenz
- JOINs statt Subqueries wo sinnvoll
- EXPLAIN für Query-Optimierung
- Migrations für Schema-Änderungen
- Backup-Strategien

Berücksichtige MySQL/MariaDB und PostgreSQL Unterschiede.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'devops',
    name: 'DevOps',
    language: 'Infrastructure',
    icon: '🔧',
    prompt: `Du bist ein erfahrener DevOps Engineer. Du folgst den Best Practices:
- Infrastructure as Code (Terraform, Ansible)
- Container-Orchestrierung (Docker, Kubernetes)
- CI/CD Pipelines (GitHub Actions, GitLab CI)
- Monitoring und Logging (Prometheus, Grafana)
- Security Best Practices (Secrets Management)
- GitOps Workflows
- 12-Factor App Prinzipien
- Blue-Green/Canary Deployments

Automatisiere alles was möglich ist. Dokumentiere Infrastructure Changes.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'powershell',
    name: 'PowerShell',
    language: 'Windows',
    icon: '🔷',
    prompt: `Du bist ein erfahrener PowerShell-Entwickler. Du folgst den Best Practices:
- Approved Verbs (Get-, Set-, New-, Remove-, etc.)
- Cmdlet-Naming Convention (Verb-Noun)
- Pipeline-orientiertes Design
- Error Handling mit try/catch
- Comment-Based Help für Dokumentation
- Module-Struktur für Wiederverwendbarkeit
- PSScriptAnalyzer für Linting
Nutze moderne PowerShell 7+ Features wo möglich.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'batch',
    name: 'Batch',
    language: 'CMD/BAT',
    icon: '🪟',
    prompt: `Du bist ein erfahrener Windows Batch-Entwickler. Du folgst den Best Practices:
- @echo off am Anfang
- setlocal enabledelayedexpansion bei Bedarf
- Proper Error Handling mit errorlevel
- Variablen in %% für Batch, % für CMD
- Kommentare mit REM oder ::
- Saubere Ausgabe mit echo.
- Exit Codes korrekt setzen
Beachte die Unterschiede zwischen CMD und Batch.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  },
  {
    id: 'general',
    name: 'Allgemein',
    language: 'Multi-Lang',
    icon: '💻',
    prompt: `Du bist ein erfahrener Software-Entwickler mit Kenntnissen in verschiedenen Sprachen.
Du erkennst automatisch die verwendete Sprache und passt deinen Stil entsprechend an.
Du folgst allgemeinen Clean Code Prinzipien:
- DRY (Don't Repeat Yourself)
- KISS (Keep It Simple, Stupid)
- YAGNI (You Aren't Gonna Need It)
- Separation of Concerns
- Aussagekräftige Namen
- Kleine Funktionen mit einer Aufgabe
- Kommentare nur wo nötig

Frage nach der Zielsprache wenn unklar.
Antworte auf Deutsch. Nutze Markdown für Code-Blöcke.`
  }
]

// Wähle einen Coder-Modus
function selectCoderMode(mode) {
  mateConfig.value.activeMode = mode.id
  mateConfig.value.systemPrompt = mode.prompt
}

// Hole Modus-Namen
function getModeName(modeId) {
  const mode = coderModes.find(m => m.id === modeId)
  return mode ? `${mode.icon} ${mode.name}` : ''
}

const { success, error } = useToast()

// Prüfe ob es ein Coder-Mate ist
const isCoderMate = computed(() => {
  const mateType = props.mate?.mateType?.toLowerCase() || ''
  const mateName = props.mate?.name?.toLowerCase() || ''
  return mateType === 'coder' || mateName.includes('coder')
})

// Gefilterte Modelle basierend auf Mate-Typ
const filteredModels = computed(() => {
  // Für Coder-Mates: nur Coder-Modelle
  if (isCoderMate.value) {
    const coderModels = availableModels.value.filter(m =>
      m.name.toLowerCase().includes('coder') ||
      m.name.toLowerCase().includes('deepseek') ||
      m.name.toLowerCase().includes('codellama')
    )
    // Falls keine Coder-Modelle vorhanden, alle anzeigen
    return coderModels.length > 0 ? coderModels : availableModels.value
  }

  // Für andere Mates: alle Modelle
  return availableModels.value
})

// Load available models (GGUF) when modal opens
async function loadAvailableModels() {
  loadingModels.value = true
  try {
    const response = await axios.get('/api/models')
    console.log('📥 Raw models response:', response.data)

    // API gibt { models: [...], current_model: "...", provider: "..." } zurück
    const modelList = response.data.models || response.data || []

    availableModels.value = modelList.map(model => ({
      // model kann String oder Object sein
      name: typeof model === 'string' ? model : (model.name || model),
      size: typeof model === 'object' ? (model.size || 'Unknown') : 'Unknown'
    }))
    console.log('✅ Loaded models for mate detail:', availableModels.value)

    // Set first model as default if current model is not available
    if (availableModels.value.length > 0 && !availableModels.value.find(m => m.name === logAnalysis.value.model)) {
      logAnalysis.value.model = availableModels.value[0].name
    }
  } catch (err) {
    console.error('Failed to load models:', err)
    // Fallback to default GGUF model
    availableModels.value = [
      { name: 'qwen2.5-3b-instruct-q4_k_m.gguf', size: 'Unknown' }
    ]
  } finally {
    loadingModels.value = false
  }
}

// Load Mate AI Configuration
async function loadMateConfig() {
  if (!props.mate?.mateId) return

  loadingMateConfig.value = true
  try {
    const response = await axios.get(`/api/fleet-mate/mates/${props.mate.mateId}/config`)
    mateConfig.value = {
      model: response.data.model || '',
      systemPrompt: response.data.systemPrompt || '',
      activeMode: response.data.activeMode || ''
    }
    console.log('✅ Mate config loaded:', mateConfig.value)
  } catch (err) {
    console.log('Keine Mate-Config gefunden, verwende Standard-Werte')
    mateConfig.value = { model: '', systemPrompt: '', activeMode: '' }
  } finally {
    loadingMateConfig.value = false
  }
}

// Save Mate AI Configuration
async function saveMateConfig() {
  if (!props.mate?.mateId) return

  savingMateConfig.value = true
  configSaved.value = false
  try {
    await axios.put(`/api/fleet-mate/mates/${props.mate.mateId}/config`, {
      model: mateConfig.value.model,
      systemPrompt: mateConfig.value.systemPrompt,
      activeMode: mateConfig.value.activeMode
    })
    console.log('✅ Mate config saved:', mateConfig.value)
    success(t('mateDetail.toast.configSaved'))
    configSaved.value = true
    setTimeout(() => { configSaved.value = false }, 3000)
  } catch (err) {
    console.error('Failed to save mate config:', err)
    error(t('mateDetail.toast.configSaveError'))
  } finally {
    savingMateConfig.value = false
  }
}

// Load models and config on mount
onMounted(() => {
  loadAvailableModels()
  loadMateConfig()
})

async function pingMate() {
  pinging.value = true
  try {
    await axios.post(`/api/fleet-mate/mates/${props.mate.mateId}/ping`)
    success(t('mateDetail.toast.pingSuccess'))
  } catch (err) {
    console.error('Failed to ping mate:', err)
    error(t('mateDetail.toast.pingFailed'))
  } finally {
    setTimeout(() => {
      pinging.value = false
    }, 500)
  }
}

let currentSessionId = ''

async function exportAsPdf() {
  try {
    // Clean up the analysis output (remove ANSI codes, terminal prompts, etc.)
    const cleanContent = analysisOutput.value
      .replace(/\$[^\n]*\n/g, '') // Remove terminal prompts
      .replace(/✓[^\n]*\n/g, '')  // Remove status messages
      .replace(/🤖[^\n]*\n/g, '') // Remove AI status messages
      .trim()

    if (!cleanContent) {
      error(t('mateDetail.toast.noDataToExport'))
      return
    }

    const response = await axios.post('/api/fleet-mate/export-pdf', {
      content: cleanContent,
      mateId: props.mate.mateId,
      logPath: logAnalysis.value.path,
      sessionId: currentSessionId
    }, {
      responseType: 'blob'
    })

    // Create download link
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url

    // Extract filename from Content-Disposition header or create default
    const contentDisposition = response.headers['content-disposition']
    let filename = 'log-analysis.pdf'
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="?(.+)"?/i)
      if (filenameMatch) {
        filename = filenameMatch[1]
      }
    }

    link.setAttribute('download', filename)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)

    success(t('mateDetail.toast.pdfSuccess'))
  } catch (err) {
    console.error('Failed to export PDF:', err)
    error(t('mateDetail.toast.pdfFailed'))
  }
}

async function startLogAnalysis() {
  analyzing.value = true
  readingProgress.value = 0
  progressPhase.value = 'reading'
  analysisOutput.value = `$ ${t('mateDetail.logAnalysis.startingAnalysis')}...\n\n`

  try {
    // Send analysis request
    const response = await axios.post(
      `/api/fleet-mate/mates/${props.mate.mateId}/analyze-log`,
      {
        logPath: logAnalysis.value.path,
        mode: logAnalysis.value.mode,
        model: logAnalysis.value.model,
        prompt: 'Analysiere dieses System-Log nach Fehlern, Warnungen und Auffälligkeiten.'
      }
    )

    const sessionId = response.data.sessionId
    currentSessionId = sessionId // Store for PDF export
    analysisOutput.value += `✓ ${t('mateDetail.logAnalysis.sessionCreated')}: ${sessionId}\n`
    analysisOutput.value += `✓ ${t('mateDetail.logAnalysis.mateReadingLog')}...\n\n`

    // Connect to SSE stream
    const eventSource = new EventSource(`/api/fleet-mate/stream/${sessionId}`)

    eventSource.addEventListener('progress', (event) => {
      const data = JSON.parse(event.data)
      readingProgress.value = Math.round(data.progress)
      progressPhase.value = data.phase || (data.progress < 50 ? 'reading' : 'analyzing')
      console.log('[SSE] progress:', readingProgress.value + '%', 'phase:', progressPhase.value)
    })

    eventSource.addEventListener('start', (event) => {
      console.log('[SSE] start event received:', event.data)
      const data = JSON.parse(event.data)
      progressPhase.value = 'analyzing'
      analysisOutput.value += `🤖 ${t('mateDetail.logAnalysis.aiStarted', { model: data.model })}...\n\n`
    })

    eventSource.addEventListener('chunk', (event) => {
      const data = JSON.parse(event.data)
      analysisOutput.value += data.chunk
    })

    let analysisCompleted = false

    eventSource.addEventListener('done', (event) => {
      console.log('[SSE] done event received:', event.data)
      analysisCompleted = true
      analysisOutput.value += `\n\n✓ ${t('mateDetail.logAnalysis.analysisComplete')}!\n`
      eventSource.close()
      analyzing.value = false
      success(t('mateDetail.toast.analysisComplete'))
    })

    eventSource.addEventListener('error', (event) => {
      console.log('[SSE] error event received, analysisCompleted:', analysisCompleted)

      // Ignore error if analysis was completed successfully (SSE auto-closes with error event)
      if (analysisCompleted) {
        console.log('[SSE] Ignoring error event - analysis was already completed')
        return
      }

      // Only show error for real failures (before completion)
      console.error('[SSE] Real error - stream interrupted before completion:', event)
      analysisOutput.value += `\n\n✗ ${t('mateDetail.logAnalysis.connectionError')}\n`
      eventSource.close()
      analyzing.value = false
      error(t('mateDetail.toast.analysisFailed'))
    })

  } catch (err) {
    console.error('Failed to start analysis:', err)
    analysisOutput.value += `\n✗ ${t('mateDetail.logAnalysis.error')}: ${err.message}\n`
    analyzing.value = false
    error(t('mateDetail.toast.analysisStartFailed'))
  }
}

function formatTime(timestamp) {
  if (!timestamp) return t('mateDetail.time.never')
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now - date
  const diffSecs = Math.floor(diffMs / 1000)
  const diffMins = Math.floor(diffSecs / 60)

  if (diffSecs < 60) return t('mateDetail.time.secondsAgo', { n: diffSecs })
  if (diffMins < 60) return t('mateDetail.time.minutesAgo', { n: diffMins })

  return date.toLocaleTimeString()
}

function formatUptime(seconds) {
  if (!seconds) return 'N/A'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  return `${hours}h`
}

function shortenCPUModel(model) {
  if (!model) return 'N/A'
  return model.replace(/\(R\)|\(TM\)|Processor|CPU/gi, '').trim().substring(0, 30)
}

function getCoreTemp(coreIndex) {
  if (!props.stats?.temperature?.per_core) return null
  return props.stats.temperature.per_core[coreIndex]
}

function getCPUColor(percent) {
  if (!percent) return 'text-gray-400'
  if (percent < 50) return 'text-green-400'
  if (percent < 75) return 'text-yellow-400'
  return 'text-red-400'
}

function getCPUBarColor(percent) {
  if (!percent) return 'bg-gray-600'
  if (percent < 50) return 'bg-gradient-to-r from-green-500 to-emerald-500'
  if (percent < 75) return 'bg-gradient-to-r from-yellow-500 to-amber-500'
  return 'bg-gradient-to-r from-red-500 to-rose-500'
}

function getRAMColor(percent) {
  if (!percent) return 'text-gray-400'
  if (percent < 70) return 'text-blue-400'
  if (percent < 85) return 'text-yellow-400'
  return 'text-red-400'
}

function getRAMBarColor(percent) {
  if (!percent) return 'bg-gray-600'
  if (percent < 70) return 'bg-gradient-to-r from-blue-500 to-cyan-500'
  if (percent < 85) return 'bg-gradient-to-r from-yellow-500 to-amber-500'
  return 'bg-gradient-to-r from-red-500 to-rose-500'
}

function getTempBadgeColor(temp) {
  if (!temp) return 'bg-gray-600 text-gray-300'
  if (temp < 60) return 'bg-green-500/20 text-green-400 border border-green-500/30'
  if (temp < 80) return 'bg-yellow-500/20 text-yellow-400 border border-yellow-500/30'
  return 'bg-red-500/20 text-red-400 border border-red-500/30'
}

function getTempIconColor(temp) {
  if (!temp) return 'text-gray-400'
  if (temp < 60) return 'text-green-400'
  if (temp < 80) return 'text-yellow-400'
  return 'text-red-400'
}

function getGPUColor(percent) {
  if (!percent) return 'text-gray-400'
  if (percent < 50) return 'text-green-400'
  if (percent < 75) return 'text-yellow-400'
  return 'text-red-400'
}

function getGPUBarColor(percent) {
  if (!percent) return 'bg-gray-600'
  if (percent < 50) return 'bg-gradient-to-r from-green-500 to-emerald-500'
  if (percent < 75) return 'bg-gradient-to-r from-yellow-500 to-amber-500'
  return 'bg-gradient-to-r from-red-500 to-rose-500'
}

function getVRAMColor(percent) {
  if (!percent) return 'text-gray-400'
  if (percent < 70) return 'text-purple-400'
  if (percent < 85) return 'text-yellow-400'
  return 'text-red-400'
}

function getVRAMBarColor(percent) {
  if (!percent) return 'bg-gray-600'
  if (percent < 70) return 'bg-gradient-to-r from-purple-500 to-pink-500'
  if (percent < 85) return 'bg-gradient-to-r from-yellow-500 to-amber-500'
  return 'bg-gradient-to-r from-red-500 to-rose-500'
}
</script>

<style scoped>
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
</style>
