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
import AddDatabaseDialog from '@/features/data-browser/AddDatabaseDialog.vue'
import AddCollectionDialog from '@/features/data-browser/AddCollectionDialog.vue'
import CreateIndexDialog from '@/features/indexes/CreateIndexDialog.vue'
import RenameCollectionDialog from '@/features/data-browser/RenameCollectionDialog.vue'
import DestructiveConfirmDialog from '@/features/data-browser/DestructiveConfirmDialog.vue'
import ServerPickerDialog from '@/features/workspaces/ServerPickerDialog.vue'
import ExportResultsDialog from '@/features/results-export/ExportResultsDialog.vue'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import hljs from 'highlight.js/lib/core'

hljs.registerLanguage('vervet-log', () => ({
  contains: [
    {
      className: 'deletion',
      begin: /^.*\[ERROR\].*$/,
      relevance: 10,
    },
    {
      className: 'warning',
      begin: /^.*\[WARNING\].*$/,
      relevance: 10,
    },
  ],
}))

const settingsStore = useSettingsStore()
const serverStore = useServerStore()
const browserStore = useDataBrowserStore()
const dialogStore = useDialogStore()

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
  () => {
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
    :hljs="hljs"
    :theme="settingsStore.isDark ? darkTheme : undefined"
    :theme-overrides="settingsStore.isDark ? darkThemeOverrides : themeOverrides"
    class="fill-height">
    <n-notification-provider>
      <n-dialog-provider>
        <app-content :loading="initializing" />
        <server-dialog v-if="dialogStore.isVisible(DialogType.Server)" />
        <group-dialog v-if="dialogStore.isVisible(DialogType.Group)" />
        <settings-dialog v-if="dialogStore.isVisible(DialogType.Settings)" />
        <about-dialog v-if="dialogStore.isVisible(DialogType.About)" />
        <add-database-dialog v-if="dialogStore.isVisible(DialogType.AddDatabase)" />
        <add-collection-dialog v-if="dialogStore.isVisible(DialogType.AddCollection)" />
        <create-index-dialog v-if="dialogStore.isVisible(DialogType.CreateIndex)" />
        <rename-collection-dialog v-if="dialogStore.isVisible(DialogType.RenameCollection)" />
        <destructive-confirm-dialog v-if="dialogStore.isVisible(DialogType.DestructiveConfirm)" />
        <server-picker-dialog v-if="dialogStore.isVisible(DialogType.ServerPicker)" />
        <export-results-dialog
          v-if="dialogStore.isVisible(DialogType.ExportResults)"
          :show="dialogStore.isVisible(DialogType.ExportResults)"
          :ejson="dialogStore.exportResultsData.ejson"
          :collection-name="dialogStore.exportResultsData.collectionName"
          @update:show="(v) => { if (!v) dialogStore.closeExportResultsDialog() }" />
      </n-dialog-provider>
    </n-notification-provider>
  </n-config-provider>
</template>

<style lang="scss">
.hljs-deletion {
  color: #e06c75;
}

.hljs-emphasis {
  color: #e5c07b;
  font-style: normal;
}

.hljs-comment {
  color: #7f848e;
}

.hljs-warning {
  color: #e8a838;
}
</style>
