import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { useSettingsStore } from '@/features/settings/settingsStore'
import * as monaco from 'monaco-editor'

interface MonacoEditorOptions {
  language: string
  value?: string
  readOnly?: boolean
  fontSize?: number
  extraOptions?: monaco.editor.IStandaloneEditorConstructionOptions
}

export function useMonacoEditor(options: MonacoEditorOptions) {
  const settingsStore = useSettingsStore()
  const container = ref<HTMLElement | null>(null)
  const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)

  function init() {
    if (!container.value) {
      return
    }

    editor.value = monaco.editor.create(container.value, {
      value: options.value ?? '',
      language: options.language,
      theme: settingsStore.isDark ? 'vervet-dark' : 'vervet-light',
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: options.fontSize ?? 14,
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      padding: { top: 8 },
      readOnly: options.readOnly ?? false,
      ...options.extraOptions,
    })
  }

  watch(
    () => settingsStore.isDark,
    (isDark) => {
      if (editor.value) {
        monaco.editor.setTheme(isDark ? 'vervet-dark' : 'vervet-light')
      }
    },
  )

  onMounted(() => {
    init()
  })

  onBeforeUnmount(() => {
    if (editor.value) {
      editor.value.dispose()
      editor.value = null
    }
  })

  return {
    container,
    editor,
  }
}
