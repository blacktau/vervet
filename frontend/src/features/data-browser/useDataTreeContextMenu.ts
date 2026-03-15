import { computed, ref } from 'vue'
import { DataNodeType, type ContextMenuOption, type DataTreeNode } from '@/features/data-browser/types.ts'

export function useDataTreeContextMenu() {
  const selectedNode = ref<DataTreeNode | null>(null)
  const showMenu = ref(false)
  const position = ref({ x: 0, y: 0 })

  function openMenu(node: DataTreeNode, event: MouseEvent) {
    selectedNode.value = node
    showMenu.value = true
    position.value = { x: event.clientX, y: event.clientY }
  }

  function closeMenu() {
    showMenu.value = false
    selectedNode.value = null
  }

  const serverMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: 'Server Info',
      key: 'serverInfo',
      disabled: false,
    },
    {
      label: 'Add Database...',
      key: 'addDatabase',
      disabled: false,
    },
    {
      label: 'Statistics',
      key: 'statistics',
      disabled: false,
    },
    {
      label: 'Disconnect',
      key: 'disconnect',
      disabled: false,
    },
  ])

  const databaseMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: 'Open Query',
      key: 'openQuery',
      disabled: false,
    },
    {
      label: 'Drop Database',
      key: 'dropDatabase',
      disabled: false,
    },
    {
      label: 'Statistics',
      key: 'statistics',
      disabled: false,
    },
    {
      label: 'Refresh',
      key: 'refresh',
      disabled: false,
    },
  ])

  const collectionsFolderMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: 'Add Collection...',
      key: 'addCollection',
      disabled: false,
    },
    {
      label: 'Refresh',
      key: 'refresh',
      disabled: false,
    },
  ])

  const viewsFolderMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: 'Refresh',
      key: 'refresh',
      disabled: false,
    },
  ])

  const collectionMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: 'Open Query',
      key: 'openQuery',
      disabled: false,
    },
    {
      label: 'Rename...',
      key: 'rename',
      disabled: false,
    },
    {
      label: 'Drop Collection',
      key: 'dropCollection',
      disabled: false,
    },
    {
      label: 'View Indexes',
      key: 'viewIndexes',
      disabled: false,
    },
    {
      label: 'Statistics',
      key: 'statistics',
      disabled: false,
    },
    {
      label: 'Refresh',
      key: 'refresh',
      disabled: false,
    },
  ])

  const contextMenuOptions = computed<ContextMenuOption[]>(() => {
    if (!selectedNode.value) return []

    switch (selectedNode.value.type) {
      case DataNodeType.Server:
        return serverMenuOptions.value
      case DataNodeType.Database:
        return databaseMenuOptions.value
      case DataNodeType.Folder: {
        const folderKey = String(selectedNode.value.key)
        const isCollectionsFolder = folderKey.endsWith(':Collections')
        return isCollectionsFolder
          ? collectionsFolderMenuOptions.value
          : viewsFolderMenuOptions.value
      }
      case DataNodeType.Collection:
      case DataNodeType.View:
        return collectionMenuOptions.value
      default:
        return []
    }
  })

  return {
    selectedNode,
    show: showMenu,
    position,
    contextMenuOptions,
    openMenu,
    closeMenu,
  }
}
