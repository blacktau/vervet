<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { computed, onMounted, reactive, ref, watchEffect } from 'vue'
import { useTabStore } from '@/stores/tabs'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { extraTheme } from '@/utils/extraTheme'
import { debounce } from 'lodash'
import { isMacOS, isWindows } from '@/init/environment'
import * as runtime from 'wailsjs/runtime'
import { WindowToggleMaximise } from 'wailsjs/runtime'
import iconUrl from '@/assets/logo.svg'

import ToolbarControlWidget from '@/features/common/ToolbarControlWidget.vue'
import Ribbon from '@/features/sidebar/Ribbon.vue'
import ResizeableWrapper from '@/features/common/ResizeableWrapper.vue'
import ContentPane from '@/features/content/ContentPane.vue'
import ContentLogPane from '@/features/content/ContentLogPane.vue'
import ServerPane from '@/features/server-pane/ServerPane.vue'
import DataBrowserPane from '@/features/data-browser/DataBrowserPane.vue'

const themeVars = useThemeVars()
const props = defineProps<{
  loading: boolean
}>()

const data = reactive({
  navMenuWidth: 50,
  toolbarHeight: 38,
})

const tabStore = useTabStore()
const connectionsStore = useDataBrowserStore()
const settingsStore = useSettingsStore()
const logPaneRef = ref()
const exThemeVars = computed(() => {
  return extraTheme(settingsStore.isDark)
})
const saveSidebarWidth = debounce(settingsStore.saveConfiguration, 1000, {trailing: true})
const handleResize = () => {
  saveSidebarWidth()
}

watchEffect(() => {
  if (tabStore.nav === 'log') {
    logPaneRef.value?.refresh()
  }
})

const logoWrapperWidth = computed(() => {
  return `${data.navMenuWidth + settingsStore.window.asideWidth - 4 }px`
})

const logoPaddingLeft = ref<number>(10)
const maximised = ref<boolean>(false)
const hideRadius = ref<boolean>(false)

const wrapperStyle = computed(() => {
  if (isWindows()) {
    return {}
  }

  if (hideRadius.value) {
    return {}
  }

  return {
    border: `1px solid ${themeVars.value.borderColor}`,
    borderRadius: '10px'
  }
})

const spinStyle = computed<CSSStyleValue>(() => {
  if (isWindows() || hideRadius.value) {
    return {
      backgroundColor: themeVars.value.bodyColor
    }
  }

  return {
    backgroundColor: themeVars.value.bodyColor,
    borderRadius: '10px'
  }
})

const onToggleFullscreen = (fullscreen: boolean) => {
  hideRadius.value = fullscreen
  if (fullscreen) {
    logoPaddingLeft.value = 10
  } else {
    logoPaddingLeft.value = isMacOS() ? 70 : 10
  }
}

const onToggleMaximize = (isMaximized: boolean) => {
  if (isMaximized) {
    maximised.value = true
    if (!isMacOS()) {
      hideRadius.value = true
    }
  } else {
    maximised.value = false
    if (!isMacOS()) {
      hideRadius.value = false
    }
  }
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
          tabStore.closeTab(currentTab.name)
        }
      }
      break
  }
}
</script>

<template>
  <n-spin :show="props.loading" :style="spinStyle" :theme-overrides="{ opacitySpinning: 0 }">
    <div id="app-content-wrapper" :style="wrapperStyle" class="flex-box-v">
      <div
        id="app-toolbar"
        :style="{ height: data.toolbarHeight + 'px' }"
        class="flex-box-h"
        style="--wails-draggable: drag"
        @dblclick="WindowToggleMaximise">
        <div
          id="app-toolbar-title"
          :style="{
            width: logoWrapperWidth,
            minWidth: logoWrapperWidth,
            paddingLeft: `${logoPaddingLeft}px`,
          }">
          <n-space :size="3" :wrap="false" :wrap-item="false" align="center">
            <n-avatar :size="32" :src="iconUrl" color="#0000" style="min-width: 32px" />
            <div style="min-width: 68px; white-space: nowrap; font-weight: 800">Vervet</div>
            <transition name="fade">
              <n-text
                v-if="tabStore.nav === 'browser'"
                class="ellipsis"
                strong
                style="font-size: 13px">
                - {{ tabStore.currentTab?.name }}
              </n-text>
            </transition>
          </n-space>
        </div>
        <div v-show="tabStore.nav !== 'browser'" class="app-toolbar-tab flex-item-expand">
          <div>content-value-tab</div>
        </div>
        <div class="flex-item-expand" style="min-width: 15px"></div>
        <toolbar-control-widget
          v-if="!isMacOS()"
          :maximised="maximised"
          :size="data.toolbarHeight"
          style="align-self: flex-start" />
      </div>
      <div
        id="app-content"
        :style="settingsStore.uiFont"
        class="flex-box-h flex-item-expand"
        style="--wails-draggable: none">
        <ribbon v-model:value="tabStore.nav" :width="data.navMenuWidth" />
        <div v-show="tabStore.nav === 'browser'" class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper
            :min-size="300"
            :offset="data.navMenuWidth"
            class="flex-item"
            @update:size="handleResize">
            <data-browser-pane
              v-show="connectionsStore.hasOpenConnections"
              class="app-side flex-item-expand" />
            />
          </resizeable-wrapper>
          <content-pane
            v-for="t in tabStore.tabs"
            v-show="t.name === tabStore.currentTab?.name"
            :key="t.name"
            class="flex-item-expand" />
        </div>
        <div v-show="tabStore.nav === 'servers'" class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper
            v-model:size="settingsStore.window.asideWidth"
            :min-size="300"
            :offset="data.navMenuWidth"
            class="flex-item"
            @update:size="handleResize">
            <server-pane class="app-side flex-item-expand" />
          </resizeable-wrapper>
<!--          <content-server-pane class="flex-item-expand" />-->
        </div>
        <div v-show="tabStore.nav === 'log'" class="content-area flex-box-h flex-item-expand">
          <content-log-pane ref="logPaneRef" class="flex-item-expand" />
        </div>
      </div>
    </div>
  </n-spin>
</template>

<style scoped lang="scss">
#app-content-wrapper {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background-color: v-bind('themeVars.bodyColor');
  color: v-bind('themeVars.textColorBase');

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

  #app-content {
    height: calc(100% - 60px);

    .content-area {
      overflow: hidden;
    }
  }

  .app-side {
    //overflow: hidden;
    height: 100%;
    background-color: v-bind('exThemeVars.sidebarColor');
    border-right: 1px solid v-bind('exThemeVars.splitColor');
  }
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}
</style>
