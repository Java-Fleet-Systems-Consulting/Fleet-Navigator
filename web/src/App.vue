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
import { ref, onMounted } from 'vue'
import GlobalConfirmDialog from './components/GlobalConfirmDialog.vue'
import UpdateNotification from './components/UpdateNotification.vue'
import SetupWizard from './components/SetupWizard.vue'

// Setup Wizard State
const showSetupWizard = ref(false)

// Check if First Run on Mount
onMounted(async () => {
  try {
    const resp = await fetch('/api/setup/status')
    const data = await resp.json()
    showSetupWizard.value = data.isFirstRun
  } catch (e) {
    // API not available, assume not first run
    console.log('Setup Status nicht verfügbar:', e)
    showSetupWizard.value = false
  }
})

// Setup Complete Handler
function onSetupComplete() {
  showSetupWizard.value = false
  // Optional: Page reload for clean state
  // window.location.reload()
}
</script>
