import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { DataNodeType, type ContextMenuOption, type DataTreeNode } from '@/features/data-browser/types.ts'

export function useDataTreeContextMenu() {
  const { t } = useI18n()
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
      label: t('dataBrowser.contextMenu.addDatabase'),
      key: 'addDatabase',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.serverStatus'),
      key: 'serverStatus',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.disconnect'),
      key: 'disconnect',
      disabled: false,
    },
  ])

  const databaseMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: t('dataBrowser.contextMenu.openQuery'),
      key: 'openQuery',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.dropDatabase'),
      key: 'dropDatabase',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.statistics'),
      key: 'statistics',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.refresh'),
      key: 'refresh',
      disabled: false,
    },
  ])

  const collectionsFolderMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: t('dataBrowser.contextMenu.addCollection'),
      key: 'addCollection',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.refresh'),
      key: 'refresh',
      disabled: false,
    },
  ])

  const viewsFolderMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: t('dataBrowser.contextMenu.refresh'),
      key: 'refresh',
      disabled: false,
    },
  ])

  const collectionMenuOptions = computed<ContextMenuOption[]>(() => [
    {
      label: t('dataBrowser.contextMenu.openQuery'),
      key: 'openQuery',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.rename'),
      key: 'rename',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.dropCollection'),
      key: 'dropCollection',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.viewIndexes'),
      key: 'viewIndexes',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.statistics'),
      key: 'statistics',
      disabled: false,
    },
    {
      label: t('dataBrowser.contextMenu.refresh'),
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
