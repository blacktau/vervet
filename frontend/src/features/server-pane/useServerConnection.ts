import { ref } from 'vue'
import { isEmpty } from 'lodash'
import * as oidcProxy from 'wailsjs/go/api/OIDCProxy'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useMessager } from '@/utils/dialog.ts'

export function useServerConnection() {
  const browserStore = useDataBrowserStore()
  const tabStore = useTabStore()

  const connectingServer = ref('')
  // Attempt generation. A cancelled OIDC connect can still be parked in the
  // backend when the user retries, so its late result must not touch the state
  // of the newer attempt (which would swallow the tab it should have opened).
  let attempt = 0

  const connectToServer = async (serverId: string) => {
    const myAttempt = ++attempt
    try {
      connectingServer.value = serverId
      const connectionResult = await browserStore.connect(serverId)
      if (myAttempt !== attempt) {
        return
      }
      if (!connectionResult.success) {
        return
      }
      if (!isEmpty(connectingServer.value)) {
        tabStore.upsertTab({
          serverId,
          title: connectionResult.name || '',
          forceSwitch: true,
          blank: false,
        })
      }
    } catch (e) {
      const messager = useMessager()
      const err = e as Error
      messager.error(err.message)
    } finally {
      if (myAttempt === attempt) {
        connectingServer.value = ''
      }
    }
  }

  const onCancelConnecting = async () => {
    if (connectingServer.value === '') return
    const serverId = connectingServer.value
    // Invalidate the in-flight attempt so its late result is ignored.
    attempt++
    // Unblocks a pending OIDC browser login; no-op for other auth methods.
    await oidcProxy.CancelLogin(serverId)
    await browserStore.disconnect(serverId)
    connectingServer.value = ''
  }

  return { connectingServer, connectToServer, onCancelConnecting }
}
