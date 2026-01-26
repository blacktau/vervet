<script lang="ts" setup>
import * as runtime from 'wailsjs/runtime'
import { WindowToggleMaximise } from 'wailsjs/runtime'
import iconUrl from '@/assets/logo.svg'
import { NavType, useTabStore } from '@/features/tabs/tabs.ts'
import { isMacOS } from '@/init/environment.ts'
import ConnectedServerTabs from '@/features/tabs/ConnectedServerTabs.vue'
import ToolbarControlWidget from '@/features/common/ToolbarControlWidget.vue'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { extraTheme } from '@/utils/extraTheme.ts'

const tabStore = useTabStore()
const settingsStore = useSettingsStore()

const props = defineProps<{
  navMenuWidth: number
  toolbarHeight: number
}>()

const logoWrapperWidth = computed(() => {
  return `${props.navMenuWidth + settingsStore.window.asideWidth - 4}px`
})

const logoPaddingLeft = ref<number>(10)
const maximised = ref<boolean>(false)

const exThemeVars = computed(() => {
  return extraTheme(settingsStore.isDark)
})

const onToggleFullscreen = (fullscreen: boolean) => {
  if (fullscreen) {
    logoPaddingLeft.value = 10
  } else {
    logoPaddingLeft.value = isMacOS() ? 70 : 10
  }
}

const onToggleMaximize = (isMaximized: boolean) => {
  maximised.value = isMaximized
}

runtime.EventsOn('window_changed', (info) => {
  const { fullscreen, maximized } = info
  onToggleFullscreen(fullscreen == true)
  onToggleMaximize(maximized)
})

onMounted(async () => {
  const fullscreen = await runtime.WindowIsFullscreen()
  onToggleFullscreen(fullscreen)
  const maximized = await runtime.WindowIsMinimised()
  onToggleMaximize(maximized)
  window.addEventListener('keydown', onKeyShortCut)
})

const onKeyShortCut = (e: KeyboardEvent) => {
  const isCtrlOn = isMacOS() ? e.metaKey : e.ctrlKey
  switch (e.key) {
    case 'w':
      if (isCtrlOn) {
        const tabStore = useTabStore()
        const currentTab = tabStore.currentTab
        if (!!currentTab) {
          tabStore.closeTab(currentTab.serverId)
        }
      }
      break
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyShortCut)
})
</script>

<template>
  <div
    id="app-toolbar"
    :style="{ height: props.toolbarHeight + 'px' }"
    class="flex-box-h"
    style="--wails-draggable: drag"
    @dblclick="WindowToggleMaximise">
    <div
      id="app-toolbar-title"
      :style="{
        width: `${logoWrapperWidth}px`,
        minWidth: `${logoWrapperWidth}px`,
        paddingLeft: `${logoPaddingLeft}px`,
      }">
      <n-space :size="3" :wrap="false" :wrap-item="false" align="center">
        <n-avatar :size="32" :src="iconUrl" color="#0000" style="min-width: 32px" />
        <div style="min-width: 68px; white-space: nowrap; font-weight: 800">Vervet</div>
      </n-space>
    </div>
    <div v-show="tabStore.nav === NavType.Browser" class="app-toolbar-tab flex-item-expand">
      <connected-server-tabs />
    </div>
    <div class="flex-item-expand" style="min-width: 15px"></div>
    <toolbar-control-widget
      v-if="!isMacOS()"
      :maximised="maximised"
      :size="props.toolbarHeight"
      style="align-self: flex-start" />
  </div>
</template>

<style lang="scss" scoped>
#app-toolbar {
  background-color: v-bind('exThemeVars.titleColor');
  border-bottom: 1px solid v-bind('exThemeVars.splitColor');

  &-title {
    padding-left: 10px;
    padding-right: 10px;
    box-sizing: border-box;
    align-self: center;
    align-items: baseline;
  }
}

.app-toolbar-tab {
  align-self: flex-end;
  margin-bottom: -1px;
  margin-left: 3px;
  overflow: auto;
}
</style>
