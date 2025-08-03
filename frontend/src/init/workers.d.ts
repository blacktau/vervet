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
