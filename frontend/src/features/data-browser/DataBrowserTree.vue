<script lang="ts" setup>
import { ref, watch } from 'vue'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

const props = defineProps<{
  loading: boolean
  pattern?: string
}>()

const browserStore = useDataBrowserStore()

const expandedKeys = ref<string[]>([])
const loadedKeys = ref<string[]>([])
const treeData = ref<DataTreeNode[]>([])

watch(
  () => browserStore.connections,
  (connections) => {
    treeData.value = connections.map((x) => {
      return {
        label: x.name,
        key: x.serverID,
        isLeaf: false,
        type: DataNodeType.Server,
        children: [],
      } as DataTreeNode
    })
  },
  { immediate: true },
)

async function handleExpand(keys: Array<string | number>) {
  const oldExpanded = expandedKeys.value.map(String)
  expandedKeys.value = keys.map(String)

  for (const key of keys.map(String)) {
    if (oldExpanded.includes(key)) continue

    const node = findNode(treeData.value, key)

    if (node != null && node.type === DataNodeType.Server) {
      if (!loadedKeys.value.includes(key)) {
        await browserStore.getDatabaseList(key, true)
        const connection = browserStore.connections.find((x) => x.serverID === key)
        if (connection?.databases != null) {
          const children = connection.databases.map((db) => ({
            label: db.name,
            key: `${key}:${db.name}`,
            isLeaf: false,
            type: DataNodeType.Database,
            children: [],
          }))
          node.children = children
          loadedKeys.value = [...loadedKeys.value, key]
        }
      }
    } else if (node != null && node.type === DataNodeType.Database) {
      const dbKey = String(node.key)
      if (!loadedKeys.value.includes(dbKey)) {
        const [serverId, dbName] = dbKey.split(':')
        await browserStore.getCollectionList(serverId, dbName, true)
        const database = browserStore.findDatabase(serverId, dbName)
        if (database?.collections != null) {
          const children = database.collections.map((col) => ({
            label: col.name,
            key: `${serverId}:${dbName}:${col.name}`,
            isLeaf: true,
            type: DataNodeType.Collection,
          }))
          node.children = children
          loadedKeys.value = [...loadedKeys.value, dbKey]
        }
      }
    }
  }

  treeData.value = [...treeData.value]
}

function findNode(nodes: DataTreeNode[], key: string | number): DataTreeNode | null {
  for (const node of nodes) {
    if (String(node.key) === String(key)) return node
    if (node.children) {
      const found = findNode(node.children, key)
      if (found) return found
    }
  }
  return null
}
</script>

<template>
  <div class="browser-tree-wrapper" @contextmenu="(e) => e.preventDefault()">
    <n-tree
      block-line
      :data="treeData"
      :expanded-keys="expandedKeys"
      :loaded-keys="loadedKeys"
      @update:expanded-keys="handleExpand"
    ></n-tree>
  </div>
</template>

<style lang="scss" scoped>
.browser-tree-wrapper {
  height: 100%;
  overflow: hidden;
}
</style>
