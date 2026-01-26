<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useI18n } from 'vue-i18n'
import { onMounted, ref, watch } from 'vue'
import { WindowSetDarkTheme, WindowSetLightTheme } from 'wailsjs/runtime'
import { darkTheme, type NLocale } from 'naive-ui'
import { darkThemeOverrides, themeOverrides } from '@/utils/theme'
import { useServerStore } from '@/features/server-pane/serverStore.ts'
import AppContent from '@/app/AppContent.vue'
import AboutDialog from '@/features/about/AboutDialog.vue'
import GroupDialog from '@/features/server-pane/GroupDialog.vue'
import ServerDialog from '@/features/server-pane/ServerDialog.vue'
import SettingsDialog from '@/features/settings/SettingsDialog.vue'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

const settingsStore = useSettingsStore()
const serverStore = useServerStore()
const browserStore = useDataBrowserStore()

const i18n = useI18n()
const initializing = ref(true)

const locale = ref<NLocale | undefined>(undefined)

onMounted(async () => {
  try {
    initializing.value = true
    await settingsStore.loadSettings()
    await settingsStore.loadFontList()
    await serverStore.refreshServers()
    await browserStore.refreshConnectedServers(true)
  } finally {
    initializing.value = false
  }
})

watch(
  () => settingsStore.isDark,
  (isDark: boolean) => {
    if (isDark) {
      WindowSetDarkTheme()
    } else {
      WindowSetLightTheme()
    }
  },
)

watch(
  () => settingsStore.general.language,
  (lang: string) => {
    i18n.locale.value = settingsStore.currentLanguage
  },
)
watch(
  () => settingsStore.general.font.family,
  (font: string) => {
    const body = document.getElementsByName('body')[0]
    if (body != null) {
      body.style = `font-family: '${font}'`
    }
  },
)
</script>

<template>
  <n-config-provider
    :inline-theme-disabled="true"
    :locale="locale"
    :theme="settingsStore.isDark ? darkTheme : undefined"
    :theme-overrides="settingsStore.isDark ? darkThemeOverrides : themeOverrides"
    class="fill-height">
    <n-dialog-provider>
      <app-content :loading="initializing" />
      <server-dialog />
      <group-dialog />
      <settings-dialog />
      <about-dialog />
    </n-dialog-provider>
  </n-config-provider>
</template>

<style lang="scss"></style>
