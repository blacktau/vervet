<script setup lang="ts">
import { useSettingsStore } from '@/features/settings/settings.ts'
import { ref, watchEffect } from 'vue'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useI18n } from 'vue-i18n'
import GeneralSettings from '@/features/settings/GeneralSettings.vue'
import EditorSettings from '@/features/settings/EditorSettings.vue'

const settingsStore = useSettingsStore()
const dialogStore = useDialogStore()
const i18n = useI18n()

const previousSettings = ref({})
const currentTab = ref('general')
const loading = ref<boolean>(false)

const initSettings = async () => {
  try {
    loading.value = true
    currentTab.value = dialogStore.getDialogData<string>(DialogType.Settings) || 'general'
    await settingsStore.loadSettings()
    previousSettings.value = {
      general: settingsStore.general,
      editor: settingsStore.editor,
      terminal: settingsStore.terminal,
    }
  } finally {
    loading.value = false
  }
}

watchEffect(() => {
  if (dialogStore.isVisible(DialogType.Settings)) {
    initSettings()
  }
})

const onSavePreferences = async () => {
  const success = await settingsStore.saveConfiguration()
  if (success) {
    dialogStore.hide(DialogType.Settings)
  }
}

const onClose = async () => {
  await settingsStore.loadSettings()
  dialogStore.hide(DialogType.Settings)
}
</script>

<template>
  <n-modal :v-model:show="dialogStore.isVisible(DialogType.Settings)"
           :auto-focus="false"
           :closable="false"
           :mask-closable="false"
           :show-icon="false"
           close-on-esc
           preset="dialog"
           style="width: 640px"
           transform-origin="center"
           @esc="onClose">
    <n-spin :show="loading">
      <n-tab-pane :tab="$t('settings.general.name')" display-directive="show:lazy" name="general">
        <general-settings :loading="loading" />
      </n-tab-pane>
      <n-tab-pane :tab="$t('settings.editor.name')" display-directive="show:lazy" name="editor">
        <editor-settings :loading="loading" />
      </n-tab-pane>
    </n-spin>
  </n-modal>
</template>

<style scoped lang="scss"></style>
