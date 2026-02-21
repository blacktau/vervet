import { ref, computed } from 'vue'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'

interface TreeNode extends DataTreeNode {
  children: TreeNode[]
}

const FOLDER_COLLECTIONS = 'Collections'
const FOLDER_VIEWS = 'Views'

export function useDataTree() {
  const browserStore = useDataBrowserStore()
  const tabStore = useTabStore()

  const expandedKeys = ref<string[]>([])
  const loadedKeys = ref<string[]>([])
  const treeData = ref<TreeNode[]>([])

  const currentServerId = computed(() => tabStore.currentTabId)

  function buildTreeForServer(serverId: string): TreeNode[] {
    const connection = browserStore.connections.find((c) => c.serverID === serverId)
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
  }

  async function expandNode(key: string) {
    const node = findNode(treeData.value, key)
    if (!node) return

    if (node.type === DataNodeType.Server) {
      await expandServer(node)
      return
    }

    if (node.type === DataNodeType.Database) {
      await expandDatabase(node)
      return
    }

    if (node.type === DataNodeType.Folder) {
      await expandFolder(node)
    }
  }

  async function expandServer(node: TreeNode) {
    const key = node.key as string

    if (loadedKeys.value.includes(key)) {
      return
    }

    await browserStore.getDatabaseList(key, true)
    const connection = browserStore.connections.find((c) => c.serverID === key)
    if (!connection?.databases) return

    node.children = connection.databases.map((db) => ({
      label: db.name,
      key: `${key}:${db.name}`,
      isLeaf: false,
      type: DataNodeType.Database,
      children: [],
    }))

    loadedKeys.value = [...loadedKeys.value, key]
    expandedKeys.value = [...expandedKeys.value, key]
  }

  async function expandDatabase(node: TreeNode) {
    const dbKey = node.key as string
    if (!dbKey || loadedKeys.value.includes(dbKey)) return

    const [serverId, dbName] = dbKey.split(':')
    if (!serverId || !dbName) return

    await browserStore.getCollectionList(serverId, dbName, true)
    await browserStore.getViewList(serverId, dbName, true)

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

    loadedKeys.value = [...loadedKeys.value, dbKey]
    expandedKeys.value = [...expandedKeys.value, dbKey]
  }

  async function expandFolder(node: TreeNode) {
    const folderKey = node.key as string
    if (!folderKey || loadedKeys.value.includes(folderKey)) return

    const parts = folderKey.split(':')
    if (parts.length < 3) return

    const serverId = parts[0]
    const dbName = parts[1]
    const folderName = parts[2]

    const isCollectionsFolder = folderName === FOLDER_COLLECTIONS
    const isViewsFolder = folderName === FOLDER_VIEWS

    if (!isCollectionsFolder && !isViewsFolder) return

    const database = browserStore.findDatabase(serverId, dbName)
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

    loadedKeys.value = [...loadedKeys.value, folderKey]
    expandedKeys.value = [...expandedKeys.value, folderKey]
  }

  function findNode(nodes: TreeNode[], key: string | number): TreeNode | null {
    for (const node of nodes) {
      if (node.key === key) {
        return node
      }
      if (node.children.length > 0) {
        const found = findNode(node.children, key)
        if (found) return found
      }
    }
    return null
  }

  function handleExpand(keys: string[]) {
    const oldExpanded = expandedKeys.value
    expandedKeys.value = keys

    for (const key of keys) {
      if (oldExpanded.includes(key)) continue
      expandNode(key)
    }
  }

  function updateTreeForCurrentServer() {
    const serverId = currentServerId.value
    if (!serverId) {
      treeData.value = []
      expandedKeys.value = []
      loadedKeys.value = []
      return
    }

    treeData.value = buildTreeForServer(serverId)

    const firstKey = treeData.value[0]?.key as string
    if (firstKey) {
      expandedKeys.value = [firstKey]
      expandNode(firstKey)
    }
  }

  return {
    treeData,
    expandedKeys,
    loadedKeys,
    currentServerId,
    handleExpand,
    updateTreeForCurrentServer,
  }
}
