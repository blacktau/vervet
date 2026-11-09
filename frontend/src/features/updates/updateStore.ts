import { defineStore } from 'pinia'
import { ref } from 'vue'
import { CheckIfDue, CheckNow, DismissUpdate, OpenReleasePage } from 'bindings/vervet/internal/api/updatesproxy'
import { onEvent } from '@/utils/wailsEvents'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { resultDetailText } from '@/utils/resultError'

export interface UpdateInfo {
  available: boolean
  version: string
  url: string
  releaseNotes: string
}

export const useUpdateStore = defineStore('updates', () => {
  const available = ref(false)
  const version = ref('')
  const url = ref('')
  const releaseNotes = ref('')
  const checking = ref(false)
  const lastError = ref<string | null>(null)

  function applyEvent(info: UpdateInfo) {
    available.value = info.available
    version.value = info.version
    url.value = info.url
    releaseNotes.value = info.releaseNotes
  }

  let stopListening: (() => void) | null = null

  function subscribe() {
    stopListening?.()
    stopListening = onEvent<UpdateInfo>('update-available', applyEvent)
    void CheckIfDue()
  }

  function unsubscribe() {
    stopListening?.()
    stopListening = null
  }

  async function checkNow() {
    checking.value = true
    lastError.value = null
    try {
      const result = await CheckNow()
      if (!result.isSuccess) {
        lastError.value = resultDetailText(result)
        return
      }
      applyEvent(result.data as UpdateInfo)
    } finally {
      checking.value = false
      await useSettingsStore().loadSettings()
    }
  }

  async function dismiss() {
    if (!version.value) {
      return
    }
    const result = await DismissUpdate(version.value)
    if (result.isSuccess) {
      available.value = false
    }
  }

  async function openReleasePage() {
    if (!url.value) {
      return
    }
    await OpenReleasePage(url.value)
  }

  return {
    available,
    version,
    url,
    releaseNotes,
    checking,
    lastError,
    applyEvent,
    subscribe,
    unsubscribe,
    checkNow,
    dismiss,
    openReleasePage,
  }
})
