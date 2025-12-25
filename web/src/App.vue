<template>
  <!-- Setup Wizard (First Run) -->
  <SetupWizard v-if="showSetupWizard" @complete="onSetupComplete" />

  <!-- Main App -->
  <template v-else>
    <router-view />
    <!-- Global Confirm Dialog (available everywhere) -->
    <GlobalConfirmDialog />
    <!-- Auto-Update Notification Banner -->
    <UpdateNotification />
  </template>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import GlobalConfirmDialog from './components/GlobalConfirmDialog.vue'
import UpdateNotification from './components/UpdateNotification.vue'
import SetupWizard from './components/SetupWizard.vue'

// Setup Wizard State
const showSetupWizard = ref(false)
const isFirstRun = ref(false)
const route = useRoute()

// Help-Seiten brauchen NIEMALS den Setup-Wizard
const isHelpPage = computed(() => route.path.startsWith('/help'))

// Setup-Wizard nur zeigen wenn: First Run UND NICHT auf Help-Seite
const shouldShowSetupWizard = computed(() => isFirstRun.value && !isHelpPage.value)

// Watch für reaktive Updates
watch(shouldShowSetupWizard, (newVal) => {
  showSetupWizard.value = newVal
}, { immediate: true })

// Check if First Run on Mount
onMounted(async () => {
  try {
    const resp = await fetch('/api/setup/status')
    const data = await resp.json()
    isFirstRun.value = data.isFirstRun
  } catch (e) {
    // API not available, assume not first run
    console.log('Setup Status nicht verfügbar:', e)
    isFirstRun.value = false
  }
})

// Setup Complete Handler
function onSetupComplete() {
  showSetupWizard.value = false
  // Optional: Page reload for clean state
  // window.location.reload()
}
</script>
