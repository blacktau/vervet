<script setup lang="ts">
import { useSettingsStore } from '@/stores/settings'
import { useI18n } from 'vue-i18n'
import { onMounted, ref, watch } from 'vue'
import * as runtime from 'wailsjs/runtime'
import { WindowSetDarkTheme, WindowSetLightTheme } from 'wailsjs/runtime'
import { darkTheme, type NLocale } from 'naive-ui'
import { darkThemeOverrides, themeOverrides } from '@/utils/theme'
import { useServerStore } from '@/components/server-pane/serverStore.ts'
import AppContent from '@/app/AppContent.vue'
import AboutDialog from '@/dialogs/AboutDialog.vue'
import GroupDialog from '@/dialogs/GroupDialog.vue'
import ServerDialog from '@/components/server-pane/ServerDialog.vue'

const settingsStore = useSettingsStore()
const serverStore = useServerStore()
const i18n = useI18n()
const initializing = ref(true)

const locale = ref<NLocale | undefined>(undefined)

onMounted(async () => {
  try {
    initializing.value = true
    await settingsStore.loadSettings()
    await settingsStore.loadFontList()
    await serverStore.refreshServers()
  } finally {
    initializing.value = false
  }
})

watch(
  () => settingsStore.isDark,
  (isDark: boolean) => {
    isDark ? WindowSetDarkTheme() : WindowSetLightTheme()
  },
)

watch(
  () => settingsStore.general.language,
  (lang: string) => {
    i18n.locale.value = settingsStore.currentLanguage
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
