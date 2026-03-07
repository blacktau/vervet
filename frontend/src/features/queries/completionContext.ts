export type CompletionContextType =
  | 'COLLECTION_NAME'
  | 'METHOD_NAME'
  | 'FIELD_NAME'
  | 'QUERY_OPERATOR'
  | 'AGG_STAGE'
  | 'KEYWORD'

export interface CompletionContext {
  type: CompletionContextType
  collection?: string
  prefix: string
}

export function analyzeContext(textBeforeCursor: string): CompletionContext {
  const trimmed = textBeforeCursor.trimEnd()

  // db.collection.method({ field: | }) → QUERY_OPERATOR
  const fieldValueMatch = trimmed.match(
    /db\.(\w+)\.\w+\(\s*\{[^}]*\b\w+\s*:\s*$/,
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

  // db.collection.find({ | }) or db.collection.find({}, { | }) → FIELD_NAME
  const insideBracesMatch = trimmed.match(
    /db\.(\w+)\.\w+\([^)]*\{\s*(?:[\w."':$\s,]*,\s*)?(\w*)$/,
  )
  if (insideBracesMatch) {
    return {
      type: 'FIELD_NAME',
      collection: insideBracesMatch[1],
      prefix: insideBracesMatch[2] || '',
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
