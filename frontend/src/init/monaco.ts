import * as monaco from 'monaco-editor'
import { BrowserOpenURL } from '../../wailsjs/runtime'
import JsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import { Uri } from 'monaco-editor'

export const initMonaco = () => {
  window.MonacoEnvironment = {
    getWorker: (workerId, label) => {
        switch (label) {
          case 'json':
            return new JsonWorker()
          default:
            return new EditorWorker()
        }
    },
  }
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
