<script lang="ts" setup>
import { h, watch } from 'vue'
import { NIcon } from 'naive-ui'
import CollectionIcon from '@/features/icon/CollectionIcon.vue'
import { CircleStackIcon, EyeIcon, FolderIcon, FolderOpenIcon } from '@heroicons/vue/24/outline'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataTree } from '@/features/data-browser/useDataTree.ts'
import { useDataTreeContextMenu } from '@/features/data-browser/useDataTreeContextMenu.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import DataTreeContextMenu from '@/features/data-browser/DataTreeContextMenu.vue'

const tabStore = useTabStore()
const browserStore = useDataBrowserStore()
const { treeData, expandedKeys, handleExpand, updateTreeForCurrentServer } = useDataTree()
const contextMenu = useDataTreeContextMenu()

const renderPrefix = ({ option }: { option: DataTreeNode }) => {
  if (option.type === DataNodeType.Database) {
    return h(NIcon, { size: 18 }, () => h(CircleStackIcon))
  }
  if (option.type === DataNodeType.Folder) {
    const isExpanded = expandedKeys.value.includes(option.key as string)
    const Icon = isExpanded ? FolderOpenIcon : FolderIcon
    return h(NIcon, { size: 18 }, () => h(Icon))
  }
  if (option.type === DataNodeType.Collection) {
    return h(NIcon, { size: 18 }, () => h(CollectionIcon))
  }
  if (option.type === DataNodeType.View) {
    return h(NIcon, { size: 18 }, () => h(EyeIcon))
  }
  return null
}

function handleContextMenuSelect(key: string) {
  const node = contextMenu.selectedNode.value
  if (!node) return

  if (key === 'disconnect' && node.type === DataNodeType.Server) {
    const serverId = node.key as string
    browserStore.disconnect(serverId)
  }
}

const nodeProps = ({ option }: { option: DataTreeNode }) => {
  return {
    onContextmenu(e: MouseEvent) {
      e.preventDefault()
      contextMenu.openMenu(option as DataTreeNode, e)
    },
  }
}

watch(
  () => tabStore.currentTabId,
  () => {
    updateTreeForCurrentServer()
  },
  { immediate: true },
)
</script>

<template>
  <div class="browser-tree-wrapper" @contextmenu="(e) => e.preventDefault()">
    <n-tree
      v-if="treeData.length > 0"
      :cancelable="false"
      :data="treeData"
      :expanded-keys="expandedKeys"
      :node-props="nodeProps"
      :render-prefix="renderPrefix"
      block-line
      block-node
      virtual-scroll
      @update:expanded-keys="handleExpand">
      <template #empty>
        <n-empty :description="$t('dataBrowser.tree.temp')" />
      </template>
    </n-tree>
    <DataTreeContextMenu
      :options="contextMenu.contextMenuOptions.value"
      :show="contextMenu.show.value"
      :x="contextMenu.position.value.x"
      :y="contextMenu.position.value.y"
      @close="contextMenu.closeMenu"
      @select="handleContextMenuSelect" />
  </div>
</template>

<style lang="scss" scoped>
.browser-tree-wrapper {
  height: 100%;
  overflow: hidden;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--n-text-color-3);
  font-size: 14px;
}
</style>
