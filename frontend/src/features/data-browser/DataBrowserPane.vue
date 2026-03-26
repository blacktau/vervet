<script lang="ts" setup>
import { useThemeVars } from 'naive-ui'
import { computed, ref, watch } from 'vue'
import DataBrowserTree from '@/features/data-browser/DataBrowserTree.vue'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { FunnelIcon } from '@heroicons/vue/24/outline'

const themeVars = useThemeVars()
const browserStore = useDataBrowserStore()
const filterPattern = ref('')
const treeRef = ref<InstanceType<typeof DataBrowserTree>>()

const collectMatchingNodes = (nodes: DataTreeNode[], pattern: string): DataTreeNode[] => {
  const lowerPattern = pattern.toLowerCase()
  const results: DataTreeNode[] = []
  for (const node of nodes) {
    if ((node.label as string).toLowerCase().includes(lowerPattern)) {
      results.push(node)
    }
    if (node.children) {
      results.push(...collectMatchingNodes(node.children as DataTreeNode[], pattern))
    }
  }
  return results
}

const isExpandable = (node: DataTreeNode) => {
  return node.type === DataNodeType.Server || node.type === DataNodeType.Database || node.type === DataNodeType.Folder
}

const matchingNodes = computed(() => {
  if (!filterPattern.value) {
    return []
  }
  return collectMatchingNodes(browserStore.currentTreeData as DataTreeNode[], filterPattern.value)
})

watch(matchingNodes, (matches) => {
  if (matches.length > 0 && treeRef.value) {
    treeRef.value.selectedKeys = [matches[0].key as string]
  }
})

const onFilterKeydown = (e: KeyboardEvent) => {
  const tree = treeRef.value
  if (!tree || matchingNodes.value.length === 0) {
    return
  }

  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const currentKey = tree.selectedKeys[0]
    const currentIndex = matchingNodes.value.findIndex((n) => n.key === currentKey)
    let nextIndex: number
    if (e.key === 'ArrowDown') {
      nextIndex = currentIndex < matchingNodes.value.length - 1 ? currentIndex + 1 : 0
    } else {
      nextIndex = currentIndex > 0 ? currentIndex - 1 : matchingNodes.value.length - 1
    }
    tree.selectedKeys = [matchingNodes.value[nextIndex].key as string]
  } else if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
    e.preventDefault()
    const currentKey = tree.selectedKeys[0]
    if (!currentKey) {
      return
    }
    const node = matchingNodes.value.find((n) => n.key === currentKey)
    if (node && isExpandable(node)) {
      tree.toggleExpandKey(currentKey)
    }
  } else if (e.key === 'Enter') {
    const currentKey = tree.selectedKeys[0]
    if (!currentKey) {
      return
    }
    const node = matchingNodes.value.find((n) => n.key === currentKey)
    if (!node) {
      return
    }
    if (node.type === DataNodeType.Server || node.type === DataNodeType.Folder) {
      tree.toggleExpandKey(currentKey)
    } else {
      tree.openQueryForNode(node)
    }
  }
}
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <data-browser-tree ref="treeRef" :filter-pattern="filterPattern" />
    <div class="nav-pane-bottom nav-pane-func flex-box-h">
      <n-input
        v-model:value="filterPattern"
        :autofocus="false"
        :placeholder="$t('dataBrowser.filter')"
        clearable
        @keydown="onFilterKeydown">
        <template #prefix>
          <n-icon :component="FunnelIcon" size="20" />
        </template>
      </n-input>
    </div>
  </div>
</template>

<style scoped lang="scss">
.nav-pane-bottom {
  color: v-bind('themeVars.iconColor');
  border-top: v-bind('themeVars.borderColor') 1px solid;
}
</style>
