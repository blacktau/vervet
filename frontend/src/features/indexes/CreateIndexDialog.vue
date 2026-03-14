<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { computed, reactive, ref, watchEffect } from 'vue'
import { type FormInst } from 'naive-ui'
import { DialogMode, DialogType, useDialogStore } from '@/stores/dialog.ts'
import { type IndexInfo, useIndexStore } from '@/features/indexes/indexStore.ts'
import { useNotifier } from '@/utils/dialog.ts'

type DialogData = {
  serverID: string
  dbName: string
  collectionName: string
  index?: IndexInfo
}

const dialogStore = useDialogStore()
const indexStore = useIndexStore()
const i18n = useI18n()
const notifier = useNotifier()

const loading = ref(false)
const formRef = ref<FormInst | null>(null)
const serverID = ref('')
const dbName = ref('')
const collectionName = ref('')
const editingIndexName = ref<string | undefined>(undefined)

const form = reactive({
  keys: [{ field: '', direction: 1 as number }],
  name: '',
  unique: false,
  sparse: false,
  ttl: null as number | null,
})

const isEditMode = computed(
  () => dialogStore.dialogs[DialogType.CreateIndex]?.type === DialogMode.Edit,
)

const dialogTitle = computed(() =>
  isEditMode.value
    ? i18n.t('indexes.dialogs.create.editTitle')
    : i18n.t('indexes.dialogs.create.title'),
)

watchEffect(() => {
  if (dialogStore.dialogs[DialogType.CreateIndex]?.visible) {
    const data = dialogStore.getDialogData<DialogData>(DialogType.CreateIndex)
    serverID.value = data?.serverID ?? ''
    dbName.value = data?.dbName ?? ''
    collectionName.value = data?.collectionName ?? ''

    if (data?.index) {
      editingIndexName.value = data.index.name
      form.keys = data.index.keys.map((k) => ({
        field: k.field,
        direction: typeof k.direction === 'number' ? k.direction : 1,
      }))
      form.name = data.index.name
      form.unique = data.index.unique
      form.sparse = data.index.sparse
      form.ttl = data.index.ttl ?? null
    } else {
      editingIndexName.value = undefined
      form.keys = [{ field: '', direction: 1 }]
      form.name = ''
      form.unique = false
      form.sparse = false
      form.ttl = null
    }
  }
})

function addKey() {
  form.keys.push({ field: '', direction: 1 })
}

function removeKey(index: number) {
  if (form.keys.length > 1) {
    form.keys.splice(index, 1)
  }
}

async function onConfirm() {
  const hasEmptyFields = form.keys.some((k) => !k.field.trim())
  if (hasEmptyFields || form.keys.length === 0) {
    notifier.error(i18n.t('indexes.dialogs.create.fieldRequired'))
    return false
  }

  loading.value = true
  try {
    const request = {
      keys: form.keys.map((k) => ({ field: k.field.trim(), direction: k.direction })),
      name: form.name || undefined,
      unique: form.unique,
      sparse: form.sparse,
      ttl: form.ttl ?? undefined,
    }

    if (isEditMode.value && editingIndexName.value) {
      const createSuccess = await indexStore.createIndex(
        serverID.value,
        dbName.value,
        collectionName.value,
        request,
      )
      if (!createSuccess) {
        return false
      }

      await indexStore.dropIndex(
        serverID.value,
        dbName.value,
        collectionName.value,
        editingIndexName.value,
      )
    } else {
      const success = await indexStore.createIndex(
        serverID.value,
        dbName.value,
        collectionName.value,
        request,
      )
      if (!success) {
        return false
      }
    }

    dialogStore.closeCreateIndexDialog()
  } catch (e) {
    const err = e as Error
    notifier.error(err.message)
    return false
  } finally {
    loading.value = false
  }
}

function onClose() {
  dialogStore.closeCreateIndexDialog()
}

const directionOptions = computed(() => [
  { label: i18n.t('indexes.dialogs.create.ascending'), value: 1 },
  { label: i18n.t('indexes.dialogs.create.descending'), value: -1 },
])
</script>

<template>
  <n-modal
    v-model:show="dialogStore.dialogs[DialogType.CreateIndex].visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="$t('common.cancel')"
    :positive-button-props="{ size: 'medium', loading: loading }"
    :positive-text="$t('common.confirm')"
    :show-icon="false"
    :title="dialogTitle"
    close-on-esc
    preset="dialog"
    style="width: 500px"
    transform-origin="center"
    @esc="onClose"
    @positive-click="onConfirm"
    @negative-click="onClose">
    <n-alert v-if="isEditMode" type="warning" style="margin-bottom: 12px">
      {{ $t('indexes.dialogs.create.editWarning') }}
    </n-alert>
    <n-form ref="formRef" :show-label="true" label-placement="top">
      <n-form-item :label="$t('indexes.dialogs.create.keys')">
        <div style="width: 100%">
          <div
            v-for="(key, index) in form.keys"
            :key="index"
            style="display: flex; gap: 8px; margin-bottom: 8px; align-items: center">
            <n-input
              v-model:value="key.field"
              :placeholder="$t('indexes.dialogs.create.field')"
              style="flex: 1" />
            <n-select
              v-model:value="key.direction"
              :options="directionOptions"
              style="width: 160px" />
            <n-button
              :disabled="form.keys.length <= 1"
              size="small"
              quaternary
              @click="removeKey(index)">
              &times;
            </n-button>
          </div>
          <n-button size="small" dashed @click="addKey">
            {{ $t('indexes.dialogs.create.addKey') }}
          </n-button>
        </div>
      </n-form-item>
      <n-form-item :label="$t('indexes.dialogs.create.name')">
        <n-input
          v-model:value="form.name"
          :placeholder="$t('indexes.dialogs.create.namePlaceholder')" />
      </n-form-item>
      <div style="display: flex; gap: 24px">
        <n-checkbox v-model:checked="form.unique">
          {{ $t('indexes.dialogs.create.unique') }}
        </n-checkbox>
        <n-checkbox v-model:checked="form.sparse">
          {{ $t('indexes.dialogs.create.sparse') }}
        </n-checkbox>
      </div>
      <n-form-item :label="$t('indexes.dialogs.create.ttl')" style="margin-top: 12px">
        <n-input-number
          v-model:value="form.ttl"
          :placeholder="$t('indexes.dialogs.create.ttlPlaceholder')"
          :min="0"
          clearable
          style="width: 100%" />
      </n-form-item>
    </n-form>
  </n-modal>
</template>
