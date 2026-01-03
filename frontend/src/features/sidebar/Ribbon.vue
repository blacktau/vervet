<script setup lang="ts">
import { type DropdownOption, useThemeVars } from 'naive-ui'
import { useRender } from '@/utils/render'
import { computed } from 'vue'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { DialogType, useDialogStore } from '@/stores/dialog'
import { useSettingsStore } from '@/features/settings/settings.ts'
import * as runtime from 'wailsjs/runtime'
import IconButton from '@/features/common/IconButton.vue'
import { extraTheme } from '@/utils/extraTheme'
import {
  BugAntIcon,
  CircleStackIcon,
  Cog8ToothIcon,
  ServerIcon,
  WrenchScrewdriverIcon,
} from '@heroicons/vue/24/outline'

import Github from '@/features/icon/Github.vue'

const themeVars = useThemeVars()
const render = useRender()

const dialogStore = useDialogStore()
const settingsStore = useSettingsStore()
const browserStore = useDataBrowserStore()

const props = withDefaults(
  defineProps<{
    value?: string
    width?: number
  }>(),
  {
    value: 'servers',
    width: 60,
  },
)

const emit = defineEmits<{
  (e: 'update:value', value: string): void
}>()

const iconSize = computed(() => Math.floor(props.width * 0.45))

const menuOptions = computed(() => {
  return [
    {
      label: 'ribbon.browser',
      key: 'browser',
      icon: CircleStackIcon,
      show: browserStore.hasOpenConnections,
    },
    {
      label: 'ribbon.servers',
      key: 'servers',
      icon: ServerIcon,
    },
  ]
})

const settingsOptions = computed(() => {
  return [
    {
      label: 'ribbon.menu.settings',
      key: 'settings',
      icon: WrenchScrewdriverIcon,
    },
    {
      label: 'ribbon.menu.reportBug',
      key: 'report',
      icon: BugAntIcon,
    },
    {
      type: 'divider',
      key: 'd1',
    },
    {
      label: 'ribbon.menu.about',
      key: 'about',
    },
  ]
})

const onSelectSettingsMenu = (key: string) => {
  switch (key) {
    case 'configuration':
      dialogStore.showNewDialog(DialogType.Settings)
      break
    case 'report':
      runtime.BrowserOpenURL('https://github.com/blacktau/vervet/issues')
      break
    case 'about':
      dialogStore.showNewDialog(DialogType.About)
      break
  }
}

const openGithub = () => {
  runtime.BrowserOpenURL('https://github.com/blacktau/vervet')
}

const exThemeVars = computed(() => {
  return extraTheme(settingsStore.isDark)
})
</script>

<template>
  <div
    id="app-ribbon"
    :style="{
      width: props.width + 'px',
      minWidth: props.width + 'px',
    }"
    class="flex-box-v">
    <div class="ribbon-wrapper flex-box-v">
      <n-tooltip
        v-for="(m, i) in menuOptions"
        :key="i"
        :delay="2"
        :show-arrow="false"
        placement="right">
        <template #trigger>
          <div
            v-show="m.show !== false"
            :class="{ 'ribbon-item-active': props.value === m.key }"
            class="ribbon-item clickable"
            @click="emit('update:value', m.key)">
            <n-icon :size="iconSize">
              <component :is="m.icon" :stroke-width="3.5" />
            </n-icon>
          </div>
        </template>
        {{ $t(m.label) }}
      </n-tooltip>
    </div>
    <div class="flex-item-expand"></div>
    <div class="nav-menu-item flex-box-v">
      <n-dropdown
        :options="settingsOptions"
        :render-icon="(option: DropdownOption) => render.renderIcon(option.icon!)"
        :render-label="
          (option: DropdownOption) =>
            render.renderLabel($t(option.label as string), { class: 'context-menu-item' })
        "
        trigger="click"
        @select="onSelectSettingsMenu">
        <icon-button :icon="Cog8ToothIcon" :size="iconSize" :stroke-width="3" />
      </n-dropdown>
      <icon-button
        :icon="Github"
        :size="iconSize"
        :tooltip-delay="100"
        t-tooltip="ribbon.github"
        @click="openGithub" />
    </div>
  </div>
</template>

<style scoped lang="scss">
#app-ribbon {
  //height: 100vh;
  border-right: v-bind('exThemeVars.splitColor') solid 1px;
  background-color: v-bind('exThemeVars.ribbonColor');
  box-sizing: border-box;
  color: v-bind('themeVars.textColor2');
  --wails-draggable: drag;

  .ribbon-wrapper {
    gap: 2px;
    margin-top: 5px;
    justify-content: center;
    align-items: center;
    box-sizing: border-box;
    padding-right: 3px;
    --wails-draggable: none;

    .ribbon-item {
      width: 100%;
      height: 100%;
      text-align: center;
      line-height: 1;
      color: v-bind('themeVars.textColor3');
      //border-left: 5px solid #000;
      border-radius: v-bind('themeVars.borderRadius');
      padding: 8px 0;
      position: relative;

      &:hover {
        background-color: rgba(0, 0, 0, 0.05);
        color: v-bind('themeVars.primaryColor');

        &:before {
          position: absolute;
          width: 3px;
          left: 0;
          top: 24%;
          bottom: 24%;
          border-radius: 9999px;
          content: '';
          background-color: v-bind('themeVars.primaryColor');
        }
      }
    }

    .ribbon-item-active {
      //background-color: v-bind('exThemeVars.ribbonActiveColor');
      color: v-bind('themeVars.primaryColor');

      &:hover {
        color: v-bind('themeVars.primaryColor') !important;
      }

      &:before {
        position: absolute;
        width: 3px;
        left: 0;
        top: 24%;
        bottom: 24%;
        border-radius: 9999px;
        content: '';
        background-color: v-bind('themeVars.primaryColor');
      }
    }
  }

  .nav-menu-item {
    align-items: center;
    padding: 10px 0 15px;
    --wails-draggable: none;

    button {
      margin: 10px 0;
    }
  }
}
</style>
