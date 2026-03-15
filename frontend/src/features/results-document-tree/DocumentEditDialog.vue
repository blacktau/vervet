<script lang="ts" setup>
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifier } from '@/utils/dialog'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'

const props = defineProps<{
  show: boolean
  document: unknown
  mode: 'edit' | 'insert'
  serverId: string
  dbName: string
  collectionName: string
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
  saved: []
}>()

const { t } = useI18n()
const notifier = useNotifier()

const jsonText = ref('')
const jsonError = ref('')
const saving = ref(false)

const title = computed(() =>
  props.mode === 'edit'
    ? t('query.dialogs.editDocument')
    : t('query.dialogs.insertDocument'),
)

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      jsonText.value = JSON.stringify(props.document, null, 2)
      jsonError.value = ''
    }
  },
)

async function save() {
  jsonError.value = ''

  try {
    JSON.parse(jsonText.value)
  } catch (e) {
    jsonError.value = t('query.dialogs.invalidJson', { error: (e as Error).message })
    return
  }

  saving.value = true
  try {
    let query: string
    if (props.mode === 'edit') {
      const doc = props.document as Record<string, unknown>
      const filter = JSON.stringify({ _id: doc._id })
      const replacement = jsonText.value
      query = `db.getCollection('${props.collectionName}').replaceOne(${filter}, ${replacement})`
    } else {
      query = `db.getCollection('${props.collectionName}').insertOne(${jsonText.value})`
    }

    const result = await shellProxy.ExecuteQuery(props.serverId, props.dbName, query)
    if (result.isSuccess) {
      emit('saved')
      emit('update:show', false)
    } else {
      notifier.error(result.error)
    }
  } catch (e) {
    notifier.error(String(e))
  } finally {
    saving.value = false
  }
}

function close() {
  emit('update:show', false)
}
</script>

<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="title"
    style="width: 700px; max-height: 80vh"
    :mask-closable="false"
    @update:show="emit('update:show', $event)"
  >
    <n-input
      v-model:value="jsonText"
      type="textarea"
      :rows="20"
      :placeholder="mode === 'insert' ? '{ }' : ''"
      style="font-family: monospace; font-size: 13px"
    />
    <n-text v-if="jsonError" type="error" style="display: block; margin-top: 8px; font-size: 12px">
      {{ jsonError }}
    </n-text>
    <template #footer>
      <n-space justify="end">
        <n-button @click="close">
          {{ t('common.cancel') }}
        </n-button>
        <n-button type="primary" :loading="saving" @click="save">
          {{ t('query.dialogs.save') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>
