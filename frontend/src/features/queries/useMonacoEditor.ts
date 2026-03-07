import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { useSettingsStore } from '@/features/settings/settingsStore'
import * as monaco from 'monaco-editor'
import { registerMongoCompletions } from './useMonacoCompletions'

interface MonacoEditorOptions {
  language: string
  value?: string
  readOnly?: boolean
  fontSize?: number
  extraOptions?: monaco.editor.IStandaloneEditorConstructionOptions
  queryId?: string
}

export function useMonacoEditor(options: MonacoEditorOptions) {
  const settingsStore = useSettingsStore()
  const container = ref<HTMLElement | null>(null)
  const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
  let completionDisposable: monaco.IDisposable | null = null

  function init() {
    if (!container.value) {
      return
    }

    const queryEditorOptions: monaco.editor.IStandaloneEditorConstructionOptions =
      options.queryId
        ? {
            wordBasedSuggestions: 'off',
            quickSuggestions: {
              other: true,
              comments: false,
              strings: true,
            },
            suggestOnTriggerCharacters: true,
            snippetSuggestions: 'none',
          }
        : {}

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
      ...queryEditorOptions,
      ...options.extraOptions,
    })

    if (options.queryId && options.language === 'javascript') {
      completionDisposable = registerMongoCompletions(options.queryId)
    }
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
    if (completionDisposable) {
      completionDisposable.dispose()
      completionDisposable = null
    }
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
