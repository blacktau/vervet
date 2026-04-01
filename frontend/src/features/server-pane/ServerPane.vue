<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { computed, ref, watch } from 'vue'
import IconButton from '@/features/common/IconButton.vue'
import ServerTree from '@/features/server-pane/ServerTree.vue'
import ExportDialog from '@/features/server-pane/ExportDialog.vue'
import { useServerStore, type RegisteredServerNode } from '@/features/server-pane/serverStore.ts'
import { useNotifier } from '@/utils/dialog'
import {
  EllipsisHorizontalIcon,
  FunnelIcon,
  PlusIcon,
  FolderPlusIcon
} from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const serverStore = useServerStore()
const notifier = useNotifier()
const filterPattern = ref('')
const serverTreeRef = ref<InstanceType<typeof ServerTree>>()

const menuOptions = computed(() => [
  {
    key: 'import',
    label: t('serverPane.importServersMenuItem'),
  },
  {
    key: 'export',
    label: t('serverPane.exportServersMenuItem'),
  },
])

async function handleMenuSelect(key: string) {
  if (key === 'import') {
    const result = await serverStore.importServers()
    if (!result.success) {
      notifier.error(result.msg)
    }
  } else if (key === 'export') {
    dialogStore.showNewDialog(DialogType.Export, { serverIDs: null })
  }
}

async function handleExport(includeSensitiveData: boolean) {
  const exportData = dialogStore.dialogs[DialogType.Export]?.data as { serverIDs: string[] | null } | undefined
  const serverIDs = exportData?.serverIDs ?? serverStore.serverTree.map((s) => s.id)
  const result = await serverStore.exportServers(serverIDs, includeSensitiveData)
  if (!result.success) {
    notifier.error(result.msg)
  }
}

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
        @click="dialogStore.openNewGroupDialog()"
        style="margin-right: 6px" />
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
      <n-dropdown
        trigger="click"
        :options="menuOptions"
        @select="handleMenuSelect">
        <n-button text :focusable="false" class="nav-pane-func-btn">
          <template #icon>
            <n-icon :size="20">
              <EllipsisHorizontalIcon :stroke-width="3.5" />
            </n-icon>
          </template>
        </n-button>
      </n-dropdown>
    </div>
    <export-dialog :on-export="handleExport" />
  </div>
</template>

<style scoped lang="scss">
.nav-pane-bottom {
  color: v-bind('themeVars.iconColor');
  border-top: v-bind('themeVars.borderColor') 1px solid;
}
</style>
