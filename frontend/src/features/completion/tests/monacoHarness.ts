// Headless Monaco harness for completion tests.
//
// Requires `// @vitest-environment happy-dom` in the test file. Imports the
// narrow editor.api entrypoint rather than the 'monaco-editor' barrel — the
// barrel loads every basic-language contribution asynchronously and those
// imports race Vitest's environment teardown.
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api'
// The real JavaScript language configuration, imported directly rather than via
// its *.contribution module: the contribution registers a lazy async loader that
// resolves after Vitest tears the environment down. Registering it matters —
// without it Monaco falls back to DEFAULT_WORD_REGEXP, which treats `$` as a
// word separator, and every $-operator assertion tests the wrong thing.
import { conf as javascriptConf } from 'monaco-editor/esm/vs/basic-languages/javascript/javascript'
import { provideMongoCompletions } from '../useMonacoCompletions'

let languageRegistered = false
function ensureJavascriptLanguage() {
  if (languageRegistered) {
    return
  }
  languageRegistered = true
  monaco.languages.register({ id: 'javascript' })
  monaco.languages.setLanguageConfiguration('javascript', javascriptConf)
}

/**
 * Builds a real Monaco text model from source with a `|` marking the caret,
 * and runs the completion provider at that position.
 *
 * completionsAt('db.users.fi|') → items for the METHOD_NAME context.
 */
export async function completionsAt(
  sourceWithCaret: string,
  queryId = 'query-1',
): Promise<monaco.languages.CompletionItem[]> {
  const { model, position } = modelWithCaret(sourceWithCaret)
  try {
    return await provideMongoCompletions(model, position, queryId)
  } finally {
    model.dispose()
  }
}

/** Labels only — the common assertion, and far easier to read on failure. */
export async function labelsAt(sourceWithCaret: string, queryId = 'query-1'): Promise<string[]> {
  const items = await completionsAt(sourceWithCaret, queryId)
  return items.map((i) => (typeof i.label === 'string' ? i.label : i.label.label))
}

export function modelWithCaret(sourceWithCaret: string): {
  model: monaco.editor.ITextModel
  position: monaco.IPosition
} {
  ensureJavascriptLanguage()
  const caret = sourceWithCaret.indexOf('|')
  if (caret < 0) {
    throw new Error(`no '|' caret marker in: ${sourceWithCaret}`)
  }
  const source = sourceWithCaret.slice(0, caret) + sourceWithCaret.slice(caret + 1)
  const before = sourceWithCaret.slice(0, caret).split('\n')
  const position = {
    lineNumber: before.length,
    column: (before[before.length - 1]?.length ?? 0) + 1,
  }
  return { model: monaco.editor.createModel(source, 'javascript'), position }
}

/** The text the suggest widget filters against: the model text under `range`. */
export function filterTextFor(
  sourceWithCaret: string,
  item: monaco.languages.CompletionItem,
): string {
  const { model } = modelWithCaret(sourceWithCaret)
  try {
    return model.getValueInRange(item.range as monaco.IRange)
  } finally {
    model.dispose()
  }
}
