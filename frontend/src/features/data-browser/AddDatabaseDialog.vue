<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watchEffect } from 'vue'
import { type FormInst, type FormItemRule } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useNotifier } from '@/utils/dialog.ts'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'

const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const i18n = useI18n()
const notifier = useNotifier()

const loading = ref(false)
const formRef = ref<FormInst | null>(null)
const serverID = ref('')

const form = reactive({
  databaseName: '',
  collectionName: '',
})

const dbForbiddenChars = /[/\\. "$*<>:|?]/
const collectionDollar = /\$/

const formRules = computed(() => ({
  databaseName: [
    {
      required: true,
      message: i18n.t('common.dialog.fieldRequired'),
      trigger: 'input',
    },
    {
      validator: (_rule: FormItemRule, value: string) => value.length <= 64,
      message: i18n.t('dataBrowser.dialogs.addDatabase.maxDbLength'),
      trigger: 'input',
    },
    {
      validator: (_rule: FormItemRule, value: string) => !dbForbiddenChars.test(value),
      message: i18n.t('dataBrowser.dialogs.addDatabase.invalidDbChars'),
      trigger: 'input',
    },
  ],
  collectionName: [
    {
      required: true,
      message: i18n.t('common.dialog.fieldRequired'),
      trigger: 'input',
    },
    {
      validator: (_rule: FormItemRule, value: string) => !value.startsWith('system.'),
      message: i18n.t('dataBrowser.dialogs.addDatabase.systemPrefix'),
      trigger: 'input',
    },
    {
      validator: (_rule: FormItemRule, value: string) => !collectionDollar.test(value),
      message: i18n.t('dataBrowser.dialogs.addDatabase.dollarSign'),
      trigger: 'input',
    },
  ],
}))

watchEffect(() => {
  if (dialogStore.dialogs[DialogType.AddDatabase].visible) {
    const sid = dialogStore.getDialogData<string>(DialogType.AddDatabase)
    serverID.value = sid ?? ''
    form.databaseName = ''
    form.collectionName = ''
  }
})

async function onConfirm() {
  try {
    await formRef.value?.validate()
  } catch {
    return false
  }

  loading.value = true
  try {
    const result = await collectionsProxy.CreateCollection(
      serverID.value,
      form.databaseName,
      form.collectionName,
    )
    if (!result.isSuccess) {
      notifier.error(i18n.t(`errors.${result.errorCode}`), { title: i18n.t('errorTitles.createDatabase'), detail: result.errorDetail })
      return false
    }

    await browserStore.refreshServerDatabases(serverID.value)
    dialogStore.closeAddDatabaseDialog()
  } catch (e) {
    const err = e as Error
    notifier.error(err.message)
  } finally {
    loading.value = false
  }
  return false
}

function onClose() {
  dialogStore.closeAddDatabaseDialog()
}
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.AddDatabase].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="$t('common.cancel')"
    :positive-button-props="{ size: 'medium', loading: loading }"
    :positive-text="$t('common.confirm')"
    :show-icon="false"
    :title="$t('dataBrowser.dialogs.addDatabase.title')"
    close-on-esc
    preset="dialog"
    transform-origin="center"
    @esc="onClose"
    @positive-click="onConfirm"
    @negative-click="onClose">
    <n-form
      ref="formRef"
      :model="form"
      :rules="formRules"
      :show-label="false"
      :show-require-mark="false"
      label-placement="top">
      <n-form-item
        :label="$t('dataBrowser.dialogs.addDatabase.databaseName')"
        path="databaseName"
        required>
        <n-input
          v-model:value="form.databaseName"
          :placeholder="$t('dataBrowser.dialogs.addDatabase.databaseNamePlaceholder')" />
      </n-form-item>
      <n-form-item
        :label="$t('dataBrowser.dialogs.addDatabase.collectionName')"
        path="collectionName"
        required>
        <n-input
          v-model:value="form.collectionName"
          :placeholder="$t('dataBrowser.dialogs.addDatabase.collectionNamePlaceholder')" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>
