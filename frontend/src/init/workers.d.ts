declare module 'monaco-editor/esm/vs/editor/editor.worker?worker' {
  export default class EditorWorker extends Worker {
    constructor();
  }
}

declare module 'monaco-editor/esm/vs/language/json/json.worker?worker' {
  export default class JsonWorker extends Worker {
    constructor();
  }
}

declare module 'monaco-editor/esm/vs/language/' {
  export default class CssWorker extends Worker {
    constructor();
  }
}

// monaco-editor ships no declarations for this file.
declare module 'monaco-editor/esm/vs/basic-languages/javascript/javascript' {
  import type { languages } from 'monaco-editor'
  export const conf: languages.LanguageConfiguration
}
