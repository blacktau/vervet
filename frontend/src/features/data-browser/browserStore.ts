import { defineStore } from 'pinia'
import { isEmpty } from 'lodash'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { useTabStore } from '@/stores/tabs.ts'
import { useNotifier } from '@/utils/dialog.ts'

type DataBrowserStoreState = {
  connections: Record<string, ServerConnection>
}

type ServerConnection = {
  serverId: string,
  name: string,
  databases: Database[]
}

type Database = {
  name: string,
  collections: Collection[]
}

type Collection = {
  name: string,
  indexes: Index[]
}

type Index = {
  name: string,
}

export const useDataBrowserStore = defineStore(
  'browser',
  {
    state: () => ({
      connections: {} as Record<string, ServerConnection>
    }),
    getters: {
      hasOpenConnections: (state: DataBrowserStoreState) => {
        return !isEmpty(state.connections)
      },
      isConnected: (state: DataBrowserStoreState) => {
        return (serverID: string) => {
          return state.connections.hasOwnProperty(serverID)
        }
      }
    },
    actions: {
      async disconnectAll() {
        for (const serverID in this.connections) {
          await connectionsProxy.Disconnect(serverID)
          delete this.connections[serverID]
        }
        const tabStore = useTabStore()
        tabStore.removeAllTabs()
      },
      async disconnect(serverId: string) {
        const connection = this.connections[serverId]
        if (!connection) {
          return false
        }
        await connectionsProxy.Disconnect(serverId)
        delete this.connections[serverId]
        return true
      },
      async getDatabaseList(serverId: string) {
        const connection = this.connections[serverId]
        if (!!connection) {
          const databases = await connectionsProxy.GetDatabases(serverId)
          if (databases.isSuccess) {
            connection.databases = databases.data.map((db) => ({ name: db, collections: [] }))
            return connection.databases
          } else {
            const notifier = useNotifier()
            notifier.error(`error retrieving databases: ${databases.error}`)
            return []
          }
        }
        return []
      },
      async connect(serverId: string, reload: boolean = false) {
        if (this.isConnected(serverId)) {
          if (!reload) {
            await connectionsProxy.Disconnect(serverId)
          }
          return
        }

        const result = await connectionsProxy.Connect(serverId)
        if (!result.isSuccess) {
          const notifier = useNotifier()
          notifier.error(`error connecting to server: ${result.error}`)
        }

        this.connections[serverId] = {
          serverId: serverId,
          name: '',
          databases: []
        }
      }

    }
  }
)
