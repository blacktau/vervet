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
  | 'EJSON_METHOD'

export interface CompletionContext {
  type: CompletionContextType
  collection?: string
  prefix: string
  /** Whether the cursor is already inside quotes (no need to add them) */
  insideQuotes?: boolean
}

/**
 * True when the next non-blank character after `i` continues the current
 * statement rather than starting a new one — leading-dot method chaining:
 *
 *   db.users
 *     .find({})
 *     .|
 */
function continuesOnNextLine(text: string, i: number): boolean {
  const rest = text.slice(i + 1)
  return /^\s*\./.test(rest)
}

/**
 * Trims everything before the statement the caret sits in.
 *
 * The rules below scan backwards with `[\s\S]*`, so without this an earlier
 * statement's `aggregate(`/`updateOne(` swallows the text down to the caret and
 * every later query is analysed as that statement's continuation. Scans forward
 * tracking bracket depth, string state and line comments; a `;` or newline at
 * depth 0 starts a new statement, so multi-line statements stay intact.
 *
 * ponytail: a depth scan, not a JS parser — enough for statement boundaries.
 */
function currentStatement(text: string): string {
  let depth = 0
  let start = 0
  let quote: string | null = null
  let inComment = false
  let out = ''

  for (let i = 0; i < text.length; i++) {
    const ch = text[i]

    if (inComment) {
      // Blank the comment out so the caret inside one sees no code.
      out += ch === '\n' ? ch : ' '
      if (ch === '\n') {
        inComment = false
        if (depth === 0 && !continuesOnNextLine(text, i)) {
          start = out.length
        }
      }
      continue
    }

    out += ch

    if (quote) {
      if (ch === '\\') {
        out += text[++i] ?? ''
      } else if (ch === quote) {
        quote = null
      } else if (ch === '\n' && quote !== '`') {
        // An unterminated quote: recover rather than treating the rest as string.
        quote = null
        if (depth === 0 && !continuesOnNextLine(text, i)) {
          start = out.length
        }
      }
      continue
    }

    if (ch === '/' && text[i + 1] === '/') {
      inComment = true
      out = out.slice(0, -1) + ' '
      continue
    }

    if (ch === '"' || ch === "'" || ch === '`') {
      quote = ch
    } else if (ch === '(' || ch === '[' || ch === '{') {
      depth++
    } else if (ch === ')' || ch === ']' || ch === '}') {
      depth = Math.max(0, depth - 1)
    } else if ((ch === ';' || ch === '\n') && depth === 0 && !continuesOnNextLine(text, i)) {
      start = out.length
    }
  }

  return out.slice(start)
}

export function analyzeContext(textBeforeCursor: string): CompletionContext {
  // Normalise every db.getCollection('name') → db.__gc0__, db.__gc1__, ... so all
  // regexes below work unchanged, then restore the real collection name in the
  // result. Every occurrence must be rewritten: leaving a later one intact makes
  // it read as a chained call and mis-detects the context (issue #297).
  const collectionNames: string[] = []
  const normalised = textBeforeCursor.replace(
    /db\.getCollection\(\s*['"]([^'"]*)['"]\s*\)/g,
    (_match, name: string) => `db.__gc${collectionNames.push(name) - 1}__`,
  )

  const ctx = analyzeContextCore(currentStatement(normalised))

  const placeholder = ctx.collection?.match(/^__gc(\d+)__$/)
  if (placeholder) {
    ctx.collection = collectionNames[Number(placeholder[1])]
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
  const nestedOpMatch = trimmed.match(
    /db\.(\w+)(?:\.\w+\([^()]*\))*\.\w+\([^)]*(?:\b\w+|"[^"]*")\s*:\s*\{\s*(\$\w*)?$/,
  )
  if (nestedOpMatch) {
    return {
      type: 'QUERY_OPERATOR',
      collection: nestedOpMatch[1],
      prefix: nestedOpMatch[2] || '',
    }
  }

  // db.collection.method({ field: | }) → QUERY_OPERATOR
  // Also matches quoted field keys: { "field.name": | }
  const fieldValueMatch = trimmed.match(
    /db\.(\w+)(?:\.\w+\([^()]*\))*\.\w+\(\s*\{[^}]*(?:\b\w+|"[^"]*")\s*:\s*$/,
  )
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

  // db.collection.aggregate([{ | or ([{ $ma| or ([{...}, { | → AGG_STAGE
  // The caret sits directly after a stage's opening brace. Anything typed past
  // that brace (a stage's own body) fails the anchor and falls through to the
  // field/operator rules below, and AGG_EXPRESSION is matched earlier.
  const aggStageOpenMatch = trimmed.match(/db\.(\w+)\.aggregate\([\s\S]*\{\s*(\$\w*)?$/)
  if (aggStageOpenMatch) {
    return {
      type: 'AGG_STAGE',
      collection: aggStageOpenMatch[1],
      prefix: aggStageOpenMatch[2] || '',
    }
  }

  // Inside braces for field name position: { "partial| or { partial|
  // Matches both quoted and unquoted field name positions
  const quotedFieldMatch = trimmed.match(
    /db\.(\w+)(?:\.\w+\([^()]*\))*\.\w+\([^)]*\{\s*(?:[\w."':$\s,]*,\s*)?"([^"]*)$/,
  )
  if (quotedFieldMatch) {
    return {
      type: 'FIELD_NAME',
      collection: quotedFieldMatch[1],
      prefix: quotedFieldMatch[2] || '',
      insideQuotes: true,
    }
  }

  const insideBracesMatch = trimmed.match(
    /db\.(\w+)(?:\.\w+\([^()]*\))*\.\w+\([^)]*\{\s*(?:[\w."':$\s,]*,\s*)?(\w*)$/,
  )
  if (insideBracesMatch) {
    return {
      type: 'FIELD_NAME',
      collection: insideBracesMatch[1],
      prefix: insideBracesMatch[2] || '',
      insideQuotes: false,
    }
  }

  // EJSON.| → EJSON_METHOD
  const ejsonMatch = trimmed.match(/EJSON\.(\w*)$/)
  if (ejsonMatch) {
    return {
      type: 'EJSON_METHOD',
      prefix: ejsonMatch[1] || '',
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
