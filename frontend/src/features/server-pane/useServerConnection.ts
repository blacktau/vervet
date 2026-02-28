import { ref } from 'vue'
import { isEmpty } from 'lodash'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useMessager } from '@/utils/dialog.ts'

export function useServerConnection() {
  const browserStore = useDataBrowserStore()
  const tabStore = useTabStore()

  const connectingServer = ref('')

  const connectToServer = async (serverId: string) => {
    try {
      connectingServer.value = serverId
      const connectionResult = await browserStore.connect(serverId)
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
      connectingServer.value = ''
    }
  }

  const onCancelConnecting = async () => {
    if (connectingServer.value === '') return
    await browserStore.disconnect(connectingServer.value)
    connectingServer.value = ''
  }

  return { connectingServer, connectToServer, onCancelConnecting }
}
