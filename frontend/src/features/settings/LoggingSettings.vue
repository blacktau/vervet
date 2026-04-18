<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { useNotifier } from '@/utils/dialog.ts'
import * as systemProxy from 'wailsjs/go/api/SystemProxy'

defineProps<{ loading: boolean }>()

const { t } = useI18n()
const settingsStore = useSettingsStore()
const dialog = useDialog()
const notifier = useNotifier()

const levelOptions = computed(() =>
  settingsStore.logLevelOptions.map((o) => ({ value: o.value, label: t(o.label) })),
)

async function revealFolder() {
  const result = await systemProxy.RevealLogsFolder()
  if (!result.isSuccess) {
    notifier.error(t(`errors.${result.errorCode}`), { detail: result.errorDetail })
  }
}

function onReveal() {
  if (settingsStore.logging.fileEnabled) {
    revealFolder()
    return
  }
  dialog.warning({
    title: t('settings.logging.revealDisabledTitle'),
    content: t('settings.logging.revealDisabledBody'),
    positiveText: t('common.ok'),
    negativeText: t('common.cancel'),
    onPositiveClick: revealFolder,
  })
}
</script>

<template>
  <n-form
    :disabled="loading"
    :model="settingsStore.logging"
    :show-require-mark="false"
    label-placement="top">
    <n-grid :x-gap="10">
      <n-form-item-gi :label="$t('settings.logging.level')" :span="24">
        <n-select v-model:value="settingsStore.logging.level" :options="levelOptions" />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.logging.console')" :span="24">
        <n-flex vertical size="small">
          <n-switch v-model:value="settingsStore.logging.consoleEnabled" />
          <n-text depth="3" style="font-size: 12px">
            {{ $t('settings.logging.consoleHelp') }}
          </n-text>
        </n-flex>
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.logging.file')" :span="24">
        <n-flex vertical size="small">
          <n-switch v-model:value="settingsStore.logging.fileEnabled" />
          <n-text depth="3" style="font-size: 12px">
            {{ $t('settings.logging.fileHelp') }}
          </n-text>
        </n-flex>
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.logging.maxSizeMB')" :span="12">
        <n-input-number
          v-model:value="settingsStore.logging.maxSizeMB"
          :disabled="!settingsStore.logging.fileEnabled"
          :min="1"
          :max="1024" />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.logging.maxBackups')" :span="12">
        <n-input-number
          v-model:value="settingsStore.logging.maxBackups"
          :disabled="!settingsStore.logging.fileEnabled"
          :min="0"
          :max="100" />
      </n-form-item-gi>
      <n-form-item-gi :span="24">
        <n-button @click="onReveal">{{ $t('settings.logging.revealFolder') }}</n-button>
      </n-form-item-gi>
      <n-form-item-gi :span="24">
        <n-text depth="3" style="font-size: 12px">
          {{ $t('settings.logging.restartHint') }}
        </n-text>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style lang="scss" scoped></style>