<script lang="ts" setup>
import { ref, watch } from 'vue'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

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

async function handleExpand(keys: Array<string>) {
  const oldExpanded = expandedKeys.value
  expandedKeys.value = keys

  for (const key of keys) {
    if (oldExpanded.includes(key)) {
      continue
    }

    const node = findNode(treeData.value, key)

    if (node == null) {
      continue
    }

    if (node.type === DataNodeType.Server) {
      if (loadedKeys.value.includes(key)) {
        continue
      }

      await browserStore.getDatabaseList(key, true)
      const connection = browserStore.connections.find((x) => x.serverID === key)
      if (connection?.databases == null) {
        continue
      }

      node.children = connection.databases.map((db) => ({
        label: db.name,
        key: `${key}:${db.name}`,
        isLeaf: false,
        type: DataNodeType.Database,
        children: [],
      }))
      loadedKeys.value = [...loadedKeys.value, key]
      continue
    }

    if (node.type === DataNodeType.Database) {
      const dbKey = node.key as string
      if (dbKey == null || loadedKeys.value.includes(dbKey)) {
        continue
      }
      const [serverId, dbName] = dbKey.split(':')
      if (serverId == null || dbName == null) {
        continue
      }
      await browserStore.getCollectionList(serverId, dbName, true)
      const database = browserStore.findDatabase(serverId, dbName)
      if (database?.collections != null) {
        node.children = database.collections.map((col) => ({
          label: col.name,
          key: `${serverId}:${dbName}:${col.name}`,
          isLeaf: true,
          type: DataNodeType.Collection,
        }))
        loadedKeys.value = [...loadedKeys.value, dbKey]
      }
    }
  }
}

function findNode(nodes: DataTreeNode[], key: string | number): DataTreeNode | null {
  for (const node of nodes) {
    if (node.key === key) {
      return node
    }

    if (node.children) {
      const found = findNode(node.children as DataTreeNode[], key)
      if (found) return found
    }
  }
  return null
}
</script>

<template>
  <div class="browser-tree-wrapper" @contextmenu="(e) => e.preventDefault()">
    <n-tree
      :data="treeData"
      :expanded-keys="expandedKeys"
      :loaded-keys="loadedKeys"
      block-line
      @update:expanded-keys="handleExpand"></n-tree>
  </div>
</template>

<style lang="scss" scoped>
.browser-tree-wrapper {
  height: 100%;
  overflow: hidden;
}
</style>
