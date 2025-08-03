<script lang="ts" setup>
import { type DropdownOption, NIcon, NSpace, NText, useThemeVars } from 'naive-ui'
import { useRender } from '@/utils/render.ts'
import { h, nextTick, ref, type VNodeArrayChildren, watch } from 'vue'
import { type RegisteredServerNode, useServerStore } from '@/features/server-pane/serverStore.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { includes, indexOf } from 'lodash'
import { useServerConnection } from '@/features/server-pane/useServerConnection.ts'
import {
  MenuKeys,
  useServerTreeContextMenu,
} from '@/features/server-pane/useServerTreeContextMenu.ts'
import PlugConnected from '@/features/icon/PlugConnected.vue'
import IconButton from '@/features/common/IconButton.vue'
import PlugDisconnected from '@/features/icon/PlugDisconnected.vue'
import SrvIcon from '@/features/icon/SrvIcon.vue'

import {
  Cog8ToothIcon,
  FolderIcon,
  FolderOpenIcon,
  ServerIcon,
  ServerStackIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

const props = defineProps<{
  filterPattern?: string
}>()

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const themeVars = useThemeVars()
const render = useRender()

const browserStore = useDataBrowserStore()
const settingsStore = useSettingsStore()
const serverStore = useServerStore()

const { connectingServer, connectToServer, onCancelConnecting } = useServerConnection()

const expandedKeys = ref<string[]>([])
const selectedKeys = ref<string[]>([])

const { contextMenuParams, openContextMenu, handleSelectContextMenu } = useServerTreeContextMenu(
  selectedKeys,
  connectToServer,
)

watch(
  () => contextMenuParams.show,
  (val) => console.log('[ServerTree] contextMenuParams.show changed:', val),
)

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
            fillColor: `rgba(56, 176, 0, ${iconTransparency})`,
          }),
      },
    )
  }
}

const renderSuffix = ({ option }: { option: RegisteredServerNode }) => {
  const items: VNodeArrayChildren = []

  if (includes(selectedKeys.value, option.id)) {
    if (option.isGroup) {
      items.push(...getGroupMenu())
    } else {
      const connected = browserStore.isConnected(option.id)
      items.push(...getServerMenu(connected))
    }
  }

  items.push(
    h('span', {
      style: {
        display: 'inline-block',
        width: '10px',
        height: '10px',
        marginLeft: '3px',
        borderRadius: '50%',
        backgroundColor: option.colour || 'transparent',
        flexShrink: 0,
      },
    }),
  )

  return renderIconMenu(items)
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

const nodeProps = ({ option }: { option: RegisteredServerNode }) => {
  return {
    onDblclick: async () => {
      if (option.isGroup) {
        nextTick().then(() => expandKey(option.id))
      } else {
        connectToServer(option.id).then(() => {})
      }
    },
    onContextmenu(e: PointerEvent) {
      console.log('[ServerTree-Node] onContextmenu', e)
      openContextMenu(option, e)
    },
  }
}

const onUpdateExpandedKeys = (keys: string[]) => {
  expandedKeys.value = keys
}
const onUpdateSelectedKeys = (keys: string[]) => {
  selectedKeys.value = keys
}

const expandKey = (key: string) => {
  const idx = expandedKeys.value.indexOf(key)
  if (idx === -1) {
    expandedKeys.value.push(key)
  } else {
    expandedKeys.value.splice(idx, 1)
  }
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
              <n-text strong>{{ $t('common.dialog.connecting') }}</n-text>
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
