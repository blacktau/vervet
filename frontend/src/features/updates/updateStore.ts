import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  CheckIfDue,
  CheckNow,
  DismissUpdate,
  InstallUpdate,
  OpenReleasePage,
  RestartToUpdate,
} from 'bindings/vervet/internal/api/updatesproxy'
import { reportRefusedLink } from '@/utils/openInBrowser'
import { onEvent } from '@/utils/wailsEvents'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { resultDetailText } from '@/utils/resultError'

export interface UpdateInfo {
  available: boolean
  version: string
  url: string
  releaseNotes: string
  canInstall: boolean
}

/** Where an in-app update has got to. */
export type InstallPhase = 'idle' | 'downloading' | 'verifying' | 'ready' | 'error'

/** Payload of the Wails updater's download-progress event. */
interface DownloadProgress {
  written: number
  total: number
}

// Emitted by the Wails updater (github.com/wailsapp/wails/v3/pkg/updater).
const EventDownloadProgress = 'wails:updater:download-progress'
const EventVerifying = 'wails:updater:verifying'
// Emitted by Vervet when restarting fails after the unsaved-queries prompt.
const EventRestartFailed = 'update-restart-failed'

export const useUpdateStore = defineStore('updates', () => {
  const available = ref(false)
  const version = ref('')
  const url = ref('')
  const releaseNotes = ref('')
  const checking = ref(false)
  const lastError = ref<string | null>(null)
  const canInstall = ref(false)
  const phase = ref<InstallPhase>('idle')
  /** Download progress from 0 to 100, or null when the size isn't known. */
  const progress = ref<number | null>(null)
  const installError = ref<string | null>(null)

  function applyEvent(info: UpdateInfo) {
    available.value = info.available
    version.value = info.version
    url.value = info.url
    releaseNotes.value = info.releaseNotes
    canInstall.value = info.canInstall
  }

  function applyProgress(p: DownloadProgress) {
    progress.value = p.total > 0 ? Math.min(100, Math.round((p.written / p.total) * 100)) : null
  }

  let stopListening: (() => void)[] = []

  function subscribe() {
    unsubscribe()
    stopListening = [
      onEvent<UpdateInfo>('update-available', applyEvent),
      onEvent<DownloadProgress>(EventDownloadProgress, applyProgress),
      onEvent(EventVerifying, () => {
        phase.value = 'verifying'
      }),
      onEvent<string>(EventRestartFailed, (message) => {
        phase.value = 'error'
        installError.value = message
      }),
    ]
    void CheckIfDue()
  }

  function unsubscribe() {
    stopListening.forEach((stop) => stop())
    stopListening = []
  }

  /** Downloads and verifies the update; `phase` becomes 'ready' when it can be installed. */
  async function install() {
    if (!canInstall.value || phase.value === 'downloading' || phase.value === 'verifying') {
      return
    }
    phase.value = 'downloading'
    progress.value = null
    installError.value = null
    const result = await InstallUpdate()
    if (!result.isSuccess) {
      phase.value = 'error'
      installError.value = resultDetailText(result)
      return
    }
    phase.value = 'ready'
  }

  /** Asks about unsaved queries, then restarts into the downloaded update. */
  async function restart() {
    if (phase.value !== 'ready') {
      return
    }
    const result = await RestartToUpdate()
    if (!result.isSuccess) {
      phase.value = 'error'
      installError.value = resultDetailText(result)
    }
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
    reportRefusedLink(url.value, await OpenReleasePage(url.value))
  }

  return {
    available,
    version,
    url,
    releaseNotes,
    checking,
    lastError,
    canInstall,
    phase,
    progress,
    installError,
    applyEvent,
    install,
    restart,
    subscribe,
    unsubscribe,
    checkNow,
    dismiss,
    openReleasePage,
  }
})
