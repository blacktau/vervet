import { defineStore } from 'pinia'
import { isEmpty } from 'lodash'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useNotifier } from '@/utils/dialog.ts'
import { i18nGlobal } from '@/i18n'
import { type models } from 'wailsjs/go/models.ts'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { type NamespaceRow } from '@/features/data-browser/namespaceSearch.ts'

export type InventoryStatus = 'idle' | 'building' | 'ready' | 'error'

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
  selectedKeys: string[]
  treeData: TreeNode[]
}

const FOLDER_COLLECTIONS = 'Collections'
const FOLDER_VIEWS = 'Views'

// Not reactive state, so it lives outside the Pinia store. Guards against a
// stale buildInventory response clobbering a newer one when calls overlap.
const inventoryGenerations = new Map<string, number>()

export const useDataBrowserStore = defineStore('browser', {
  state: () => ({
    connections: [] as ServerConnection[],
    serverTreeStates: {} as Record<string, ServerTreeState>,
    inventoryStatus: {} as Record<string, InventoryStatus>,
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
    getInventoryStatus(): (serverId: string) => InventoryStatus {
      return (serverId: string) => this.inventoryStatus[serverId] ?? 'idle'
    },
    currentSelectedKeys(): string[] {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return []
      }
      return this.serverTreeStates[serverId]?.selectedKeys ?? []
    },
    // Flattens the cached database structures into search rows. Derived rather
    // than stored, so every path that already mutates connection.databases —
    // add, drop, rename, refresh — keeps the index correct for free.
    searchIndex(): NamespaceRow[] {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return []
      }

      const connection = this.connections.find((c) => c.serverID === serverId)
      if (!connection?.databases) {
        return []
      }

      const rows: NamespaceRow[] = []
      for (const database of connection.databases) {
        rows.push({
          serverID: serverId,
          db: database.name,
          name: database.name,
          type: DataNodeType.Database,
          path: database.name,
        })
        for (const collection of database.collections) {
          rows.push({
            serverID: serverId,
            db: database.name,
            name: collection.name,
            type: DataNodeType.Collection,
            path: `${database.name}.${collection.name}`,
          })
        }
        for (const view of database.views) {
          rows.push({
            serverID: serverId,
            db: database.name,
            name: view,
            type: DataNodeType.View,
            path: `${database.name}.${view}`,
          })
        }
      }
      return rows
    },
  },
  actions: {
    getOrCreateTreeState(serverId: string): ServerTreeState {
      if (!this.serverTreeStates[serverId]) {
        this.serverTreeStates[serverId] = {
          expandedKeys: [],
          loadedKeys: [],
          selectedKeys: [],
          treeData: [],
        }
      }
      return this.serverTreeStates[serverId]
    },

    setSelectedKeys(keys: string[]) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }
      const state = this.getOrCreateTreeState(serverId)
      state.selectedKeys = keys
    },

    // Opens a query tab for a tree key. Lives in the store rather than the tree
    // component because the global find overlay needs it too.
    openQueryForKey(key: string, type: DataNodeType) {
      const tabStore = useTabStore()
      const parts = key.split(':')
      const serverId = parts[0]
      const dbName = parts[1]

      if (type === DataNodeType.Database) {
        if (serverId && dbName) {
          tabStore.openQuery(serverId, dbName)
        }
        return
      }

      if (type === DataNodeType.Collection || type === DataNodeType.View) {
        const name = parts[3]
        if (serverId && dbName && name) {
          const settingsStore = useSettingsStore()
          const queryText = `db.getCollection('${name}').find({}).limit(${settingsStore.query.defaultLimit})`
          tabStore.openQuery(serverId, dbName, queryText, name)
        }
      }
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

      await Promise.all([
        this.getCollectionList(sid, dbName, true),
        this.getViewList(sid, dbName, true),
      ])

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

      const collectionsFolder = node.children.find((c) => c.label === FOLDER_COLLECTIONS)
      if (collectionsFolder) {
        const folderKey = collectionsFolder.key as string
        const database = this.findDatabase(sid, dbName)
        if (database?.collections) {
          collectionsFolder.children = database.collections.map((col) => ({
            label: col.name,
            key: `${folderKey}:${col.name}`,
            isLeaf: true,
            type: DataNodeType.Collection,
            children: [],
          }))
        }
        state.loadedKeys = [...state.loadedKeys, dbKey, folderKey]
        state.expandedKeys = [...state.expandedKeys, dbKey, folderKey]
      } else {
        state.loadedKeys = [...state.loadedKeys, dbKey]
        state.expandedKeys = [...state.expandedKeys, dbKey]
      }
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
          this.expandNode(serverId, firstKey)
        }
      }
    },

    // Expands every ancestor of key in order, awaiting each so an unloaded
    // branch is fetched on the way down, then selects the target. Used by
    // global find, where the target is routinely in a branch the user has
    // never opened.
    async revealNode(serverId: string, key: string) {
      const state = this.getOrCreateTreeState(serverId)
      if (state.treeData.length === 0) {
        state.treeData = this.buildTreeForServer(serverId)
      }

      const parts = key.split(':')
      const ancestorKeys: string[] = []
      for (let i = 1; i < parts.length; i++) {
        ancestorKeys.push(parts.slice(0, i).join(':'))
      }

      for (const ancestorKey of ancestorKeys) {
        await this.expandNode(serverId, ancestorKey)
      }

      const missing = ancestorKeys.filter(
        (k) => state.loadedKeys.includes(k) && !state.expandedKeys.includes(k),
      )
      if (missing.length > 0) {
        state.expandedKeys = [...state.expandedKeys, ...missing]
      }

      this.setSelectedKeys([key])
    },

    async disconnectAll() {
      const result = await connectionsProxy.DisconnectAll()
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.warning(i18nGlobal.t('errorTitles.disconnect'), { detail: result.errorDetail })
      }
      // Always clean up frontend state
      this.connections = []
      this.serverTreeStates = {}
      const tabStore = useTabStore()
      tabStore.removeAllTabs()
    },
    async disconnect(serverId: string) {
      const server = this.connections.find((x) => x.serverID === serverId)
      if (server != null) {
        const result = await connectionsProxy.Disconnect(serverId)
        if (!result.isSuccess) {
          const notifier = useNotifier()
          notifier.warning(i18nGlobal.t('errorTitles.disconnect'), { detail: result.errorDetail })
        }
      }
      this._cleanupConnection(serverId)
      return true
    },
    // Drop a connection's frontend state without calling the backend. Used by
    // the backend-emitted "connection-disconnected" event, and as the shared
    // cleanup path for user-initiated disconnect. Always invoked so that the
    // tab close button cannot get wedged when the connections list has
    // drifted from the backend.
    _cleanupConnection(serverId: string) {
      this.connections = this.connections.filter((x) => x.serverID !== serverId)
      delete this.serverTreeStates[serverId]
      useTabStore().removeTabById(serverId)
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

      const databases = await databasesProxy.GetDatabases(serverId)
      if (databases.isSuccess) {
        connection.databases = databases.data.map((db) => ({ name: db, collections: [], views: [] }))
        return connection.databases
      }

      const notifier = useNotifier()
      notifier.error(i18nGlobal.t(`errors.${databases.errorCode}`), { title: i18nGlobal.t('errorTitles.listDatabases'), detail: databases.errorDetail })
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

      const collections = await collectionsProxy.GetCollections(serverId, dbName)
      if (collections.isSuccess) {
        database.collections = collections.data.map((col) => ({ name: col, indexes: [] }))
        return database.collections
      }

      const notifier = useNotifier()
      notifier.error(i18nGlobal.t(`errors.${collections.errorCode}`), { title: i18nGlobal.t('errorTitles.listCollections'), detail: collections.errorDetail })
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

      const views = await collectionsProxy.GetViews(serverId, dbName)
      if (views.isSuccess) {
        database.views = views.data
        return database.views
      }

      const notifier = useNotifier()
      notifier.error(i18nGlobal.t(`errors.${views.errorCode}`), { title: i18nGlobal.t('errorTitles.listViews'), detail: views.errorDetail })
      return []
    },
    // Fetches the server's whole namespace inventory in one call and merges it
    // into the cached database structures. Deliberately not awaited by connect:
    // a large server must not delay the UI.
    async buildInventory(serverId: string) {
      const connection = this.connections.find((c) => c.serverID === serverId)
      if (connection == null) {
        return
      }

      const generation = (inventoryGenerations.get(serverId) ?? 0) + 1
      inventoryGenerations.set(serverId, generation)

      this.inventoryStatus[serverId] = 'building'

      const result = await collectionsProxy.GetNamespaceInventory(serverId)

      // A newer call for the same server started after this one and owns the
      // outcome now; drop this stale response instead of clobbering it.
      if (inventoryGenerations.get(serverId) !== generation) {
        return
      }

      if (!result.isSuccess) {
        this.inventoryStatus[serverId] = 'error'
        return
      }

      connection.databases = result.data.databases.map((db) => ({
        name: db.name,
        collections: (db.collections ?? []).map((name) => ({ name, indexes: [] })),
        views: db.views ?? [],
      }))

      this.inventoryStatus[serverId] = 'ready'
    },
    findDatabase(serverId: string, dbName: string): Database | undefined {
      const connection = this.connections.find((x) => x.serverID === serverId)
      if (connection?.databases != null) {
        return connection.databases.find((x) => x.name === dbName)
      }
      return undefined
    },
    async refreshDatabaseCollections(serverId: string, dbName: string) {
      const state = this.serverTreeStates[serverId]
      if (!state) {
        return
      }

      const dbKey = `${serverId}:${dbName}`
      // Remove the database key from loadedKeys so expandDatabase re-fetches
      state.loadedKeys = state.loadedKeys.filter((k) => k !== dbKey)

      // Find the database node and re-expand it
      const dbNode = this.findNode(state.treeData, dbKey)
      if (dbNode) {
        await this.expandDatabase(serverId, dbNode)
        // Trigger reactivity
        state.treeData = [...state.treeData]
      }
    },
    async refreshServerDatabases(serverId: string) {
      const state = this.serverTreeStates[serverId]
      if (!state) {
        return
      }

      // Remove the server key from loadedKeys so expandServer re-fetches
      state.loadedKeys = state.loadedKeys.filter((k) => k !== serverId)

      // Find the server node and re-expand it
      const serverNode = this.findNode(state.treeData, serverId)
      if (serverNode) {
        await this.expandServer(serverId, serverNode)
        // Trigger reactivity
        state.treeData = [...state.treeData]
      }

      this.buildInventory(serverId)
    },
    async connect(serverId: string, reload: boolean = false) {
      if (this.isConnected(serverId)) {
        if (!reload) {
          await this.disconnect(serverId)
          return {
            success: true,
            serverId: serverId,
            name: '',
          }
        }
        return {
          success: true,
          serverId: serverId,
          name: this.connections.find((x) => x.serverID === serverId)?.name || '',
        }
      }

      const result = await connectionsProxy.Connect(serverId)
      if (!result.isSuccess) {
        if (result.errorCode !== 'oidc_login_canceled') {
          const notifier = useNotifier()
          notifier.error(i18nGlobal.t(`errors.${result.errorCode}`), { title: i18nGlobal.t('errorTitles.connect'), detail: result.errorDetail })
        }
        return {
          success: false,
        }
      }

      await this.refreshConnectedServers(true)
      // Fire and forget: indexing a large server must not block the connect.
      this.buildInventory(serverId)
      return {
        success: true,
        serverId: serverId,
        name: result.data.name,
      }
    },
  },
})
