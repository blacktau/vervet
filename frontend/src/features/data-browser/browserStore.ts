import { defineStore } from 'pinia'
import { isEmpty } from 'lodash'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useNotifier } from '@/utils/dialog.ts'
import { type models } from 'wailsjs/go/models.ts'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'

type DataBrowserStoreState = {
  connections: models.Connection[]
  serverTreeStates: Record<string, ServerTreeState>
}

type ServerConnection = models.Connection & {
  databases?: Database[]
}

type Database = {
  name: string
  collections: Collection[]
  views: string[]
}

type Collection = {
  name: string
  indexes: Index[]
}

type Index = {
  name: string
}

interface TreeNode extends DataTreeNode {
  children: TreeNode[]
}

interface ServerTreeState {
  expandedKeys: string[]
  loadedKeys: string[]
  treeData: TreeNode[]
}

const FOLDER_COLLECTIONS = 'Collections'
const FOLDER_VIEWS = 'Views'

export const useDataBrowserStore = defineStore('browser', {
  state: () => ({
    connections: [] as ServerConnection[],
    serverTreeStates: {} as Record<string, ServerTreeState>,
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
    currentTreeData(): TreeNode[] {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) return []
      return this.serverTreeStates[serverId]?.treeData ?? []
    },
    currentExpandedKeys(): string[] {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) return []
      return this.serverTreeStates[serverId]?.expandedKeys ?? []
    },
  },
  actions: {
    getOrCreateTreeState(serverId: string): ServerTreeState {
      if (!this.serverTreeStates[serverId]) {
        this.serverTreeStates[serverId] = {
          expandedKeys: [],
          loadedKeys: [],
          treeData: [],
        }
      }
      return this.serverTreeStates[serverId]
    },

    buildTreeForServer(serverId: string): TreeNode[] {
      const connection = this.connections.find((c) => c.serverID === serverId)
      if (!connection) return []

      return [
        {
          label: connection.name,
          key: connection.serverID,
          isLeaf: false,
          type: DataNodeType.Server,
          children: [],
        },
      ]
    },

    findNode(nodes: TreeNode[], key: string | number): TreeNode | null {
      for (const node of nodes) {
        if (node.key === key) {
          return node
        }
        if (node.children.length > 0) {
          const found = this.findNode(node.children, key)
          if (found) return found
        }
      }
      return null
    },

    async expandNode(serverId: string, key: string) {
      const state = this.serverTreeStates[serverId]
      if (!state) return

      const node = this.findNode(state.treeData, key)
      if (!node) return

      if (node.type === DataNodeType.Server) {
        await this.expandServer(serverId, node)
        return
      }

      if (node.type === DataNodeType.Database) {
        await this.expandDatabase(serverId, node)
        return
      }

      if (node.type === DataNodeType.Folder) {
        this.expandFolder(serverId, node)
      }
    },

    async expandServer(serverId: string, node: TreeNode) {
      const state = this.serverTreeStates[serverId]
      if (!state) return

      const key = node.key as string
      if (state.loadedKeys.includes(key)) return

      await this.getDatabaseList(key, true)
      const connection = this.connections.find((c) => c.serverID === key)
      if (!connection?.databases) return

      node.children = connection.databases.map((db) => ({
        label: db.name,
        key: `${key}:${db.name}`,
        isLeaf: false,
        type: DataNodeType.Database,
        children: [],
      }))

      state.loadedKeys = [...state.loadedKeys, key]
      state.expandedKeys = [...state.expandedKeys, key]
      state.treeData = [...state.treeData]
    },

    async expandDatabase(serverId: string, node: TreeNode) {
      const state = this.serverTreeStates[serverId]
      if (!state) return

      const dbKey = node.key as string
      if (!dbKey || state.loadedKeys.includes(dbKey)) return

      const [sid, dbName] = dbKey.split(':')
      if (!sid || !dbName) return

      await this.getCollectionList(sid, dbName, true)
      await this.getViewList(sid, dbName, true)

      node.children = [
        {
          label: FOLDER_COLLECTIONS,
          key: `${dbKey}:${FOLDER_COLLECTIONS}`,
          isLeaf: false,
          type: DataNodeType.Folder,
          children: [],
        },
        {
          label: FOLDER_VIEWS,
          key: `${dbKey}:${FOLDER_VIEWS}`,
          isLeaf: false,
          type: DataNodeType.Folder,
          children: [],
        },
      ]

      state.loadedKeys = [...state.loadedKeys, dbKey]
      state.expandedKeys = [...state.expandedKeys, dbKey]
      state.treeData = [...state.treeData]
    },

    async expandFolder(serverId: string, node: TreeNode) {
      const state = this.serverTreeStates[serverId]
      if (!state) return

      const folderKey = node.key as string
      if (!folderKey || state.loadedKeys.includes(folderKey)) return

      const parts = folderKey.split(':')
      if (parts.length < 3) return

      const sid = parts[0]
      const dbName = parts[1]
      const folderName = parts[2]

      const isCollectionsFolder = folderName === FOLDER_COLLECTIONS
      const isViewsFolder = folderName === FOLDER_VIEWS

      if (!isCollectionsFolder && !isViewsFolder) return

      await this.getDatabaseList(sid!)
      if (isCollectionsFolder) {
        await this.getCollectionList(sid!, dbName!, true)
      } else {
        await this.getViewList(sid!, dbName!, true)
      }

      const database = this.findDatabase(sid!, dbName!)
      if (!database) return

      if (isCollectionsFolder && database.collections) {
        node.children = database.collections.map((col) => ({
          label: col.name,
          key: `${folderKey}:${col.name}`,
          isLeaf: true,
          type: DataNodeType.Collection,
          children: [],
        }))
      } else if (isViewsFolder && database.views) {
        node.children = database.views.map((view) => ({
          label: view,
          key: `${folderKey}:${view}`,
          isLeaf: true,
          type: DataNodeType.View,
          children: [],
        }))
      }

      state.loadedKeys = [...state.loadedKeys, folderKey]
      state.expandedKeys = [...state.expandedKeys, folderKey]
      state.treeData = [...state.treeData]
    },

    handleExpand(keys: string[]) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) return

      const state = this.serverTreeStates[serverId]
      if (!state) return

      const oldExpanded = state.expandedKeys
      state.expandedKeys = keys

      for (const key of keys) {
        if (oldExpanded.includes(key)) continue
        this.expandNode(serverId, key)
      }
    },

    updateTreeForServer(serverId: string | undefined) {
      if (!serverId) return

      const state = this.getOrCreateTreeState(serverId)

      if (state.treeData.length === 0) {
        state.treeData = this.buildTreeForServer(serverId)

        const firstKey = state.treeData[0]?.key as string
        if (firstKey) {
          state.expandedKeys = [firstKey]
          this.expandNode(serverId, firstKey)
        }
      }
    },

    async disconnectAll() {
      await connectionsProxy.DisconnectAll()
      this.connections = []
      this.serverTreeStates = {}
      const tabStore = useTabStore()
      tabStore.removeAllTabs()
    },
    async disconnect(serverId: string) {
      const server = this.connections.find((x) => x.serverID === serverId)
      if (server != null) {
        await connectionsProxy.Disconnect(serverId)
        this.connections = this.connections.filter((x) => x.serverID !== serverId)
        delete this.serverTreeStates[serverId]
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
      if (connection == null) {
        return []
      }

      if (!force && connection.databases != null && connection.databases.length > 0) {
        return connection.databases
      }

      const databases = await connectionsProxy.GetDatabases(serverId)
      if (databases.isSuccess) {
        connection.databases = databases.data.map((db) => ({ name: db, collections: [], views: [] }))
        return connection.databases
      }

      const notifier = useNotifier()
      notifier.error(`error retrieving databases: ${databases.error}`)
      return []
    },
    async getCollectionList(
      serverId: string,
      dbName: string,
      force: boolean = false,
    ): Promise<Collection[]> {
      const connection = this.connections.find((x) => x.serverID === serverId)
      if (connection == null || connection.databases == null) {
        return []
      }

      const database = connection.databases.find((x) => x.name === dbName)
      if (database == null) {
        return []
      }

      if (!force && database.collections != null && database.collections.length > 0) {
        return database.collections
      }

      const collections = await connectionsProxy.GetCollections(serverId, dbName)
      if (collections.isSuccess) {
        database.collections = collections.data.map((col) => ({ name: col, indexes: [] }))
        return database.collections
      }

      const notifier = useNotifier()
      notifier.error(`error retrieving collections: ${collections.error}`)
      return []
    },
    async getViewList(serverId: string, dbName: string, force: boolean = false): Promise<string[]> {
      const connection = this.connections.find((x) => x.serverID === serverId)
      if (connection == null || connection.databases == null) {
        return []
      }

      const database = connection.databases.find((x) => x.name === dbName)
      if (database == null) {
        return []
      }

      if (!force && database.views != null) {
        return database.views
      }

      const views = await connectionsProxy.GetViews(serverId, dbName)
      if (views.isSuccess) {
        database.views = views.data
        return database.views
      }

      const notifier = useNotifier()
      notifier.error(`error retrieving views: ${views.error}`)
      return []
    },
    findDatabase(serverId: string, dbName: string): Database | undefined {
      const connection = this.connections.find((x) => x.serverID === serverId)
      if (connection?.databases != null) {
        return connection.databases.find((x) => x.name === dbName)
      }
      return undefined
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
