export type CompletionContextType =
  | 'COLLECTION_NAME'
  | 'COLLECTION_NAME_STRING'
  | 'METHOD_NAME'
  | 'CURSOR_METHOD'
  | 'FIELD_NAME'
  | 'QUERY_OPERATOR'
  | 'AGG_STAGE'
  | 'KEYWORD'
  | 'UPDATE_OPERATOR'
  | 'AGG_EXPRESSION'
  | 'USE_DATABASE'

export interface CompletionContext {
  type: CompletionContextType
  collection?: string
  prefix: string
  /** Whether the cursor is already inside quotes (no need to add them) */
  insideQuotes?: boolean
}

export function analyzeContext(textBeforeCursor: string): CompletionContext {
  // Normalise db.getCollection('name') → db.__gc__ so all regexes below work
  // unchanged, then restore the real collection name in the result.
  const gcPlaceholder = '__gc__'
  let realCollectionName: string | undefined
  const gcMatch = textBeforeCursor.match(/db\.getCollection\(\s*['"]([^'"]*)['"]\s*\)/)
  if (gcMatch) {
    realCollectionName = gcMatch[1]
    textBeforeCursor = textBeforeCursor.replace(
      /db\.getCollection\(\s*['"][^'"]*['"]\s*\)/,
      `db.${gcPlaceholder}`,
    )
  }

  const ctx = analyzeContextCore(textBeforeCursor)

  if (realCollectionName != null && ctx.collection === gcPlaceholder) {
    ctx.collection = realCollectionName
  }

  return ctx
}

function analyzeContextCore(textBeforeCursor: string): CompletionContext {
  // use <database> — check before trimming so trailing space is preserved
  const useMatch = textBeforeCursor.match(/(?:^|\n)\s*use\s+(\w*)$/)
  if (useMatch) {
    return {
      type: 'USE_DATABASE',
      prefix: useMatch[1] || '',
    }
  }

  const trimmed = textBeforeCursor.trimEnd()

  // db.getCollection('| or db.getCollection("| → COLLECTION_NAME_STRING
  const getCollMatch = trimmed.match(/db\.getCollection\(\s*(['"])(\w*)$/)
  if (getCollMatch) {
    return {
      type: 'COLLECTION_NAME_STRING',
      prefix: getCollMatch[2] || '',
      insideQuotes: true,
    }
  }

  // db.collection.updateOne/Many/findOneAndUpdate({  }, {  })
  const updateMatch = trimmed.match(
    /db\.(\w+)\.(?:updateOne|updateMany|findOneAndUpdate)\([\s\S]*,\s*\{\s*(\$\w*)?$/,
  )
  if (updateMatch) {
    return {
      type: 'UPDATE_OPERATOR',
      collection: updateMatch[1],
      prefix: updateMatch[2] || '',
    }
  }

  // Inside an aggregate stage's value object: { $group: { total: { $sum| or { $project: { x: { $
  // Must be checked before QUERY_OPERATOR since those regexes would also match inside aggregate.
  // The key insight: we need at least two nested { after .aggregate( — one for the stage, one for the field value.
  // We match the last { that isn't closed, and check for a $ prefix there.
  const aggExpressionMatch = trimmed.match(
    /db\.(\w+)\.aggregate\([\s\S]*\$\w+\s*:\s*\{[^}]*\{\s*(\$\w*)?$/,
  )
  if (aggExpressionMatch) {
    return {
      type: 'AGG_EXPRESSION',
      collection: aggExpressionMatch[1],
      prefix: aggExpressionMatch[2] || '',
    }
  }

  // db.collection.method({ field: { $op| }) → QUERY_OPERATOR (inside nested operator object)
  const nestedOpMatch = trimmed.match(/db\.(\w+)\.\w+\([^)]*(?:\b\w+|"[^"]*")\s*:\s*\{\s*(\$\w*)?$/)
  if (nestedOpMatch) {
    return {
      type: 'QUERY_OPERATOR',
      collection: nestedOpMatch[1],
      prefix: nestedOpMatch[2] || '',
    }
  }

  // db.collection.method({ field: | }) → QUERY_OPERATOR
  // Also matches quoted field keys: { "field.name": | }
  const fieldValueMatch = trimmed.match(/db\.(\w+)\.\w+\(\s*\{[^}]*(?:\b\w+|"[^"]*")\s*:\s*$/)
  if (fieldValueMatch) {
    return {
      type: 'QUERY_OPERATOR',
      collection: fieldValueMatch[1],
      prefix: '',
    }
  }

  // db.collection.aggregate([ | ]) → AGG_STAGE (empty pipeline)
  const aggEmptyMatch = trimmed.match(/db\.(\w+)\.aggregate\(\s*\[\s*$/)
  if (aggEmptyMatch) {
    return { type: 'AGG_STAGE', collection: aggEmptyMatch[1], prefix: '' }
  }

  // db.collection.aggregate([{...}, | ]) → AGG_STAGE (after existing stages)
  const aggAfterStageMatch = trimmed.match(/db\.(\w+)\.aggregate\([\s\S]*,\s*$/)
  if (aggAfterStageMatch) {
    return { type: 'AGG_STAGE', collection: aggAfterStageMatch[1], prefix: '' }
  }

  // Inside braces for field name position: { "partial| or { partial|
  // Matches both quoted and unquoted field name positions
  const quotedFieldMatch = trimmed.match(
    /db\.(\w+)\.\w+\([^)]*\{\s*(?:[\w."':$\s,]*,\s*)?"([^"]*)$/,
  )
  if (quotedFieldMatch) {
    return {
      type: 'FIELD_NAME',
      collection: quotedFieldMatch[1],
      prefix: quotedFieldMatch[2] || '',
      insideQuotes: true,
    }
  }

  const insideBracesMatch = trimmed.match(/db\.(\w+)\.\w+\([^)]*\{\s*(?:[\w."':$\s,]*,\s*)?(\w*)$/)
  if (insideBracesMatch) {
    return {
      type: 'FIELD_NAME',
      collection: insideBracesMatch[1],
      prefix: insideBracesMatch[2] || '',
      insideQuotes: false,
    }
  }

  // db.collection.method(...).| → CURSOR_METHOD (chained modifiers)
  // Matches after a closing paren: .find({}).| or .find({}).lim|
  // Also matches chained: .find({}).sort({}).| or .find({}).limit(10).|
  const cursorMethodMatch = trimmed.match(/\)\s*\.(\w*)$/)
  if (cursorMethodMatch) {
    return {
      type: 'CURSOR_METHOD',
      prefix: cursorMethodMatch[1] || '',
    }
  }

  // db.collection.| → METHOD_NAME
  const methodMatch = trimmed.match(/db\.(\w+)\.(\w*)$/)
  if (methodMatch) {
    return {
      type: 'METHOD_NAME',
      collection: methodMatch[1],
      prefix: methodMatch[2] || '',
    }
  }

  // db.| → COLLECTION_NAME
  const collMatch = trimmed.match(/db\.(\w*)$/)
  if (collMatch) {
    return {
      type: 'COLLECTION_NAME',
      prefix: collMatch[1] || '',
    }
  }

  // Default: keyword
  const lastWord = trimmed.match(/(\w*)$/)
  return {
    type: 'KEYWORD',
    prefix: lastWord ? lastWord[1] : '',
  }
}
