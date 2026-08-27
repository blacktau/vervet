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

/** Methods whose second argument is an update document. */
const UPDATE_METHODS = new Set(['updateOne', 'updateMany', 'findOneAndUpdate'])

interface Frame {
  open: '{' | '[' | '('
  /** For '{': the object key that owns it, e.g. `age` or `$match`. */
  key?: string
  /** For '(': the method being called, e.g. `find`. */
  callee?: string
  /** Top-level commas seen inside this frame — the argument/element index. */
  commas: number
}

interface Scan {
  stack: Frame[]
  /** Partial word at the caret, or the string contents when inside quotes. */
  prefix: string
  insideQuotes: boolean
  /** The caret sits in a value position: the last token was a `:`. */
  afterColon: boolean
}

/**
 * Walks a statement to the caret, tracking the bracket stack.
 *
 * Which list to suggest depends on nesting — `{ $match: { |` wants field names
 * while `{ age: { |` wants query operators, and they differ only in what owns
 * the enclosing brace. Ordered regexes can't see that, which is how nested
 * operator objects came to kill completions for the rest of the document.
 */
function scan(text: string): Scan {
  const stack: Frame[] = []
  let quote: string | null = null
  let quoteStart = 0
  let lastIdent = ''
  let afterColon = false
  let word = ''

  const top = () => stack[stack.length - 1]

  for (let i = 0; i < text.length; i++) {
    const ch = text[i]

    if (quote) {
      if (ch === '\\') {
        i++
      } else if (ch === quote) {
        lastIdent = text.slice(quoteStart, i)
        quote = null
      }
      continue
    }

    if (ch === '"' || ch === "'" || ch === '`') {
      quote = ch
      quoteStart = i + 1
      word = ''
      continue
    }

    if (/[\w$.]/.test(ch)) {
      word += ch
      continue
    }

    if (word) {
      lastIdent = word
      word = ''
    }

    if (ch === '{' || ch === '[') {
      stack.push({ open: ch, key: afterColon ? lastIdent : undefined, commas: 0 })
      afterColon = false
      lastIdent = ''
    } else if (ch === '(') {
      stack.push({ open: ch, callee: lastIdent.split('.').pop(), commas: 0 })
      afterColon = false
      lastIdent = ''
    } else if (ch === '}' || ch === ']' || ch === ')') {
      stack.pop()
      afterColon = false
      lastIdent = ''
    } else if (ch === ',') {
      const frame = top()
      if (frame) {
        frame.commas++
      }
      afterColon = false
      lastIdent = ''
    } else if (ch === ':') {
      afterColon = true
    }
  }

  return {
    stack,
    prefix: quote ? text.slice(quoteStart) : word,
    insideQuotes: quote !== null,
    afterColon,
  }
}

/** The innermost call the caret sits inside, e.g. `find` in `db.x.find({ |`. */
function enclosingCall(stack: Frame[]): Frame | undefined {
  for (let i = stack.length - 1; i >= 0; i--) {
    if (stack[i].open === '(') {
      return stack[i]
    }
  }
  return undefined
}

/**
 * The aggregation stage the caret is inside, e.g. `$match` — the first frame
 * below the pipeline array that is owned by a `$`-prefixed key.
 */
function enclosingStage(stack: Frame[]): string | undefined {
  return stack.find((f) => f.key?.startsWith('$'))?.key
}

function analyzeContextCore(textBeforeCursor: string): CompletionContext {
  // use <database> — checked before trimming so a trailing space is preserved
  const useMatch = textBeforeCursor.match(/(?:^|\n)\s*use\s+(\w*)$/)
  if (useMatch) {
    return { type: 'USE_DATABASE', prefix: useMatch[1] || '' }
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

  const braceContext = analyzeBraces(textBeforeCursor)
  if (braceContext) {
    return braceContext
  }

  // EJSON.| → EJSON_METHOD
  const ejsonMatch = trimmed.match(/EJSON\.(\w*)$/)
  if (ejsonMatch) {
    return { type: 'EJSON_METHOD', prefix: ejsonMatch[1] || '' }
  }

  // db.collection.method(...).| → CURSOR_METHOD (chained modifiers)
  const cursorMethodMatch = trimmed.match(/\)\s*\.(\w*)$/)
  if (cursorMethodMatch) {
    return { type: 'CURSOR_METHOD', prefix: cursorMethodMatch[1] || '' }
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
    return { type: 'COLLECTION_NAME', prefix: collMatch[1] || '' }
  }

  const lastWord = trimmed.match(/(\w*)$/)
  return { type: 'KEYWORD', prefix: lastWord ? lastWord[1] : '' }
}

/**
 * The context for a caret inside a `{` or `[`, decided from the bracket stack.
 * Returns undefined when the caret isn't inside one, leaving the caller's
 * simpler tail rules (method names, cursor methods, keywords) to handle it.
 */
function analyzeBraces(text: string): CompletionContext | undefined {
  const { stack, prefix, insideQuotes, afterColon } = scan(text)
  const frame = stack[stack.length - 1]
  if (!frame || frame.open === '(') {
    return undefined
  }

  const call = enclosingCall(stack)
  const collection = text.match(/\bdb\.(\w+)\b/)?.[1]
  const inAggregate = stack.some((f) => f.open === '(' && f.callee === 'aggregate')
  const inUpdateDoc = !!call && UPDATE_METHODS.has(call.callee ?? '') && call.commas >= 1
  const stage = inAggregate ? enclosingStage(stack) : undefined

  // $match takes a query, not an aggregation expression, and $sort's values are
  // just 1/-1 — offering expressions in either place suggests the wrong list.
  const operatorContext = (): CompletionContextType => {
    if (!inAggregate || stage === '$match') {
      return 'QUERY_OPERATOR'
    }
    if (stage === '$sort') {
      return 'KEYWORD'
    }
    return 'AGG_EXPRESSION'
  }

  const field = (): CompletionContext => ({
    type: 'FIELD_NAME',
    collection,
    prefix,
    insideQuotes,
  })

  // A value position: `{ age: |`. Inside quotes it's a literal, and suggesting
  // operators there would insert `$gt` into the user's string.
  if (afterColon) {
    if (insideQuotes) {
      return { type: 'KEYWORD', prefix }
    }
    const type = operatorContext()
    return type === 'KEYWORD' ? { type, prefix } : { type, collection, prefix }
  }

  if (frame.open === '[') {
    return inAggregate ? { type: 'AGG_STAGE', collection, prefix } : field()
  }

  // A brace with no owning key: a call argument, or an array element.
  if (!frame.key) {
    if (inAggregate && stack[stack.length - 2]?.open === '[') {
      return { type: 'AGG_STAGE', collection, prefix }
    }
    if (inUpdateDoc) {
      return { type: 'UPDATE_OPERATOR', collection, prefix }
    }
    return field()
  }

  // Owned by an operator or stage name — `{ $match: { |`, `{ $set: { |`,
  // `{ tags: { $elemMatch: { |` — whose keys are all field paths.
  if (frame.key.startsWith('$')) {
    return field()
  }

  // Owned by a plain field — `{ age: { |` — whose keys are operators.
  const type = operatorContext()
  return type === 'KEYWORD' ? { type, prefix } : { type, collection, prefix }
}
