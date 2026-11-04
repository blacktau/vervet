import { watch } from 'vue'
import { EventsOn } from 'wailsjs/runtime/runtime'
import * as closeGuardProxy from 'wailsjs/go/api/CloseGuardProxy'
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

  EventsOn('app-close-requested', async () => {
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
