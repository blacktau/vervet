<script lang="ts" setup>
import { NButton, useThemeVars } from 'naive-ui'
import { computed, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { NavType, useTabStore } from '@/features/tabs/tabs.ts'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { extraTheme } from '@/utils/extraTheme'
import { debounce } from 'lodash'
import { isMacOS, isWindows } from '@/init/environment'
import { useI18n } from 'vue-i18n'
import * as runtime from 'wailsjs/runtime'
import LeftRibbon from '@/features/sidebar/LeftRibbon.vue'
import ResizeableWrapper from '@/features/common/ResizeableWrapper.vue'
import ServerPane from '@/features/server-pane/ServerPane.vue'
import DataBrowserPane from '@/features/data-browser/DataBrowserPane.vue'
import UnconnectedContent from '@/features/unconnected-content/UnconnectedContent.vue'
import TitleBar from '@/app/TitleBar.vue'
import UnifiedContentPane from '@/features/tabs/UnifiedContentPane.vue'
import WorkspacePane from '@/features/workspaces/WorkspacePane.vue'
import { useUpdateStore } from '@/features/updates/updateStore'

const themeVars = useThemeVars()
const props = defineProps<{
  loading: boolean
}>()

const i18n = useI18n()
const notification = useNotification()
const tabStore = useTabStore()
const serverStore = useServerStore()
const dataBrowserStore = useDataBrowserStore()
const settingsStore = useSettingsStore()
const updateStore = useUpdateStore()

runtime.EventsOn('oidc-reauth-required', (serverID: string) => {
  const server = serverStore.findServerById(serverID)
  const name = server?.name ?? serverID
  notification.warning({
    title: i18n.t('oidc.reAuthTitle'),
    content: i18n.t('oidc.reAuthMessage', { name }),
    duration: 10000,
  })
})

runtime.EventsOn('config-parse-error', (detail: string) => {
  notification.warning({
    title: i18n.t('errorTitles.configParseError'),
    content: i18n.t('errors.configParseError'),
    meta: detail,
    duration: 15000,
  })
})

const data = reactive({
  navMenuWidth: 50,
  toolbarHeight: 38,
})

const exThemeVars = computed(() => {
  return extraTheme(settingsStore.isDark)
})

const saveSidebarWidth = debounce(settingsStore.saveConfiguration, 1000, { trailing: true })
const handleResize = () => {
  saveSidebarWidth()
}

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
    borderRadius: '10px',
  }
})

const spinStyle = computed<CSSStyleValue>(() => {
  if (isWindows() || hideRadius.value) {
    return {
      backgroundColor: themeVars.value.bodyColor,
    }
  }

  return {
    backgroundColor: themeVars.value.bodyColor,
    borderRadius: '10px',
  }
})

const onToggleFullscreen = (fullscreen: boolean) => {
  hideRadius.value = fullscreen
}

const onToggleMaximize = (isMaximized: boolean) => {
  if (isMaximized) {
    if (!isMacOS()) {
      hideRadius.value = true
    }
  } else {
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
  updateStore.subscribe()
})

onBeforeUnmount(() => {
  updateStore.unsubscribe()
})

watch(
  () => updateStore.available,
  (isAvailable, wasAvailable) => {
    if (!isAvailable || wasAvailable) {
      return
    }
    notification.info({
      title: i18n.t('settings.updates.available', { version: updateStore.version }),
      duration: 0,
      meta: '',
      action: () =>
        h('div', { style: 'display: flex; gap: 8px;' }, [
          h(
            NButton,
            {
              text: true,
              type: 'primary',
              onClick: () => updateStore.openReleasePage(),
            },
            { default: () => i18n.t('settings.updates.viewRelease') },
          ),
          h(
            NButton,
            {
              text: true,
              onClick: () => updateStore.dismiss(),
            },
            { default: () => i18n.t('settings.updates.dismiss') },
          ),
        ]),
    })
  },
)
</script>

<template>
  <n-spin :show="props.loading" :style="spinStyle" :theme-overrides="{ opacitySpinning: 0 }">
    <div id="app-content-wrapper" :style="wrapperStyle" class="flex-box-v">
      <!-- title bar -->
      <title-bar :nav-menu-width="data.navMenuWidth" :toolbar-height="data.toolbarHeight" />

      <!-- content area -->
      <div
        id="app-content"
        :style="settingsStore.uiFont"
        class="flex-box-h flex-item-expand"
        style="--wails-draggable: none">
        <left-ribbon v-model:value="tabStore.nav" :width="data.navMenuWidth" />
        <div v-show="tabStore.nav === 'browser'" class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper
            v-model:size="settingsStore.window.asideWidth"
            :min-size="300"
            :offset="data.navMenuWidth"
            class="flex-item"
            @update:size="handleResize">
            <data-browser-pane
              v-show="dataBrowserStore.hasOpenConnections"
              class="app-side flex-item-expand" />
          </resizeable-wrapper>
          <unified-content-pane class="flex-item-expand" />
        </div>
        <div
          v-show="tabStore.nav === NavType.Servers"
          class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper
            v-model:size="settingsStore.window.asideWidth"
            :min-size="300"
            :offset="data.navMenuWidth"
            class="flex-item"
            @update:size="handleResize">
            <server-pane class="app-side flex-item-expand" />
          </resizeable-wrapper>
          <unconnected-content class="flex-item-expand" />
        </div>
        <div
          v-show="tabStore.nav === NavType.Workspaces"
          class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper
            v-model:size="settingsStore.window.asideWidth"
            :min-size="300"
            :offset="data.navMenuWidth"
            class="flex-item"
            @update:size="handleResize">
            <workspace-pane class="app-side flex-item-expand" />
          </resizeable-wrapper>
          <unified-content-pane class="flex-item-expand" />
        </div>
      </div>
    </div>
  </n-spin>
</template>

<style lang="scss" scoped>
#app-content-wrapper {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background-color: v-bind('themeVars.bodyColor');
  color: v-bind('themeVars.textColorBase');

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
