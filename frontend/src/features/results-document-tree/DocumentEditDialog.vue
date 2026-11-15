<script lang="ts" setup>
import { ref, watch, computed, nextTick, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import {
  documentIdFilter,
  ejsonParseExpression,
  toEditableEjson,
  writeCount,
} from './editableEjson'
import { formatNamespace } from './useDocumentContextMenu'
import * as shellProxy from 'bindings/vervet/internal/api/shellproxy'
import * as monaco from 'monaco-editor'
import { errorCodeText } from '@/utils/resultError'

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
const settingsStore = useSettingsStore()

const container = ref<HTMLElement | null>(null)
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
const jsonError = ref('')
const saving = ref(false)
const documentId = ref<unknown>(undefined)

const namespace = computed(() => formatNamespace(props.dbName, props.collectionName))

const title = computed(() =>
  props.mode === 'edit'
    ? t('query.dialogs.editDocumentIn', { namespace: namespace.value })
    : t('query.dialogs.insertDocumentIn', { namespace: namespace.value }),
)

const idDisplay = computed(() => {
  if (documentId.value === undefined) {
    return ''
  }
  return JSON.stringify(documentId.value)
})

function prepareDocument(doc: unknown): string {
  if (props.mode === 'insert') {
    return '{\n  \n}'
  }

  const editable = toEditableEjson(doc) as Record<string, unknown>
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { _id, ...rest } = editable
  documentId.value = (doc as Record<string, unknown>)?._id
  return JSON.stringify(rest, null, 2)
}

watch(
  () => props.show,
  async (visible) => {
    if (visible) {
      jsonError.value = ''
      const content = prepareDocument(props.document)

      await nextTick()
      if (container.value && !editor.value) {
        editor.value = monaco.editor.create(container.value, {
          value: content,
          language: 'json',
          theme: settingsStore.isDark ? 'vervet-dark' : 'vervet-light',
          automaticLayout: true,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          wordWrap: 'on',
          padding: { top: 8 },
        })
        editor.value.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
          save()
        })
      } else if (editor.value) {
        editor.value.setValue(content)
      }
    } else {
      if (editor.value) {
        editor.value.dispose()
        editor.value = null
      }
      documentId.value = undefined
    }
  },
  // The table mounts this dialog when it first opens, already shown
  { immediate: true },
)

watch(
  () => settingsStore.isDark,
  (isDark) => {
    if (editor.value) {
      monaco.editor.setTheme(isDark ? 'vervet-dark' : 'vervet-light')
    }
  },
)

async function save() {
  // Ctrl+S reaches here while a save is running, and would insert the document twice
  if (!editor.value || saving.value) {
    return
  }

  jsonError.value = ''
  const text = editor.value.getValue()

  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch (e) {
    jsonError.value = t('query.dialogs.invalidJson', { error: (e as Error).message })
    return
  }
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    jsonError.value = t('query.dialogs.documentMustBeObject')
    return
  }

  saving.value = true
  try {
    const jsBody = ejsonParseExpression(text)
    let query: string
    const collection = JSON.stringify(props.collectionName)
    if (props.mode === 'edit') {
      const filter = documentIdFilter(documentId.value)
      query = `db.getCollection(${collection}).replaceOne(${filter}, ${jsBody})`
    } else {
      query = `db.getCollection(${collection}).insertOne(${jsBody})`
    }

    const result = await shellProxy.ExecuteQuery(
      props.serverId,
      crypto.randomUUID(),
      props.dbName,
      query,
      '',
    )
    if (result.isSuccess) {
      // Keep the dialog open, so the edits aren't lost
      if (props.mode === 'edit' && writeCount(result.data?.documents, 'matchedCount') === 0) {
        jsonError.value = t('query.dialogs.documentNotFound', { namespace: namespace.value })
        return
      }
      emit('saved')
      emit('update:show', false)
    } else {
      notifier.error(errorCodeText(result.errorCode), { title: t(props.mode === 'edit' ? 'errorTitles.updateDocument' : 'errorTitles.insertDocument'), detail: result.errorDetail })
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
    <div v-if="mode === 'edit' && idDisplay" class="id-display">
      <n-text depth="3">_id: {{ idDisplay }}</n-text>
    </div>
    <div ref="container" class="editor-container" />
    <n-text depth="3" class="ejson-hint">{{ t('query.dialogs.extendedJsonHint') }}</n-text>
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

<style lang="scss" scoped>
.id-display {
  margin-bottom: 8px;
  padding: 6px 10px;
  border-radius: 4px;
  background-color: var(--n-color-modal);
  border: 1px solid var(--n-border-color);
  font-family: monospace;
  font-size: 13px;
}

.editor-container {
  height: 400px;
  width: 100%;
}

.ejson-hint {
  display: block;
  margin-top: 8px;
  font-size: 12px;
}
</style>
