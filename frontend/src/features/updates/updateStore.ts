import { defineStore } from 'pinia'
import { ref } from 'vue'
import { CheckNow, DismissUpdate, OpenReleasePage } from 'wailsjs/go/api/UpdatesProxy'
import { EventsOn, EventsOff } from 'wailsjs/runtime/runtime'
import { useSettingsStore } from '@/features/settings/settingsStore'

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

  function subscribe() {
    EventsOn('update-available', (info: UpdateInfo) => {
      applyEvent(info)
    })
  }

  function unsubscribe() {
    EventsOff('update-available')
  }

  async function checkNow() {
    checking.value = true
    lastError.value = null
    try {
      const result = await CheckNow()
      if (!result.isSuccess) {
        lastError.value = result.errorDetail || result.errorCode || 'Unknown error'
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
