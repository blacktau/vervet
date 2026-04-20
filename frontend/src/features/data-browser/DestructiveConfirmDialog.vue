<script lang="ts" setup>
import { computed, reactive, ref, watchEffect } from 'vue'
import { useI18n } from 'vue-i18n'
import { type FormInst, type FormItemRule } from 'naive-ui'
import {
  DialogType,
  useDialogStore,
  type DestructiveConfirmData,
} from '@/stores/dialog.ts'
import { useNotifier } from '@/utils/dialog.ts'

const dialogStore = useDialogStore()
const notifier = useNotifier()
const { t } = useI18n()

const formRef = ref<FormInst | null>(null)
const loading = ref(false)

const form = reactive({
  typedName: '',
})

const data = ref<DestructiveConfirmData | null>(null)

watchEffect(() => {
  if (dialogStore.dialogs[DialogType.DestructiveConfirm].visible) {
    data.value = dialogStore.getDialogData<DestructiveConfirmData>(DialogType.DestructiveConfirm) ?? null
    form.typedName = ''
    loading.value = false
  } else {
    data.value = null
  }
})

const matches = computed(() => {
  if (!data.value) {
    return false
  }
  return form.typedName.trim() === data.value.name
})

const formRules = computed(() => ({
  typedName: [
    {
      validator: (_rule: FormItemRule, value: string) => {
        if (!data.value) {
          return false
        }
        return (value ?? '').trim() === data.value.name
      },
      message: t('dataBrowser.dialogs.destructiveConfirm.mismatchHelp'),
      trigger: 'input',
    },
  ],
}))

const title = computed(() => {
  if (!data.value) {
    return ''
  }
  return t(`dataBrowser.dialogs.destructiveConfirm.${data.value.kind}.title`)
})

const inputLabel = computed(() => {
  if (!data.value) {
    return ''
  }
  return t(`dataBrowser.dialogs.destructiveConfirm.${data.value.kind}.inputLabel`)
})

const impactText = computed(() => {
  if (!data.value) {
    return ''
  }
  const { kind, impact } = data.value
  if (kind === 'database') {
    const hasCollections = typeof impact.collectionCount === 'number'
    const hasDocs = typeof impact.documentCount === 'number'
    if (hasCollections && hasDocs) {
      return t('dataBrowser.dialogs.destructiveConfirm.database.impactWithCounts', {
        collections: impact.collectionCount!.toLocaleString(),
        documents: impact.documentCount!.toLocaleString(),
      })
    }
    if (hasCollections) {
      return t('dataBrowser.dialogs.destructiveConfirm.database.impactCollectionsOnly', {
        collections: impact.collectionCount!.toLocaleString(),
      })
    }
    return t('dataBrowser.dialogs.destructiveConfirm.database.impactUnavailable')
  }
  if (typeof impact.documentCount === 'number') {
    return t('dataBrowser.dialogs.destructiveConfirm.collection.impactDocuments', {
      documents: impact.documentCount.toLocaleString(),
    })
  }
  return t('dataBrowser.dialogs.destructiveConfirm.collection.impactUnavailable')
})

async function onConfirm() {
  if (!data.value) {
    return false
  }
  try {
    await formRef.value?.validate()
  } catch {
    return false
  }
  if (!matches.value) {
    return false
  }

  loading.value = true
  try {
    await data.value.onConfirm()
  } catch (e) {
    const err = e as Error
    notifier.error(err.message)
  } finally {
    loading.value = false
    dialogStore.closeDestructiveConfirmDialog()
  }
  return false
}

function onClose() {
  dialogStore.closeDestructiveConfirmDialog()
}
</script>

<template>
  <n-modal
    v-if="data"
    v-model:show="dialogStore.dialogs[DialogType.DestructiveConfirm].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium', disabled: loading }"
    :negative-text="$t('dataBrowser.dialogs.destructiveConfirm.cancel')"
    :positive-button-props="{
      size: 'medium',
      type: 'error',
      disabled: !matches || loading,
      loading: loading,
    }"
    :positive-text="$t('dataBrowser.dialogs.destructiveConfirm.confirm')"
    :show-icon="false"
    :title="title"
    close-on-esc
    preset="dialog"
    transform-origin="center"
    @esc="onClose"
    @positive-click="onConfirm"
    @negative-click="onClose">
    <n-space vertical :size="16">
      <n-alert type="warning" :bordered="false" :show-icon="true">
        <i18n-t
          :keypath="`dataBrowser.dialogs.destructiveConfirm.${data.kind}.warning`"
          tag="span">
          <template #name>
            <n-text code>{{ data.name }}</n-text>
          </template>
        </i18n-t>
      </n-alert>
      <n-text depth="2">{{ impactText }}</n-text>
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        :show-require-mark="false"
        label-placement="top">
        <n-form-item :label="inputLabel" path="typedName">
          <n-input
            v-model:value="form.typedName"
            :placeholder="data.name"
            :autofocus="true" />
        </n-form-item>
      </n-form>
    </n-space>
  </n-modal>
</template>
