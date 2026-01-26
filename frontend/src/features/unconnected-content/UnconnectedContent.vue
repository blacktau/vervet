<script lang="ts" setup>
import { useThemeVars } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { PlusIcon } from '@heroicons/vue/24/outline'

const dialogStore = useDialogStore()
const themeVars = useThemeVars()
</script>

<template>
  <div class="content-container flex-box-v">
    <n-empty :description="$t('unconnected.emptyContent')">
      <template #extra>
        <n-button :focusable="false" @click="dialogStore.showNewServerDialog()">
          <template #icon>
            <n-icon :component="PlusIcon" size="18" />
          </template>
          {{ $t('unconnected.newConnection') }}
        </n-button>
      </template>
    </n-empty>
  </div>
</template>

<style lang="scss" scoped>
@use '@/css/content';

.content-container {
  justify-content: center;
  padding: 5px;
  box-sizing: border-box;
  position: relative;
}

.color-preset-item {
  width: 24px;
  height: 24px;
  margin-right: 2px;
  border: white 3px solid;
  cursor: pointer;

  &_selected,
  &:hover {
    border-color: #cdd0d6;
  }
}
</style>
