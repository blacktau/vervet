export function resolveRawValue(documents: unknown[], rowKey: string): unknown {
  const match = rowKey.match(/^__doc_(\d+)(.*)$/)
  if (!match) {
    return undefined
  }

  const docIndex = parseInt(match[1], 10)
  const doc = documents[docIndex]
  if (doc === undefined) {
    return undefined
  }

  const rest = match[2]
  if (!rest) {
    return doc
  }

  // rest starts with '.', split into path segments
  const segments = rest.slice(1).split('.')
  let current: unknown = doc
  for (const segment of segments) {
    if (current === null || current === undefined || typeof current !== 'object') {
      return undefined
    }
    current = (current as Record<string, unknown>)[segment]
  }

  return current
}
