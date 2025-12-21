import { ref, readonly } from 'vue'
import { useI18n } from 'vue-i18n'

// Shared state for the confirm dialog
const isOpen = ref(false)
const dialogConfig = ref({
  title: '',
  message: '',
  type: 'warning',
  confirmText: '',
  cancelText: ''
})

let resolvePromise = null

/**
 * Composable for showing confirmation dialogs
 *
 * Usage:
 * const { confirm } = useConfirmDialog()
 *
 * // Simple usage
 * const confirmed = await confirm('Wirklich löschen?')
 *
 * // With options
 * const confirmed = await confirm({
 *   title: 'Chat löschen',
 *   message: 'Diese Aktion kann nicht rückgängig gemacht werden.',
 *   type: 'danger',
 *   confirmText: 'Löschen'
 * })
 */
export function useConfirmDialog() {
  const { t } = useI18n()

  /**
   * Show a confirmation dialog
   * @param {string|Object} options - Message string or config object
   * @returns {Promise<boolean>} - Resolves to true if confirmed, false if cancelled
   */
  function confirm(options) {
    return new Promise((resolve) => {
      resolvePromise = resolve

      if (typeof options === 'string') {
        // Simple string message
        dialogConfig.value = {
          title: t('confirmDialog.defaultTitle'),
          message: options,
          type: 'warning',
          confirmText: t('confirmDialog.confirm'),
          cancelText: t('confirmDialog.cancel')
        }
      } else {
        // Full config object
        dialogConfig.value = {
          title: options.title || t('confirmDialog.defaultTitle'),
          message: options.message || t('confirmDialog.defaultMessage'),
          type: options.type || 'warning',
          confirmText: options.confirmText || t('confirmDialog.confirm'),
          cancelText: options.cancelText || t('confirmDialog.cancel')
        }
      }

      isOpen.value = true
    })
  }

  /**
   * Shortcut for delete confirmation
   */
  function confirmDelete(itemName, additionalMessage = '') {
    return confirm({
      title: t('confirmDialog.deleteTitle', { name: itemName }),
      message: additionalMessage || t('confirmDialog.deleteMessage'),
      type: 'danger',
      confirmText: t('common.delete'),
      cancelText: t('confirmDialog.cancel')
    })
  }

  /**
   * Handle confirm action (called by ConfirmDialog component)
   */
  function handleConfirm() {
    isOpen.value = false
    if (resolvePromise) {
      resolvePromise(true)
      resolvePromise = null
    }
  }

  /**
   * Handle cancel action (called by ConfirmDialog component)
   */
  function handleCancel() {
    isOpen.value = false
    if (resolvePromise) {
      resolvePromise(false)
      resolvePromise = null
    }
  }

  return {
    // State (readonly for external use)
    isOpen: readonly(isOpen),
    dialogConfig: readonly(dialogConfig),

    // Methods
    confirm,
    confirmDelete,
    handleConfirm,
    handleCancel
  }
}

// Export singleton state for the dialog component
export function useConfirmDialogState() {
  return {
    isOpen,
    dialogConfig,
    handleConfirm() {
      isOpen.value = false
      if (resolvePromise) {
        resolvePromise(true)
        resolvePromise = null
      }
    },
    handleCancel() {
      isOpen.value = false
      if (resolvePromise) {
        resolvePromise(false)
        resolvePromise = null
      }
    }
  }
}
