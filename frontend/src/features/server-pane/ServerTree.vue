<script lang="ts" setup>
import { type DropdownOption, NIcon, NSpace, NText, useThemeVars } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRender } from '@/utils/render.ts'
import { h, nextTick, reactive, ref, type VNodeArrayChildren } from 'vue'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/stores/tabs.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useDialogStore } from '@/stores/dialog.ts'
import { includes, indexOf, isEmpty } from 'lodash'
import { useDialoger, useMessager } from '@/utils/dialog.ts'
import PlugConnected from '@/features/icon/PlugConnected.vue'
import IconButton from '@/features/common/IconButton.vue'

import {
  Cog8ToothIcon,
  DocumentDuplicateIcon,
  FolderIcon,
  FolderOpenIcon,
  PencilSquareIcon,
  ServerIcon,
  ServerStackIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

import PlugDisconnected from '@/features/icon/PlugDisconnected.vue'
import SrvIcon from '@/features/icon/SrvIcon.vue'
import { getServerColour } from '@/features/server-pane/helpers.ts'
import { ServerNodeType, type ServerTreeNode } from '@/features/server-pane/types.ts'

const themeVars = useThemeVars()
const i18n = useI18n()
const render = useRender()
const connectingServer = ref('')

const browserStore = useDataBrowserStore()
const tabStore = useTabStore()
const settingsStore = useSettingsStore()
const dialogStore = useDialogStore()
const serverStore = useServerStore()

const props = defineProps<{
  filterPattern?: string
}>()

const contextMenuParams = reactive<{
  show: boolean
  x: number
  y: number
  options?: unknown
  currentNode?: unknown
}>({
  show: false,
  x: 0,
  y: 0,
  options: undefined,
  currentNode: undefined,
})

const MenuKeys = {
  GroupRename: 'group_rename',
  GroupDelete: 'group_delete',
  ServerDisconnect: 'server_disconnect',
  ServerEdit: 'server_edit',
  ServerClone: 'server_clone',
  ServerConnect: 'server_connect',
  ServerDelete: 'server_delete',
}

const menuOptions = {
  [ServerNodeType.Group]: () => [
    {
      key: MenuKeys.GroupRename,
      label: 'serverPane.serverTree.renameGroup',
      icon: PencilSquareIcon,
    },
    {
      key: MenuKeys.GroupDelete,
      label: 'serverPane.serverTree.deleteGroup',
      icon: TrashIcon,
    },
  ],
  [ServerNodeType.Server]: ({ serverId }: { serverId: string }) => {
    const common = [
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
      {
        type: 'divider',
        key: 'd1',
      },
      {
        key: MenuKeys.ServerDelete,
        label: 'serverPane.serverTree.deleteServer',
        icon: TrashIcon,
      },
    ]

    const connected = browserStore.isConnected(serverId)
    if (connected) {
      return [
        {
          key: MenuKeys.ServerDisconnect,
          label: 'serverPane.serverTree.disconnect',
          icon: PlugDisconnected,
        },
        ...common,
      ]
    } else {
      return [
        {
          key: MenuKeys.ServerConnect,
          label: 'serverPane.serverTree.connectServer',
          icon: PlugConnected,
        },
        ...common,
      ]
    }
  },
}

const expandedKeys = ref<string[]>([])
const selectedKeys = ref<string[]>([])

const renderLabel = (x: { option: ServerTreeNode }) => {
  const option = x.option
  if (option.type == ServerNodeType.Server) {
    const colour = getServerColour(option)
    if (colour) {
      return h(NText, { style: { color: colour, fontWeight: '450' } }, () => option.label)
    }
  }
  return h(NText, {}, () => option.label)
}

const renderIconMenu = (items: VNodeArrayChildren) => {
  return h(
    NSpace,
    {
      align: 'center',
      inline: true,
      size: 3,
      wrapItem: false,
      wrap: false,
      style: 'margin-right: 5px',
    },
    () => items,
  )
}

const getServerNodeIcon = (server: ServerTreeNode) => {
  if (server.is) {
    return ServerStackIcon
  } else if (server.isSrv) {
    return SrvIcon
  }

  return ServerIcon
}

const renderPrefix = ({ option }: { option: ServerTreeNode }) => {
  const iconTransparency = settingsStore.isDark ? 0.75 : 1
  if (option.type == ServerNodeType.Group) {
    const opened = indexOf(expandedKeys.value, option.key) >= 0
    const icon = opened ? FolderOpenIcon : FolderIcon
    return h(
      NIcon,
      { size: 20 },
      {
        default: () =>
          h(icon, {
            open: opened,
            fillColor: `rgba(56, 176, 0, ${iconTransparency})`,
          }),
      },
    )
  } else {
    const connected = browserStore.isConnected(option.key as string)
    const icon = getServerNodeIcon(option)
    const iconColour = connected ? '#38b000' : 'currentColor'

    return h(
      NIcon,
      { size: 20, color: iconColour },
      {
        default: () =>
          h(icon, {
            inverse: false, //connected,
            filColor: `rgba(56, 176, 0. ${iconTransparency})`,
          }),
      },
    )
  }
}

const renderSuffix = ({ option }: { option: ServerTreeNode }) => {
  if (!includes(selectedKeys.value, option.key)) {
    return undefined
  }

  if (option.isGroup) {
    return renderIconMenu(getGroupMenu())
  } else {
    const connected = browserStore.isConnected(option.key as string)
    return renderIconMenu(getServerMenu(connected))
  }
}

const getServerMenu = (connected: boolean) => {
  const btns = []
  if (connected) {
    btns.push(
      h(IconButton, {
        tTooltip: 'serverPane.serverTree.disconnect',
        icon: PlugDisconnected,
        onClick: () => handleSelectContextMenu(MenuKeys.ServerDisconnect),
      }),
      h(IconButton, {
        tTooltip: 'serverPane.serverTree.editServer',
        icon: Cog8ToothIcon,
        onClick: () => handleSelectContextMenu(MenuKeys.ServerEdit),
      }),
    )
  } else {
    btns.push(
      h(IconButton, {
        tTooltip: 'serverPane.serverTree.connectServer',
        icon: PlugConnected,
        onClick: () => handleSelectContextMenu(MenuKeys.ServerConnect),
      }),
      h(IconButton, {
        tTooltip: 'serverPane.serverTree.editServer',
        icon: Cog8ToothIcon,
        onClick: () => handleSelectContextMenu(MenuKeys.ServerEdit),
      }),
      h(IconButton, {
        tTooltip: 'serverPane.serverTree.deleteServer',
        icon: TrashIcon,
        onClick: () => handleSelectContextMenu(MenuKeys.ServerDelete),
      }),
    )
  }
  return btns
}

const getGroupMenu = () => {
  return [
    h(IconButton, {
      tTooltip: 'serverPane.serverTree.groupRename',
      icon: Cog8ToothIcon,
      onClick: () => handleSelectContextMenu(MenuKeys.GroupRename),
    }),
    h(IconButton, {
      tTooltip: 'serverPane.serverTree.groupDelete',
      icon: TrashIcon,
      onClick: () => handleSelectContextMenu(MenuKeys.GroupDelete),
    }),
  ]
}

const nodeProps = ({ option }: { option: ServerTreeNode }) => {
  return {
    onDblclick: async () => {
      if (option.isGroup) {
        nextTick().then(() => expandKey(option.key as string))
      } else {
        connectToServer(option.key as string).then(() => {})
      }
    },
    onContextmenu(e: Event) {
      e.preventDefault()
      const mop = menuOptions[option.type]
      if (!mop) {
        return
      }
    },
  }
}

const connectToServer = async (serverId: string) => {
  try {
    connectingServer.value = serverId
    if (!browserStore.isConnected(serverId)) {
      await browserStore.connect(serverId)
    }

    if (!isEmpty(connectingServer.value)) {
      tabStore.upsertTab({
        server: serverId,
        forceSwitch: true,
      })
    }
  } catch (e) {
    const messager = useMessager()
    const err = e as Error
    messager.error(err.message)
  } finally {
    connectingServer.value = ''
  }
}

const onUpdateExpandedKeys = (keys: string[]) => {
  expandedKeys.value = keys
}
const onUpdateSelectedKeys = (keys: string[]) => {
  selectedKeys.value = keys
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
        const messager = useMessager()
        messager.error(msg || '')
      }
    },
  )
}

const deleteGroup = async (serverId: string) => {
  const dialoger = useDialoger()
  const server = serverStore.findServerById(serverId)
  const name = server?.name ?? 'unknown'
  dialoger.warning(i18n.t('serverPane.serverTree.deleteGroupTooltip', { name }), async () => {
    const { success, msg } = await serverStore.deleteGroup(serverId)
    if (!success) {
      const messager = useMessager()
      messager.error(msg || '')
    }
  })
}

const expandKey = (key: string) => {
  const idx = expandedKeys.value.indexOf(key)
  if (idx === -1) {
    expandedKeys.value.push(key)
  } else {
    expandedKeys.value.splice(idx, 1)
  }
}

const handleSelectContextMenu = (key: string) => {
  contextMenuParams.show = false
  console.log('handleSelectContextMenu: ', key, selectedKeys.value.join(';'))
  const selectedKey = selectedKeys.value.length > 0 ? selectedKeys.value[0] : undefined
  if (!selectedKey) {
    return
  }

  const serverId = selectedKey

  switch (key) {
    case MenuKeys.ServerConnect:
      connectToServer(serverId).then(() => {})
      break
    case MenuKeys.ServerEdit:
      if (browserStore.isConnected(serverId)) {
        const dialoger = useDialoger()
        dialoger.warning(i18n.t('serverPane.serverTree.editDisconnectConfirmation'), () => {
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
        if (!closed) {
          return
        }

        const messager = useMessager()
        messager.success(i18n.t('common.dialog.handleSuccess'))
      })
      break
    case MenuKeys.GroupRename:
      if (selectedKey.length == 0) {
        return
      }

      dialogStore.openRenameGroupDialog(selectedKey)
      break
    case MenuKeys.GroupDelete:
      if (selectedKey.length == 0) {
        return
      }

      deleteGroup(selectedKey)
      break
    default:
      console.warn(`missing context menu option handling for key '${key}'`)
  }
}

const onCancelConnecting = async () => {
  if (connectingServer.value === '') {
    return
  }

  await browserStore.disconnect(connectingServer.value)
  connectingServer.value = ''
}
</script>

<template>
  <div class="server-tree-wrapper" @keydown.esc="contextMenuParams.show = false">
    <n-tree
      :animated="false"
      :block-line="true"
      :block-node="true"
      :cancelable="false"
      :data="serverStore.mappedTree"
      :draggable="true"
      :expanded-keys="expandedKeys"
      :node-props="nodeProps"
      :pattern="props.filterPattern"
      :render-label="renderLabel"
      :render-prefix="renderPrefix"
      :render-suffix="renderSuffix"
      :selected-keys="selectedKeys"
      class="fill-height"
      virtual-scroll
      @update:expanded-keys="onUpdateExpandedKeys"
      @update:selected-keys="onUpdateSelectedKeys">
      <template #empty>
        <n-empty :description="$t('serverPane.serverTree.empty')" class="empty-content" />
      </template>
    </n-tree>

    <n-modal :show="connectingServer !== ''" transform-origin="center">
      <n-card
        :bordered="false"
        :content-style="{ textAlign: 'center' }"
        aria-modal="true"
        role="dialog"
        style="width: 400px">
        <n-spin>
          <template #description>
            <n-space vertical>
              <n-text strong>{{ $t('common.dialog.connecting') }}}</n-text>
              <n-button :focusable="false" secondary size="small" @click="onCancelConnecting">
                {{ $t('common.dialog.cancelConnecting') }}
              </n-button>
            </n-space>
          </template>
        </n-spin>
      </n-card>
    </n-modal>

    <n-dropdown
      :keyboard="true"
      :options="contextMenuParams.options"
      :render-icon="({ icon }: DropdownOption) => render.renderIcon(icon)"
      :render-label="
        ({ label }: DropdownOption) =>
          render.renderLabel($t(label as string), { class: 'context-menu-item' })
      "
      :show="contextMenuParams.show"
      :x="contextMenuParams.x"
      :y="contextMenuParams.y"
      placement="bottom-start"
      trigger="manual"
      @clickoutside="contextMenuParams.show = false"
      @select="handleSelectContextMenu" />
  </div>
</template>

<style lang="scss" scoped>
@use '@/css/content';

.server-tree-wrapper {
  height: 100%;
  overflow: hidden;
}
</style>
