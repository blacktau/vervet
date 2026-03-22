<script lang="ts" setup>
import { h, ref } from 'vue'
import { NInput, type TreeOption } from 'naive-ui'
import { useWorkspaceStore } from '@/features/workspaces/workspaceStore'
import { useDialoger } from '@/utils/dialog'
import { useDialogStore } from '@/stores/dialog'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()
const dialoger = useDialoger()
const dialogStore = useDialogStore()

const contextMenuX = ref(0)
const contextMenuY = ref(0)
const showContextMenu = ref(false)
const contextMenuOptions = ref<Array<{ label: string; key: string }>>([])
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

function handleNodeDblClick(info: { option: TreeOption }) {
  const node = info.option
  if (node.isLeaf) {
    openFile(node.key as string)
  }
}

function openFile(filePath: string) {
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
      { label: t('workspaces.deleteFile'), key: 'delete' },
    ]
  } else if (isRootFolder(info.option)) {
    // Root folder context menu
    contextMenuOptions.value = [
      { label: t('workspaces.newFile'), key: 'newFile' },
      { label: t('workspaces.removeFolder'), key: 'removeFolder' },
    ]
  } else {
    // Subfolder context menu
    contextMenuOptions.value = [
      { label: t('workspaces.newFile'), key: 'newFile' },
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
    type: 'info',
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
      const filePath = `${dirPath}/${fileName}`
      openFile(filePath)
    },
  })
}

function handleRename(node: TreeOption) {
  const nameRef = ref(node.label as string)
  dialoger.show({
    type: 'info',
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
      if (!newName) {
        return
      }
      // Rename will be handled by a future proxy call
      await workspaceStore.refreshTree()
    },
  })
}

function handleDelete(node: TreeOption) {
  dialoger.warning({
    title: t('workspaces.deleteFile'),
    content: t('workspaces.deleteFileConfirm', { name: node.label }),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      // Delete will be handled by a future proxy call
      await workspaceStore.refreshTree()
    },
  })
}

function handleRemoveFolder(node: TreeOption) {
  dialoger.warning({
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
      block-line
      selectable
      @update:expanded-keys="handleExpandedKeysUpdate"
      @node-props="(info) => ({
        onDblclick: () => handleNodeDblClick({ option: info.option }),
        onContextmenu: (event: MouseEvent) => handleContextMenu({ event, option: info.option }),
      })" />

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
