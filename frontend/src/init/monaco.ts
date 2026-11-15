import * as monaco from 'monaco-editor'
import { openInBrowser } from '@/utils/openInBrowser'
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import TsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import { Uri } from 'monaco-editor'

export const initMonaco = () => {
  window.MonacoEnvironment = {
    getWorker: (workerId, label) => {
        switch (label) {
          case 'json':
            return new JsonWorker()
          case 'typescript':
          case 'javascript':
            return new TsWorker()
          default:
            return new EditorWorker()
        }
    },
  }

  // Disable default JavaScript language features so our custom MongoDB
  // completion provider is the only source of suggestions and hovers.
  monaco.typescript.javascriptDefaults.setCompilerOptions({
    noLib: true,
    allowNonTsExtensions: true,
  })
  monaco.typescript.javascriptDefaults.setDiagnosticsOptions({
    noSemanticValidation: true,
    noSyntaxValidation: true,
  })
  monaco.typescript.javascriptDefaults.setExtraLibs([])
  monaco.typescript.javascriptDefaults.setModeConfiguration({
    completionItems: false,
    hovers: false,
    documentSymbols: false,
    definitions: false,
    references: false,
    documentHighlights: false,
    rename: false,
    diagnostics: false,
    documentRangeFormattingEdits: false,
    signatureHelp: false,
    onTypeFormattingEdits: false,
    codeActions: false,
    inlayHints: false,
  })
}

// Colour escapes apart from the rest of the string, so `\t` in 'C:\temp'
// stands out as a tab rather than reading as part of the path (issue #369).
monaco.editor.defineTheme('vervet-light', {
  base: 'vs',
  inherit: true,
  rules: [
    { token: 'string.escape', foreground: '0086B3' },
    { token: 'string.escape.invalid', foreground: 'D600D6', fontStyle: 'bold underline' },
  ],
  colors: {}
})

monaco.editor.defineTheme('vervet-dark', {
  base: 'vs-dark',
  inherit: true,
  rules: [
    { token: 'string.escape', foreground: '569CD6' },
    { token: 'string.escape.invalid', foreground: 'C586C0', fontStyle: 'bold underline' },
  ],
  colors: {}
})

monaco.editor.registerLinkOpener({
  open(resource: Uri): boolean | Promise<boolean> {
    void openInBrowser(resource.toString())
    return true;
  },
})
