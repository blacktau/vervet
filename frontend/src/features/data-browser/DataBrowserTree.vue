<script lang="ts" setup>
import { h, watch } from 'vue'
import { NIcon } from 'naive-ui'
import CollectionIcon from '@/features/icon/CollectionIcon.vue'
import { CircleStackIcon } from '@heroicons/vue/24/outline'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataTree } from '@/features/data-browser/useDataTree.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'

const tabStore = useTabStore()
const { treeData, expandedKeys, handleExpand, updateTreeForCurrentServer } = useDataTree()

const renderPrefix = ({ option }: { option: DataTreeNode }) => {
  if (option.type === DataNodeType.Collection) {
    return h(NIcon, { size: 18 }, () => h(CollectionIcon))
  }
  if (option.type === DataNodeType.Database) {
    return h(NIcon, { size: 18 }, () => h(CircleStackIcon))
  }
  return null
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
    <div v-if="!tabStore.currentTabId" class="empty-state">
      No server selected
    </div>
    <n-tree
      v-else
      :data="treeData"
      :expanded-keys="expandedKeys"
      :render-prefix="renderPrefix"
      block-line
      @update:expanded-keys="handleExpand" />
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
