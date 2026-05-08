import { ref, onMounted, onBeforeUnmount, watch, shallowRef } from 'vue'
import { useSettingsStore } from '@/features/settings/settingsStore'
import * as monaco from 'monaco-editor'
import { registerMongoCompletions } from '@/features/completion/useMonacoCompletions'

interface MonacoEditorOptions {
  language: string
  value?: string
  readOnly?: boolean
  fontSize?: number
  extraOptions?: monaco.editor.IStandaloneEditorConstructionOptions
  queryId?: string
}

const VALIDATION_DEBOUNCE_MS = 250
const MARKER_OWNER = 'vervet-syntax'

let validatorPromise: Promise<typeof import('./syntaxValidator')> | null = null
function ensureValidator() {
  return (validatorPromise ??= import('./syntaxValidator'))
}

export function useMonacoEditor(options: MonacoEditorOptions) {
  const settingsStore = useSettingsStore()
  const container = ref<HTMLElement | null>(null)
  const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
  let completionDisposable: monaco.IDisposable | null = null
  let contentDisposable: monaco.IDisposable | null = null
  let validatorTimer: ReturnType<typeof setTimeout> | null = null

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
      fontSize: settingsStore.editor.font.size || options.fontSize || 14,
      fontFamily: settingsStore.editor.font.family || undefined,
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      padding: { top: 8 },
      readOnly: options.readOnly ?? false,
      ...queryEditorOptions,
      ...options.extraOptions,
    })

    if (options.queryId && options.language === 'javascript') {
      completionDisposable = registerMongoCompletions(options.queryId, editor.value)
      attachSyntaxValidation(editor.value)
    }
  }

  function attachSyntaxValidation(ed: monaco.editor.IStandaloneCodeEditor) {
    const runValidation = async () => {
      const model = ed.getModel()
      if (!model) {
        return
      }
      const source = model.getValue()
      const { validate } = await ensureValidator()
      // Re-check the model is still alive after the async hop.
      if (model.isDisposed()) {
        return
      }
      const markers = validate(source)
      monaco.editor.setModelMarkers(model, MARKER_OWNER, markers)
    }

    contentDisposable = ed.onDidChangeModelContent(() => {
      if (validatorTimer) {
        clearTimeout(validatorTimer)
      }
      validatorTimer = setTimeout(runValidation, VALIDATION_DEBOUNCE_MS)
    })

    // Initial pass (covers the value passed via options.value).
    runValidation()
  }

  watch(
    () => settingsStore.isDark,
    (isDark) => {
      if (editor.value) {
        monaco.editor.setTheme(isDark ? 'vervet-dark' : 'vervet-light')
      }
    },
  )

  watch(
    () => [settingsStore.editor.font.family, settingsStore.editor.font.size],
    ([family, size]) => {
      if (editor.value) {
        editor.value.updateOptions({
          fontFamily: (family as string) || undefined,
          fontSize: (size as number) || 14,
        })
      }
    },
  )

  onMounted(() => {
    init()
  })

  onBeforeUnmount(() => {
    if (validatorTimer) {
      clearTimeout(validatorTimer)
      validatorTimer = null
    }
    if (contentDisposable) {
      contentDisposable.dispose()
      contentDisposable = null
    }
    if (completionDisposable) {
      completionDisposable.dispose()
      completionDisposable = null
    }
    if (editor.value) {
      const model = editor.value.getModel()
      if (model) {
        monaco.editor.setModelMarkers(model, MARKER_OWNER, [])
      }
      editor.value.dispose()
      editor.value = null
    }
  })

  return {
    container,
    editor,
  }
}
