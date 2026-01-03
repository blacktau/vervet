<script setup lang="ts">
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { computed, ref, watchEffect } from 'vue'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useI18n } from 'vue-i18n'
import GeneralSettings from '@/features/settings/GeneralSettings.vue'
import EditorSettings from '@/features/settings/EditorSettings.vue'
import TerminalSettings from '@/features/settings/TerminalSettings.vue'

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


watchEffect(() => {
  console.log('SettingsDialog->visible',dialogStore.dialogs[DialogType.Settings].visible)

})

</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.Settings].visible"
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
      <n-tabs
        v-model:value="currentTab"
        animated
        pane-style="min-height: 300px"
        placement="left"
        tab-style="justify-content: right; font-weight: 420;"
        type="line">
        <n-tab-pane :tab="$t('settings.general.name')" display-directive="show:lazy" name="general">
          <general-settings :loading="loading" />
        </n-tab-pane>
        <n-tab-pane :tab="$t('settings.editor.name')" display-directive="show:lazy" name="editor">
          <editor-settings :loading="loading" />
        </n-tab-pane>
        <n-tab-pane :tab="$t('settings.terminal.name')" display-directive="show:lazy" name="editor">
          <terminal-settings :loading="loading" />
        </n-tab-pane>
      </n-tabs>
    </n-spin>
    <template #action>
      <div class="flex-item-expanded">
        <n-button :disabled="loading" @click="settingsStore.restoreConfiguration()">
                  {{ $t('settings.restoreDefaults') }}
        </n-button>
      </div>
      <div class="flex-item n-dialog-action">
        <n-button :disabled="loading" @click="onClose">{{ $t('common.cancel') }}</n-button>
        <n-button :disabled="loading" type="primary" @click="onSavePreferences">
          {{ $t('common.save') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped lang="scss"></style>
