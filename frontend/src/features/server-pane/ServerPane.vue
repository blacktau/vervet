<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useRender } from '@/utils/render.ts'
import { ref } from 'vue'
import IconButton from '@/features/common/IconButton.vue'
import ServerTree from '@/features/server-pane/ServerTree.vue'
import {
  ArrowRightEndOnRectangleIcon,
  ArrowRightStartOnRectangleIcon,
  EllipsisVerticalIcon,
  FunnelIcon,
  PlusIcon,
  FolderPlusIcon
} from '@heroicons/vue/24/outline'

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const serverStore = useServerStore()
const render = useRender()
const filterPattern = ref('')

const moreOptions = [
  {
    key: 'import',
    label: 'interface.serverPane.importServers',
    icon: ArrowRightEndOnRectangleIcon,
  },
  {
    key: 'export',
    label: 'interface.serverPane.exportServers',
    icon: ArrowRightStartOnRectangleIcon,
  },
]

const onSelectOptions = async (select: string) => {
  switch (select) {
    case 'import':
      await serverStore.importServers()
      await serverStore.refreshServers(true)
      break
    case 'export':
      await serverStore.exportServers()
      break
  }
}
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <server-tree :filter-pattern="filterPattern" />
    <div class="nav-pane-bottom nav-pane-func flex-box-h">
      <icon-button
        :button-class="['nav-pane-func-btn']"
        :icon="PlusIcon"
        :stroke-width="3.5"
        size="20"
        t-tooltip="interface.serverPane.addServer"
        @click="dialogStore.showNewServerDialog()" />
      <icon-button
        :button-class="['nav-pane-func-btn']"
        :icon="FolderPlusIcon"
        :stroke-width="3.5"
        size="20"
        t-tooltip="interface.serverPane.addGroup"
        @click="dialogStore.openNewGroupDialog()" />
      <n-divider vertical />
      <n-input
        v-model:value="filterPattern"
        :autofocus="false"
        :placeholder="$t('interface.serverPane.filter')"
        clearable>
        <template #prefix>
          <n-icon :component="FunnelIcon" size="20" />
        </template>
      </n-input>
      <n-dropdown
        :options="moreOptions"
        :render-icon="(option) => render.renderIcon(option.icon, { strokeWidth: 3.5 })"
        :render-label="(option) => $t(option.label)"
        placement="top-end"
        style="min-width: 130px"
        trigger="click"
        @select="onSelectOptions">
        <icon-button
          :button-class="['nav-pane-func-btn']"
          :icon="EllipsisVerticalIcon"
          :stroke-width="3.5"
          size="20" />
      </n-dropdown>
    </div>
  </div>
</template>

<style scoped lang="scss">
.nav-pane-bottom {
  color: v-bind('themeVars.iconColor');
  border-top: v-bind('themeVars.borderColor') 1px solid;
}
</style>
