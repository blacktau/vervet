<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watchEffect } from 'vue'
import { type FormInst, type FormItemRule } from 'naive-ui'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useNotifier } from '@/utils/dialog.ts'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'

const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const i18n = useI18n()
const notifier = useNotifier()

const loading = ref(false)
const formRef = ref<FormInst | null>(null)
const serverID = ref('')
const dbName = ref('')

const form = reactive({
  collectionName: '',
})

const collectionDollar = /\$/

const formRules = computed(() => ({
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
  if (dialogStore.dialogs[DialogType.AddCollection].visible) {
    const data = dialogStore.getDialogData<{ serverID: string; dbName: string }>(
      DialogType.AddCollection,
    )
    serverID.value = data?.serverID ?? ''
    dbName.value = data?.dbName ?? ''
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
    const result = await connectionsProxy.CreateCollection(
      serverID.value,
      dbName.value,
      form.collectionName,
    )
    if (!result.isSuccess) {
      notifier.error(result.error)
      return false
    }

    await browserStore.refreshDatabaseCollections(serverID.value, dbName.value)
    dialogStore.closeAddCollectionDialog()
  } catch (e) {
    const err = e as Error
    notifier.error(err.message)
  } finally {
    loading.value = false
  }
  return false
}

function onClose() {
  dialogStore.closeAddCollectionDialog()
}
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.AddCollection].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="$t('common.cancel')"
    :positive-button-props="{ size: 'medium', loading: loading }"
    :positive-text="$t('common.confirm')"
    :show-icon="false"
    :title="$t('dataBrowser.dialogs.addCollection.title')"
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
        :label="$t('dataBrowser.dialogs.addCollection.collectionName')"
        path="collectionName"
        required>
        <n-input
          v-model:value="form.collectionName"
          :placeholder="$t('dataBrowser.dialogs.addCollection.collectionNamePlaceholder')" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>
