import { markRaw, nextTick, reactive, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Cog8ToothIcon,
  DocumentDuplicateIcon,
  FolderPlusIcon,
  PencilSquareIcon,
  PlusCircleIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useDialoger, useMessager } from '@/utils/dialog.ts'
import PlugConnected from '@/features/icon/PlugConnected.vue'
import PlugDisconnected from '@/features/icon/PlugDisconnected.vue'

export const MenuKeys = {
  GroupAddServer: 'group_add_server',
  GroupAddSubGroup: 'group_add_sub_group',
  GroupRename: 'group_rename',
  GroupDelete: 'group_delete',
  ServerDisconnect: 'server_disconnect',
  ServerEdit: 'server_edit',
  ServerClone: 'server_clone',
  ServerConnect: 'server_connect',
  ServerDelete: 'server_delete',
} as const

type ContextMenuEntry = {
  key: string
  label: string
  icon?: unknown
  type?: 'divider'
}

export function useServerTreeContextMenu(
  selectedKeys: Ref<string[]>,
  connectToServer: (serverId: string) => Promise<void>,
) {
  const i18n = useI18n()
  const browserStore = useDataBrowserStore()
  const dialogStore = useDialogStore()
  const serverStore = useServerStore()

  const contextMenuParams = reactive<{
    show: boolean
    x: number
    y: number
    options: ContextMenuEntry[]
    currentNode?: RegisteredServerNode
  }>({
    show: false,
    x: 0,
    y: 0,
    options: [],
    currentNode: undefined,
  })

  const buildMenuOptions = (option: RegisteredServerNode): ContextMenuEntry[] => {
    if (option.isGroup) {
      return [
        {
          key: MenuKeys.GroupAddServer,
          label: 'serverPane.serverTree.addServerToGroup',
          icon: PlusCircleIcon,
        },
        {
          key: MenuKeys.GroupAddSubGroup,
          label: 'serverPane.serverTree.addSubGroup',
          icon: FolderPlusIcon,
        },
        {
          key: MenuKeys.GroupRename,
          label: 'serverPane.serverTree.renameGroup',
          icon: PencilSquareIcon,
        },
        { type: 'divider', key: 'd1', label: '' },
        {
          key: MenuKeys.GroupDelete,
          label: 'serverPane.serverTree.deleteGroup',
          icon: TrashIcon,
        },
      ]
    }

    const common: ContextMenuEntry[] = [
      {
        key: MenuKeys.ServerEdit,
        label: 'serverPane.serverTree.editServer',
        icon: Cog8ToothIcon,
      },
      {
        key: MenuKeys.ServerClone,
        label: 'serverPane.serverTree.cloneServer',
        icon: DocumentDuplicateIcon,
      },
      { type: 'divider', key: 'd1' },
      {
        key: MenuKeys.ServerDelete,
        label: 'serverPane.serverTree.deleteServer',
        icon: TrashIcon,
      },
    ]

    const connectOption: ContextMenuEntry = browserStore.isConnected(option.id)
      ? {
          key: MenuKeys.ServerDisconnect,
          label: 'serverPane.serverTree.disconnect',
          icon: PlugDisconnected,
        }
      : {
          key: MenuKeys.ServerConnect,
          label: 'serverPane.serverTree.connectServer',
          icon: PlugConnected,
        }

    return [connectOption, ...common]
  }

  const openContextMenu = (option: RegisteredServerNode, e: PointerEvent) => {
    e.preventDefault()
    contextMenuParams.show = false
    nextTick().then(() => {
      const menuOptions = buildMenuOptions(option)
      contextMenuParams.options = markRaw(menuOptions) as never
      contextMenuParams.currentNode = option
      contextMenuParams.x = e.clientX
      contextMenuParams.y = e.clientY
      contextMenuParams.show = true
      selectedKeys.value = [option.id]
    })
  }

  const deleteServer = async (serverId: string) => {
    const dialoger = useDialoger()
    const server = serverStore.findServerById(serverId)
    const name = server?.name ?? 'unknown'
    dialoger.warning(
      i18n.t('common.deleteTooltip', { type: i18n.t('serverPane.typeName'), name }),
      async () => {
        const { success, msg } = await serverStore.deleteServer(serverId)
        if (!success) {
          useMessager().error(msg || '')
        }
      },
    )
  }

  const deleteGroup = async (groupId: string) => {
    const dialoger = useDialoger()
    const group = serverStore.findServerById(groupId)
    const name = group?.name ?? 'unknown'
    const hasChildren = (group?.children?.length ?? 0) > 0

    const content = hasChildren
      ? i18n.t('serverPane.serverTree.deleteGroupWithChildrenTooltip', { name })
      : i18n.t('serverPane.serverTree.deleteGroupTooltip', { name })

    const onConfirm = async () => {
      const { success, msg } = await serverStore.deleteGroup(groupId)
      if (!success) {
        useMessager().error(msg || '')
      }
    }

    if (hasChildren) {
      dialoger.show({
        type: 'error',
        title: i18n.t('common.warning'),
        content,
        positiveText: i18n.t('common.confirm'),
        negativeText: i18n.t('common.cancel'),
        positiveButtonProps: { type: 'error' },
        onPositiveClick: onConfirm,
      })
    } else {
      dialoger.warning(content, onConfirm)
    }
  }

  const handleSelectContextMenu = (key: string) => {
    contextMenuParams.show = false
    const serverId = selectedKeys.value[0]
    if (!serverId) return

    switch (key) {
      case MenuKeys.ServerConnect:
        connectToServer(serverId).then(() => {})
        break
      case MenuKeys.ServerEdit:
        if (browserStore.isConnected(serverId)) {
          useDialoger().warning(i18n.t('serverPane.serverTree.editDisconnectConfirmation'), () => {
            browserStore.disconnect(serverId)
            dialogStore.showServerEditDialog(serverId)
          })
        } else {
          dialogStore.showServerEditDialog(serverId)
        }
        break
      case MenuKeys.ServerClone:
        dialogStore.showCloneServerDialog(serverId)
        break
      case MenuKeys.ServerDelete:
        deleteServer(serverId)
        break
      case MenuKeys.ServerDisconnect:
        browserStore.disconnect(serverId).then((closed) => {
          if (!closed) return
          useMessager().success(i18n.t('common.dialog.handleSuccess'))
        })
        break
      case MenuKeys.GroupAddServer:
        dialogStore.showNewDialog(DialogType.Server, { parentId: serverId })
        break
      case MenuKeys.GroupAddSubGroup:
        dialogStore.showNewDialog(DialogType.Group, { parentId: serverId })
        break
      case MenuKeys.GroupRename:
        dialogStore.openRenameGroupDialog(serverId)
        break
      case MenuKeys.GroupDelete:
        deleteGroup(serverId)
        break
      default:
        console.warn(`missing context menu option handling for key '${key}'`)
    }
  }

  return { contextMenuParams, openContextMenu, handleSelectContextMenu }
}
