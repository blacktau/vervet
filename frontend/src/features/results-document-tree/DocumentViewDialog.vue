<script lang="ts" setup>
import { computed, ref, watch, nextTick, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { humanizeEjson } from './humanizeEjson'
import * as monaco from 'monaco-editor'

const props = defineProps<{
  show: boolean
  document: unknown
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const { t } = useI18n()
const notifier = useNotifier()
const settingsStore = useSettingsStore()

const container = ref<HTMLElement | null>(null)
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)

const jsonText = computed(() => JSON.stringify(humanizeEjson(props.document), null, 2))

watch(
  () => props.show,
  async (visible) => {
    if (visible) {
      await nextTick()
      if (container.value && !editor.value) {
        editor.value = monaco.editor.create(container.value, {
          value: jsonText.value,
          language: 'json',
          theme: settingsStore.isDark ? 'vervet-dark' : 'vervet-light',
          automaticLayout: true,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          wordWrap: 'on',
          padding: { top: 8 },
          readOnly: true,
          domReadOnly: true,
          folding: true,
        })
      } else if (editor.value) {
        editor.value.setValue(jsonText.value)
      }
    } else {
      if (editor.value) {
        editor.value.dispose()
        editor.value = null
      }
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

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(jsonText.value)
    notifier.success(t('query.contextMenu.copied'))
  } catch {
    notifier.error('Failed to copy to clipboard')
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
    :title="t('query.dialogs.viewDocument')"
    style="width: 700px; max-height: 80vh"
    :mask-closable="true"
    @update:show="emit('update:show', $event)"
  >
    <div ref="container" class="editor-container" />
    <template #footer>
      <n-space justify="end">
        <n-button @click="copyToClipboard">
          {{ t('query.contextMenu.copyDocument') }}
        </n-button>
        <n-button @click="close">
          {{ t('common.cancel') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>

<style lang="scss" scoped>
.editor-container {
  height: 400px;
  width: 100%;
}
</style>
