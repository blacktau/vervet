<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { ref } from 'vue'
import IconButton from '@/features/common/IconButton.vue'
import ServerTree from '@/features/server-pane/ServerTree.vue'
import {
  FunnelIcon,
  PlusIcon,
  FolderPlusIcon
} from '@heroicons/vue/24/outline'

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const filterPattern = ref('')
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
        t-tooltip="serverPane.addServer"
        @click="dialogStore.showNewServerDialog()" />
      <icon-button
        :button-class="['nav-pane-func-btn']"
        :icon="FolderPlusIcon"
        :stroke-width="3.5"
        size="20"
        t-tooltip="serverPane.addGroup"
        @click="dialogStore.openNewGroupDialog()" />
      <n-divider vertical />
      <n-input
        v-model:value="filterPattern"
        :autofocus="false"
        :placeholder="$t('serverPane.filter')"
        clearable>
        <template #prefix>
          <n-icon :component="FunnelIcon" size="20" />
        </template>
      </n-input>
    </div>
  </div>
</template>

<style scoped lang="scss">
.nav-pane-bottom {
  color: v-bind('themeVars.iconColor');
  border-top: v-bind('themeVars.borderColor') 1px solid;
}
</style>
