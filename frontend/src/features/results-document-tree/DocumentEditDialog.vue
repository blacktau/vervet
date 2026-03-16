<script lang="ts" setup>
import { ref, watch, computed, nextTick, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { humanizeEjson, dehumanizeEjson } from './humanizeEjson'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import * as monaco from 'monaco-editor'

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
const documentId = ref<unknown>(null)

const title = computed(() =>
  props.mode === 'edit'
    ? t('query.dialogs.editDocument')
    : t('query.dialogs.insertDocument'),
)

const idDisplay = computed(() => {
  if (!documentId.value) {
    return ''
  }
  return JSON.stringify(documentId.value)
})

function prepareDocument(doc: unknown): string {
  if (props.mode === 'insert') {
    return '{\n  \n}'
  }

  const humanized = humanizeEjson(doc) as Record<string, unknown>
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { _id, ...rest } = humanized
  documentId.value = (doc as Record<string, unknown>)?._id ?? null
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
      } else if (editor.value) {
        editor.value.setValue(content)
      }
    } else {
      if (editor.value) {
        editor.value.dispose()
        editor.value = null
      }
      documentId.value = null
    }
  },
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
  if (!editor.value) {
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

  // Dehumanize (convert ISO strings back to $date, etc.)
  const dehumanized = dehumanizeEjson(parsed) as Record<string, unknown>

  saving.value = true
  try {
    let query: string
    if (props.mode === 'edit') {
      const filter = JSON.stringify({ _id: documentId.value })
      const replacement = JSON.stringify(dehumanized)
      query = `db.getCollection('${props.collectionName}').replaceOne(${filter}, ${replacement})`
    } else {
      query = `db.getCollection('${props.collectionName}').insertOne(${JSON.stringify(dehumanized)})`
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
    <div v-if="mode === 'edit' && idDisplay" class="id-display">
      <n-text depth="3">_id: {{ idDisplay }}</n-text>
    </div>
    <div ref="container" class="editor-container" />
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
</style>
