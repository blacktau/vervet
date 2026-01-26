import { defineStore } from 'pinia'
import { isEmpty } from 'lodash'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useNotifier } from '@/utils/dialog.ts'
import { type models } from 'wailsjs/go/models.ts'

type DataBrowserStoreState = {
  connections: models.Connection[]
}

type ServerConnection = models.Connection & {
  databases?: Database[]
}

type Database = {
  name: string
  collections: Collection[]
}

type Collection = {
  name: string
  indexes: Index[]
}

type Index = {
  name: string
}

export const useDataBrowserStore = defineStore('browser', {
  state: () => ({
    connections: [] as ServerConnection[],
  }),
  getters: {
    hasOpenConnections: (state: DataBrowserStoreState) => {
      return !isEmpty(state.connections)
    },
    isConnected: (state: DataBrowserStoreState) => {
      return (serverID?: string) => {
        if (serverID == null) {
          return false
        }

        const server = state.connections.find((x) => x.serverID === serverID)
        return server != null
      }
    },
  },
  actions: {
    async disconnectAll() {
      await connectionsProxy.DisconnectAll()
      this.connections = []
      const tabStore = useTabStore()
      tabStore.removeAllTabs()
    },
    async disconnect(serverId: string) {
      const server = this.connections.find((x) => x.serverID === serverId)
      if (server != null) {
        await connectionsProxy.Disconnect(serverId)
        this.connections = this.connections.filter((x) => x.serverID !== serverId)
        const tabStore = useTabStore()
        tabStore.removeTabById(serverId)
      }
      return true
    },
    async refreshConnectedServers(force: boolean = false) {
      if (!force && !isEmpty(this.connections)) {
        return
      }

      const connections = await connectionsProxy.GetConnections()
      if (!connections.isSuccess) {
        return
      }

      this.connections = connections.data
    },
    async getDatabaseList(serverId: string, force: boolean = false): Promise<Database[]> {
      const connection = this.connections.find((x) => x.serverID === serverId)
      if (connection != null) {
        if (!force && connection.databases != null) {
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
      }
      return []
    },
    async connect(serverId: string, reload: boolean = false) {
      if (this.isConnected(serverId)) {
        if (!reload) {
          await connectionsProxy.Disconnect(serverId)
        }
        return {
          success: true,
          serverId: serverId,
          name: this.connections.find((x) => x.serverID === serverId)?.name || '',
        }
      }

      const result = await connectionsProxy.Connect(serverId)
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(`error connecting to server: ${result.error}`)
        return {
          success: false,
        }
      }

      await this.refreshConnectedServers(true)
      return {
        success: true,
        serverId: serverId,
        name: result.data.name,
      }
    },
  },
})
