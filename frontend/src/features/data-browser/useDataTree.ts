import { ref, computed } from 'vue'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'

interface TreeNode extends DataTreeNode {
  children: TreeNode[]
}

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

    for (const child of node.children) {
      await expandNode(child.key as string)
    }
  }

  async function expandDatabase(node: TreeNode) {
    const dbKey = node.key as string
    if (!dbKey || loadedKeys.value.includes(dbKey)) return

    const [serverId, dbName] = dbKey.split(':')
    if (!serverId || !dbName) return

    await browserStore.getCollectionList(serverId, dbName, true)
    const database = browserStore.findDatabase(serverId, dbName)
    if (!database?.collections) return

    node.children = database.collections.map((col) => ({
      label: col.name,
      key: `${serverId}:${dbName}:${col.name}`,
      isLeaf: true,
      type: DataNodeType.Collection,
      children: [],
    }))

    node.isLeaf = true
    loadedKeys.value = [...loadedKeys.value, dbKey]
    expandedKeys.value = [...expandedKeys.value, dbKey]
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
