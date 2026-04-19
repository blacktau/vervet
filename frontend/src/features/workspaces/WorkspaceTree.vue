<script lang="ts" setup>
import { h, ref } from 'vue'
import { NInput, type TreeOption } from 'naive-ui'
import { useWorkspaceStore } from '@/features/workspaces/workspaceStore'
import { useDialoger, useNotifier } from '@/utils/dialog'
import { useDialogStore } from '@/stores/dialog'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'
import { useI18n } from 'vue-i18n'
import * as workspacesProxy from 'wailsjs/go/api/WorkspacesProxy'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()
const dialoger = useDialoger()
const dialogStore = useDialogStore()
const tabStore = useTabStore()
const queryStore = useQueryStore()
const browserStore = useDataBrowserStore()
const notifier = useNotifier()

const contextMenuX = ref(0)
const contextMenuY = ref(0)
const showContextMenu = ref(false)
const contextMenuOptions = ref<Array<{ label?: string; key: string; type?: string }>>([])
const contextNode = ref<TreeOption | null>(null)

function isRootFolder(node: TreeOption): boolean {
  if (!workspaceStore.activeWorkspace) {
    return false
  }
  return workspaceStore.activeWorkspace.folders.includes(node.key as string)
}

async function handleLoad(node: TreeOption): Promise<void> {
  const path = node.key as string
  const children = await workspaceStore.loadDirectory(path)
  node.children = children
}

function handleExpandedKeysUpdate(keys: string[]) {
  workspaceStore.expandedKeys = keys
}

function nodeProps(info: { option: TreeOption }) {
  return {
    onDblclick: () => {
      if (info.option.isLeaf) {
        openFile(info.option.key as string)
      }
    },
    onContextmenu: (event: MouseEvent) => {
      handleContextMenu({ event, option: info.option })
    },
  }
}

function openFile(filePath: string) {
  const currentTab = tabStore.currentTab
  if (currentTab && browserStore.isConnected(currentTab.serverId)) {
    // Find the database from the active query tab, if any
    const activeQueryId = currentTab.activeInnerTabId
    const activeQuery = activeQueryId
      ? currentTab.queries.find((q) => q.id === activeQueryId)
      : currentTab.queries[0]

    const database = activeQuery
      ? (queryStore.getQueryState(activeQuery.id).selectedDatabase || activeQuery.database)
      : undefined

    if (database) {
      const queryId = tabStore.openQuery(currentTab.serverId, database)
      if (queryId) {
        queryStore.loadFileByPath(queryId, filePath)
      }
      return
    }
  }

  dialogStore.openServerPickerDialog({ filePath })
}

function handleContextMenu(info: { event: MouseEvent; option: TreeOption }) {
  info.event.preventDefault()
  contextNode.value = info.option
  contextMenuX.value = info.event.clientX
  contextMenuY.value = info.event.clientY

  if (info.option.isLeaf) {
    // File context menu
    contextMenuOptions.value = [
      { label: t('workspaces.openFile'), key: 'open' },
      { label: t('workspaces.renameFile'), key: 'rename' },
      { type: 'divider', key: 'd1' },
      { label: t('workspaces.deleteFile'), key: 'delete' },
    ]
  } else if (isRootFolder(info.option)) {
    // Root folder context menu
    contextMenuOptions.value = [
      { label: t('workspaces.newFile'), key: 'newFile' },
      { label: t('workspaces.newFolder'), key: 'newFolder' },
      { type: 'divider', key: 'd1' },
      { label: t('workspaces.removeFolder'), key: 'removeFolder' },
    ]
  } else {
    // Subfolder context menu
    contextMenuOptions.value = [
      { label: t('workspaces.newFile'), key: 'newFile' },
      { label: t('workspaces.newFolder'), key: 'newFolder' },
    ]
  }

  showContextMenu.value = true
}

function handleContextMenuSelect(key: string) {
  showContextMenu.value = false
  const node = contextNode.value
  if (!node) {
    return
  }

  if (key === 'open') {
    openFile(node.key as string)
  }

  if (key === 'newFile') {
    handleNewFile(node)
  }

  if (key === 'newFolder') {
    handleNewFolder(node)
  }

  if (key === 'rename') {
    handleRename(node)
  }

  if (key === 'delete') {
    handleDelete(node)
  }

  if (key === 'removeFolder') {
    handleRemoveFolder(node)
  }
}

function handleClickOutside() {
  showContextMenu.value = false
}

function handleNewFile(node: TreeOption) {
  const nameRef = ref('')
  dialoger.show({
    title: t('workspaces.newFile'),
    positiveText: t('common.create'),
    negativeText: t('common.cancel'),
    content: () => h(NInput, {
      value: nameRef.value,
      onUpdateValue: (v: string) => { nameRef.value = v },
      placeholder: t('workspaces.fileName'),
      autofocus: true,
    }),
    onPositiveClick: async () => {
      const fileName = nameRef.value.trim()
      if (!fileName) {
        return
      }

      const dirPath = node.key as string
      const result = await workspacesProxy.CreateFile(dirPath, fileName)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || result.errorCode)
        return
      }

      await workspaceStore.refreshTree()
      openFile(result.data as string)
    },
  })
}

function handleNewFolder(node: TreeOption) {
  const nameRef = ref('')
  dialoger.show({
    title: t('workspaces.newFolder'),
    positiveText: t('common.create'),
    negativeText: t('common.cancel'),
    content: () => h(NInput, {
      value: nameRef.value,
      onUpdateValue: (v: string) => { nameRef.value = v },
      placeholder: t('workspaces.folderName'),
      autofocus: true,
    }),
    onPositiveClick: async () => {
      const folderName = nameRef.value.trim()
      if (!folderName) {
        return
      }

      const dirPath = node.key as string
      const result = await workspacesProxy.CreateFolder(dirPath, folderName)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || result.errorCode)
        return
      }

      await workspaceStore.refreshTree()
    },
  })
}

function handleRename(node: TreeOption) {
  const nameRef = ref(node.label as string)
  dialoger.show({
    title: t('workspaces.renameFile'),
    positiveText: t('common.save'),
    negativeText: t('common.cancel'),
    content: () => h(NInput, {
      value: nameRef.value,
      onUpdateValue: (v: string) => { nameRef.value = v },
      placeholder: t('workspaces.fileName'),
      autofocus: true,
    }),
    onPositiveClick: async () => {
      const newName = nameRef.value.trim()
      if (!newName || newName === node.label) {
        return
      }

      const oldPath = node.key as string
      const parentDir = oldPath.substring(0, oldPath.lastIndexOf('/'))
      const newPath = `${parentDir}/${newName}`

      const result = await workspacesProxy.RenameFile(oldPath, newPath)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || result.errorCode)
        return
      }

      queryStore.renameFilePath(oldPath, newPath)
      await workspaceStore.refreshTree()
    },
  })
}

function handleDelete(node: TreeOption) {
  dialoger.show({
    type: 'warning',
    title: t('workspaces.deleteFile'),
    content: t('workspaces.deleteFileConfirm', { name: node.label }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      const result = await workspacesProxy.DeleteFile(node.key as string)
      if (!result.isSuccess) {
        notifier.error(result.errorDetail || result.errorCode)
        return
      }

      await workspaceStore.refreshTree()
    },
  })
}

function handleRemoveFolder(node: TreeOption) {
  dialoger.show({
    type: 'warning',
    title: t('workspaces.removeFolder'),
    content: t('workspaces.removeFolderConfirm', { name: node.label }),
    positiveText: t('common.remove'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await workspaceStore.removeFolder(node.key as string)
    },
  })
}
</script>

<template>
  <div class="workspace-tree-container">
    <n-tree
      :data="workspaceStore.treeData"
      :expanded-keys="workspaceStore.expandedKeys"
      :on-load="handleLoad"
      :node-props="nodeProps"
      block-line
      selectable
      @update:expanded-keys="handleExpandedKeysUpdate" />

    <n-dropdown
      :options="contextMenuOptions"
      :show="showContextMenu"
      :x="contextMenuX"
      :y="contextMenuY"
      placement="bottom-start"
      trigger="manual"
      @clickoutside="handleClickOutside"
      @select="handleContextMenuSelect" />
  </div>
</template>

<style lang="scss" scoped>
.workspace-tree-container {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
}
</style>
