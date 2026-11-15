import { watch } from 'vue'
import { onEvent } from '@/utils/wailsEvents'
import * as closeGuardProxy from 'bindings/vervet/internal/api/closeguardproxy'
import { useTabStore } from '@/features/tabs/tabs'

/** Stops closing the application from silently dropping unsaved queries. */
export function useAppCloseGuard() {
  const tabStore = useTabStore()

  // Go only asks when told there's something to ask about, so a frontend that has
  // stopped responding can't keep the application from closing.
  watch(
    () => tabStore.hasUnsavedQueries,
    (unsaved) => {
      void closeGuardProxy.SetUnsavedChanges(unsaved)
    },
    { immediate: true },
  )

  onEvent('app-close-requested', async () => {
    try {
      await closeGuardProxy.AcknowledgeCloseRequest()
      if (await tabStore.resolveAllUnsavedQueries()) {
        await closeGuardProxy.ConfirmClose()
        return
      }
    } catch {
      // Falls through: staying open and able to ask again beats blocking every close.
    }
    await closeGuardProxy.CancelClose()
  })
}
