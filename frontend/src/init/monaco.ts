import * as monaco from 'monaco-editor'
import { BrowserOpenURL } from '../../wailsjs/runtime'
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
  monaco.languages.typescript.javascriptDefaults.setCompilerOptions({
    noLib: true,
    allowNonTsExtensions: true,
  })
  monaco.languages.typescript.javascriptDefaults.setDiagnosticsOptions({
    noSemanticValidation: true,
    noSyntaxValidation: true,
  })
  monaco.languages.typescript.javascriptDefaults.setExtraLibs([])
  monaco.languages.typescript.javascriptDefaults.setModeConfiguration({
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

monaco.editor.defineTheme('vervet-light', {
  base: 'vs',
  inherit: true,
  rules: [],
  colors: {}
})

monaco.editor.defineTheme('vervet-dark', {
  base: 'vs-dark',
  inherit: true,
  rules: [],
  colors: {}
})

monaco.editor.registerLinkOpener({
  open(resource: Uri): boolean | Promise<boolean> {
    BrowserOpenURL(resource.toString())
    return true;
  },
})
