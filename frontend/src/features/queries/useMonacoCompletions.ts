import * as monaco from 'monaco-editor'
import { analyzeContext } from './completionContext'
import { mongoMethods, queryOperators, aggStages } from './completionData'
import { getCollectionSchema, getCollectionNames } from './useSchemaCache'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from './queryStore'
import type { CompletionContext } from './completionContext'

function toCompletionItems(
  items: { label: string; detail: string }[],
  range: monaco.IRange,
  kind: monaco.languages.CompletionItemKind,
): monaco.languages.CompletionItem[] {
  return items.map((item) => ({
    label: item.label,
    kind,
    detail: item.detail,
    insertText: item.label,
    range,
  }))
}

function fieldInfoToCompletions(
  fields: { path: string; types: string[]; children?: unknown[] }[],
  range: monaco.IRange,
): monaco.languages.CompletionItem[] {
  return fields.map((field) => ({
    label: field.path,
    kind: monaco.languages.CompletionItemKind.Field,
    detail: field.types.join(' | '),
    insertText: field.path,
    range,
  }))
}

export function registerMongoCompletions(queryId: string): monaco.IDisposable {
  return monaco.languages.registerCompletionItemProvider('javascript', {
    triggerCharacters: ['.', '{', '[', ' ', ','],

    async provideCompletionItems(
      model: monaco.editor.ITextModel,
      position: monaco.Position,
    ): Promise<monaco.languages.CompletionList> {
      const textBeforeCursor = model.getValueInRange({
        startLineNumber: 1,
        startColumn: 1,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      })

      const ctx = analyzeContext(textBeforeCursor)

      const word = model.getWordUntilPosition(position)
      const range: monaco.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      }

      const suggestions = await getSuggestions(ctx, range, queryId)
      return { suggestions }
    },
  })
}

async function getSuggestions(
  ctx: CompletionContext,
  range: monaco.IRange,
  queryId: string,
): Promise<monaco.languages.CompletionItem[]> {
  const tabStore = useTabStore()
  const queryStore = useQueryStore()
  const serverId = tabStore.currentTabId
  const state = queryStore.getQueryState(queryId)
  const dbName = state?.selectedDatabase

  switch (ctx.type) {
    case 'COLLECTION_NAME': {
      if (!serverId || !dbName) {
        return []
      }
      const names = await getCollectionNames(serverId, dbName)
      return names
        .filter((n) => n.startsWith(ctx.prefix))
        .map((name) => ({
          label: name,
          kind: monaco.languages.CompletionItemKind.Module,
          insertText: name,
          range,
        }))
    }

    case 'METHOD_NAME':
      return toCompletionItems(
        mongoMethods.filter((m) => m.label.startsWith(ctx.prefix)),
        range,
        monaco.languages.CompletionItemKind.Method,
      )

    case 'FIELD_NAME': {
      if (!serverId || !dbName || !ctx.collection) {
        return []
      }
      const schema = await getCollectionSchema(serverId, dbName, ctx.collection)
      if (!schema) {
        return []
      }
      return fieldInfoToCompletions(
        schema.fields.filter((f) => f.path.startsWith(ctx.prefix)),
        range,
      )
    }

    case 'QUERY_OPERATOR':
      return toCompletionItems(queryOperators, range, monaco.languages.CompletionItemKind.Operator)

    case 'AGG_STAGE':
      return toCompletionItems(aggStages, range, monaco.languages.CompletionItemKind.Keyword)

    case 'KEYWORD':
      return [
        {
          label: 'db',
          kind: monaco.languages.CompletionItemKind.Variable,
          detail: 'Database object',
          insertText: 'db',
          range,
        },
      ]

    default:
      return []
  }
}
