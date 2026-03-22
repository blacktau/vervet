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
const dbName = ref('')
const oldName = ref('')

const form = reactive({
  newName: '',
})

const collectionDollar = /\$/

const formRules = computed(() => ({
  newName: [
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
    {
      validator: (_rule: FormItemRule, value: string) => value !== oldName.value,
      message: i18n.t('dataBrowser.dialogs.renameCollection.sameName'),
      trigger: 'input',
    },
  ],
}))

watchEffect(() => {
  if (dialogStore.dialogs[DialogType.RenameCollection].visible) {
    const data = dialogStore.getDialogData<{
      serverID: string
      dbName: string
      collectionName: string
    }>(DialogType.RenameCollection)
    serverID.value = data?.serverID ?? ''
    dbName.value = data?.dbName ?? ''
    oldName.value = data?.collectionName ?? ''
    form.newName = ''
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
    const result = await collectionsProxy.RenameCollection(
      serverID.value,
      dbName.value,
      oldName.value,
      form.newName,
    )
    if (!result.isSuccess) {
      notifier.error(i18n.t(`errors.${result.errorCode}`), { title: i18n.t('errorTitles.renameCollection'), detail: result.errorDetail })
      return false
    }

    await browserStore.refreshDatabaseCollections(serverID.value, dbName.value)
    dialogStore.closeRenameCollectionDialog()
  } catch (e) {
    const err = e as Error
    notifier.error(err.message)
  } finally {
    loading.value = false
  }
  return false
}

function onClose() {
  dialogStore.closeRenameCollectionDialog()
}
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.RenameCollection].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="$t('common.cancel')"
    :positive-button-props="{ size: 'medium', loading: loading }"
    :positive-text="$t('common.confirm')"
    :show-icon="false"
    :title="$t('dataBrowser.dialogs.renameCollection.title')"
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
        :label="$t('dataBrowser.dialogs.renameCollection.newName')"
        path="newName"
        required>
        <n-input
          v-model:value="form.newName"
          :placeholder="$t('dataBrowser.dialogs.renameCollection.newNamePlaceholder')" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>
