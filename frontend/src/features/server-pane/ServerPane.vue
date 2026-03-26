<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { computed, ref, watch } from 'vue'
import IconButton from '@/features/common/IconButton.vue'
import ServerTree from '@/features/server-pane/ServerTree.vue'
import { useServerStore, type RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import {
  FunnelIcon,
  PlusIcon,
  FolderPlusIcon
} from '@heroicons/vue/24/outline'

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const serverStore = useServerStore()
const filterPattern = ref('')
const serverTreeRef = ref<InstanceType<typeof ServerTree>>()

const collectMatchingNodes = (nodes: RegisteredServerNode[], pattern: string): RegisteredServerNode[] => {
  const lowerPattern = pattern.toLowerCase()
  const results: RegisteredServerNode[] = []
  for (const node of nodes) {
    if (node.name.toLowerCase().includes(lowerPattern)) {
      results.push(node)
    }
    if (node.children) {
      results.push(...collectMatchingNodes(node.children, pattern))
    }
  }
  return results
}

const matchingNodes = computed(() => {
  if (!filterPattern.value) {
    return []
  }
  return collectMatchingNodes(serverStore.serverTree, filterPattern.value)
})

watch(matchingNodes, (matches) => {
  if (matches.length > 0 && serverTreeRef.value) {
    serverTreeRef.value.selectedKeys = [matches[0].id]
  }
})

const onFilterKeydown = (e: KeyboardEvent) => {
  const tree = serverTreeRef.value
  if (!tree || matchingNodes.value.length === 0) {
    return
  }

  if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
    e.preventDefault()
    const currentId = tree.selectedKeys[0]
    const currentIndex = matchingNodes.value.findIndex((s) => s.id === currentId)
    let nextIndex: number
    if (e.key === 'ArrowDown') {
      nextIndex = currentIndex < matchingNodes.value.length - 1 ? currentIndex + 1 : 0
    } else {
      nextIndex = currentIndex > 0 ? currentIndex - 1 : matchingNodes.value.length - 1
    }
    tree.selectedKeys = [matchingNodes.value[nextIndex].id]
  } else if (e.key === 'ArrowRight' || e.key === 'ArrowLeft') {
    e.preventDefault()
    const currentId = tree.selectedKeys[0]
    if (!currentId) {
      return
    }
    const node = matchingNodes.value.find((s) => s.id === currentId)
    if (node?.isGroup) {
      tree.expandKey(currentId)
    }
  } else if (e.key === 'Enter') {
    const currentId = tree.selectedKeys[0]
    if (!currentId) {
      return
    }
    const node = matchingNodes.value.find((s) => s.id === currentId)
    if (node?.isGroup) {
      tree.expandKey(currentId)
    } else if (node) {
      tree.connectToServer(currentId)
    }
  }
}
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <server-tree ref="serverTreeRef" :filter-pattern="filterPattern" />
    <div class="nav-pane-bottom nav-pane-func flex-box-h">
      <icon-button
        :button-class="['nav-pane-func-btn']"
        :icon="PlusIcon"
        :stroke-width="3.5"
        size="20"
        t-tooltip="serverPane.addServer"
        @click="dialogStore.showNewServerDialog()" />
      <icon-button
        :button-class="['nav-pane-func-btn']"
        :icon="FolderPlusIcon"
        :stroke-width="3.5"
        size="20"
        t-tooltip="serverPane.addGroup"
        @click="dialogStore.openNewGroupDialog()" />
      <n-divider vertical />
      <n-input
        v-model:value="filterPattern"
        :autofocus="false"
        :placeholder="$t('serverPane.filter')"
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
