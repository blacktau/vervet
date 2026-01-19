<script lang="ts" setup>
import {
  type DropdownOption,
  type MenuDividerOption,
  NIcon,
  NSpace,
  NText,
  useThemeVars,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRender } from '@/utils/render.ts'
import {
  computed,
  h,
  type HTMLAttributes,
  markRaw,
  nextTick,
  reactive,
  ref,
  type VNodeArrayChildren,
} from 'vue'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useDialogStore } from '@/stores/dialog.ts'
import { includes, indexOf, isEmpty } from 'lodash'
import { useDialoger, useMessager } from '@/utils/dialog.ts'
import PlugConnected from '@/features/icon/PlugConnected.vue'
import IconButton from '@/features/common/IconButton.vue'
import PlugDisconnected from '@/features/icon/PlugDisconnected.vue'
import SrvIcon from '@/features/icon/SrvIcon.vue'
import { getServerColour } from '@/features/server-pane/helpers.ts'
import { ServerNodeType } from '@/features/server-pane/types.ts'

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
import type { MenuRenderOption } from 'naive-ui/es/menu/src/interface'

const props = defineProps<{
  filterPattern?: string
}>()

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const themeVars = useThemeVars()
const i18n = useI18n()
const render = useRender()

const browserStore = useDataBrowserStore()
const tabStore = useTabStore()
const settingsStore = useSettingsStore()
const dialogStore = useDialogStore()
const serverStore = useServerStore()

const connectingServer = ref('')
const expandedKeys = ref<string[]>([])
const selectedKeys = ref<string[]>([])

const contextMenuParams = reactive<{
  show: boolean
  x: number
  y: number
  options: Array<MenuRenderOption | MenuDividerOption>
  currentNode?: RegisteredServerNode
}>({
  show: false,
  x: 0,
  y: 0,
  options: [],
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

const menuOptions: Record<
  ServerNodeType,
  (option: RegisteredServerNode) => Array<MenuRenderOption | MenuDividerOption>
> = {
  [ServerNodeType.Group]: () => [
    {
      key: MenuKeys.GroupRename,
      label: 'serverPane.serverTree.renameGroup',
      icon: PencilSquareIcon,
      type: 'render',
    } as MenuRenderOption,
    {
      key: MenuKeys.GroupDelete,
      label: 'serverPane.serverTree.deleteGroup',
      icon: TrashIcon,
      type: 'render',
    } as MenuRenderOption,
  ],
  [ServerNodeType.Server]: (option: RegisteredServerNode) => {
    const serverId = option.id
    const common = [
      {
        key: MenuKeys.ServerEdit,
        label: 'serverPane.serverTree.editServer',
        icon: Cog8ToothIcon,
        type: 'render',
      } as MenuRenderOption,
      {
        key: MenuKeys.ServerClone,
        label: 'serverPane.serverTree.cloneServer',
        icon: DocumentDuplicateIcon,
        type: 'render',
      } as MenuRenderOption,
      {
        type: 'divider',
        key: 'd1',
      } as MenuDividerOption,
      {
        key: MenuKeys.ServerDelete,
        label: 'serverPane.serverTree.deleteServer',
        icon: TrashIcon,
        type: 'render',
      } as MenuRenderOption,
    ]

    const connected = browserStore.isConnected(serverId)
    if (connected) {
      return [
        ...common,
        {
          key: MenuKeys.ServerDisconnect,
          label: 'serverPane.serverTree.disconnect',
          icon: PlugDisconnected,
          type: 'render',
        } as MenuRenderOption,
      ]
    }

    return [
      ...common,
      {
        key: MenuKeys.ServerConnect,
        label: 'serverPane.serverTree.connectServer',
        icon: PlugConnected,
        type: 'render',
      } as MenuRenderOption,
    ]
  },
}

const renderLabel = (x: { option: RegisteredServerNode }) => {
  return h(NText, {}, () => x.option.name)
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

const getServerNodeIcon = (server: RegisteredServerNode) => {
  if (server.isCluster) {
    return ServerStackIcon
  } else if (server.isSrv) {
    return SrvIcon
  }

  return ServerIcon
}

const renderPrefix = ({ option }: { option: RegisteredServerNode }) => {
  const iconTransparency = settingsStore.isDark ? 0.75 : 1
  if (option.isGroup) {
    const opened = indexOf(expandedKeys.value, option.id) >= 0
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
    const connected = browserStore.isConnected(option.id)
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

const renderSuffix = ({ option }: { option: RegisteredServerNode }) => {
  if (!includes(selectedKeys.value, option.id)) {
    return undefined
  }

  if (option.isGroup) {
    return renderIconMenu(getGroupMenu())
  } else {
    const connected = browserStore.isConnected(option.id)
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

const colorCalc = (node: RegisteredServerNode) => {
  if (node?.id == null) {
    return undefined
  }

  const isSelected = selectedKeys.value.indexOf(node.id) > -1

  return getServerColour(node, isSelected, settingsStore.isDark)
}

const nodeProps = computed(() => (x: { option: RegisteredServerNode }) => {
  const option = x.option

  return {
    style: {
      backgroundColor: colorCalc(option),
    },
    onDblclick: async () => {
      if (option.isGroup) {
        nextTick().then(() => expandKey(option.id))
      } else {
        connectToServer(option.id).then(() => {})
      }
    },
    onContextmenu(e: PointerEvent) {
      console.log('onContextMenu', e)
      e.preventDefault()
      const type = option.isGroup ? ServerNodeType.Group : ServerNodeType.Server
      const mop = menuOptions[type]
      if (mop == null) {
        return
      }
      contextMenuParams.show = false
      nextTick().then(() => {
        console.log('tick...')
        contextMenuParams.options = markRaw(mop(option)) as never
        contextMenuParams.currentNode = option
        contextMenuParams.x = e.clientX
        contextMenuParams.y = e.clientY
        contextMenuParams.show = true
        selectedKeys.value = [option.id]
      })
    },
  } as HTMLAttributes
})

const connectToServer = async (serverId: string) => {
  try {
    connectingServer.value = serverId
    const connectionResult = await browserStore.connect(serverId)
    if (!connectionResult.success) {
      return
    }

    if (!isEmpty(connectingServer.value)) {
      tabStore.upsertTab({
        serverId: serverId,
        title: connectionResult.name || '',
        forceSwitch: true,
        blank: false,
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
      :data="serverStore.serverTree"
      :draggable="true"
      :expanded-keys="expandedKeys"
      :node-props="nodeProps"
      :pattern="props.filterPattern"
      :render-label="renderLabel"
      :render-prefix="renderPrefix"
      :render-suffix="renderSuffix"
      :selected-keys="selectedKeys"
      class="fill-height"
      key-field="id"
      label-field="name"
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
      @clickoutside="() => (contextMenuParams.show = false)"
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
