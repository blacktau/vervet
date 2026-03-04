export function getDocPreview(doc: unknown): string {
  if (typeof doc !== 'object' || doc === null) {
    return String(doc)
  }

  const parts: string[] = []
  for (const [key, val] of Object.entries(doc)) {
    if (key === '_id') {
      continue
    }
    if (parts.length >= 3) {
      break
    }
    if (val === null || typeof val === 'string' || typeof val === 'number' || typeof val === 'boolean') {
      parts.push(`${key}: ${JSON.stringify(val)}`)
    }
  }
  return parts.join(', ')
}

export function getDocId(doc: unknown): string {
  if (typeof doc !== 'object' || doc === null) {
    return ''
  }
  const obj = doc as Record<string, unknown>
  const id = obj._id
  if (id === undefined) {
    return ''
  }
  if (typeof id === 'object' && id !== null && '$oid' in id) {
    return String((id as Record<string, unknown>).$oid)
  }
  return String(id)
}
