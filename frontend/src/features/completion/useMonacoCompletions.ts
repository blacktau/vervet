import * as monaco from 'monaco-editor'
import { analyzeContext } from './completionContext'
import {
  mongoMethods,
  cursorMethods,
  queryOperators,
  aggStages,
  updateOperators,
  aggExpressions,
} from './completionData'
import { getCollectionSchema, getCollectionNames } from './useSchemaCache'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import type { CompletionContext } from './completionContext'

interface FieldInfo {
  path: string
  types: string[]
  children?: FieldInfo[]
}

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

/**
 * Flattens a schema field tree into dotted-path entries.
 * e.g. { path: "address", children: [{ path: "street" }, { path: "country" }] }
 * becomes: ["address", "address.street", "address.country"]
 */
function flattenFields(
  fields: FieldInfo[],
  parentPath: string = '',
): { path: string; types: string[]; hasChildren: boolean }[] {
  const result: { path: string; types: string[]; hasChildren: boolean }[] = []
  for (const field of fields) {
    const fullPath = parentPath ? `${parentPath}.${field.path}` : field.path
    const hasChildren = (field.children?.length ?? 0) > 0
    result.push({ path: fullPath, types: field.types, hasChildren })
    if (field.children) {
      result.push(...flattenFields(field.children, fullPath))
    }
  }
  return result
}

function fieldCompletions(
  fields: FieldInfo[],
  prefix: string,
  range: monaco.IRange,
  insideQuotes: boolean,
): monaco.languages.CompletionItem[] {
  const flat = flattenFields(fields)
  const matching = flat.filter((f) => f.path.startsWith(prefix))

  return matching.map((field) => ({
    label: field.path,
    kind: field.hasChildren
      ? monaco.languages.CompletionItemKind.Struct
      : monaco.languages.CompletionItemKind.Field,
    detail: field.types.join(' | '),
    insertText: insideQuotes ? field.path : `"${field.path}": `,
    range,
  }))
}

export function registerMongoCompletions(queryId: string): monaco.IDisposable {
  return monaco.languages.registerCompletionItemProvider('javascript', {
    triggerCharacters: ['.', '{', '[', ' ', ',', '"', "'", '$'],

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
      let range: monaco.IRange = {
        startLineNumber: position.lineNumber,
        endLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
      }

      // For field names inside quotes, extend the range to cover the full dotted prefix
      // so the entire typed path gets replaced by the completion
      if (ctx.type === 'FIELD_NAME' && ctx.insideQuotes && ctx.prefix.includes('.')) {
        const lineText = model.getLineContent(position.lineNumber)
        // Find the opening quote before the cursor
        const textBeforeOnLine = lineText.substring(0, position.column - 1)
        const lastQuote = Math.max(
          textBeforeOnLine.lastIndexOf('"'),
          textBeforeOnLine.lastIndexOf("'"),
        )
        if (lastQuote >= 0) {
          range = {
            startLineNumber: position.lineNumber,
            endLineNumber: position.lineNumber,
            startColumn: lastQuote + 2, // after the quote character (1-indexed)
            endColumn: position.column,
          }
        }
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

    case 'COLLECTION_NAME_STRING': {
      if (!serverId || !dbName) {
        return []
      }
      const names = await getCollectionNames(serverId, dbName)
      return names
        .filter((n) => n.startsWith(ctx.prefix))
        .map((name) => ({
          label: name,
          kind: monaco.languages.CompletionItemKind.Module,
          detail: 'Collection',
          insertText: name,
          range,
        }))
    }

    case 'METHOD_NAME':
      return mongoMethods
        .filter((m) => m.label.startsWith(ctx.prefix))
        .map((m) => ({
          label: m.label,
          kind: monaco.languages.CompletionItemKind.Method,
          detail: m.detail,
          insertText: m.snippet,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range,
        }))

    case 'CURSOR_METHOD':
      return cursorMethods
        .filter((m) => m.label.startsWith(ctx.prefix))
        .map((m) => ({
          label: m.label,
          kind: monaco.languages.CompletionItemKind.Method,
          detail: m.detail,
          insertText: m.snippet,
          insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
          range,
        }))

    case 'FIELD_NAME': {
      if (!serverId || !dbName || !ctx.collection) {
        return []
      }
      const schema = await getCollectionSchema(serverId, dbName, ctx.collection)
      if (!schema) {
        return []
      }
      return fieldCompletions(schema.fields, ctx.prefix, range, ctx.insideQuotes ?? false)
    }

    case 'QUERY_OPERATOR':
      return toCompletionItems(queryOperators, range, monaco.languages.CompletionItemKind.Operator)

    case 'AGG_STAGE':
      return toCompletionItems(aggStages, range, monaco.languages.CompletionItemKind.Keyword)

    case 'AGG_EXPRESSION':
      return toCompletionItems(aggExpressions, range, monaco.languages.CompletionItemKind.Function)

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

    case 'UPDATE_OPERATOR':
      return toCompletionItems(updateOperators, range, monaco.languages.CompletionItemKind.Operator)

    default:
      return []
  }
}
