/**
 * Converts canonical Extended JSON (EJSON) values into human-readable equivalents.
 *
 * - { "$date": "..." }        → "..." (ISO string)
 * - { "$numberLong": "N" }    → N
 * - { "$numberInt": "N" }     → N
 * - { "$numberDouble": "N" }  → N
 * - Other objects/arrays       → recursed
 * - { "$oid": "..." }         → kept as-is (needed for queries)
 */
export function humanizeEjson(value: unknown): unknown {
  if (value === null || value === undefined) {
    return value
  }

  if (Array.isArray(value)) {
    return value.map(humanizeEjson)
  }

  if (typeof value === 'object') {
    const obj = value as Record<string, unknown>

    if ('$date' in obj && typeof obj.$date === 'string') {
      return obj.$date
    }

    if ('$numberLong' in obj && typeof obj.$numberLong === 'string') {
      return Number(obj.$numberLong)
    }

    if ('$numberInt' in obj && typeof obj.$numberInt === 'string') {
      return Number(obj.$numberInt)
    }

    if ('$numberDouble' in obj && typeof obj.$numberDouble === 'string') {
      const str = obj.$numberDouble
      if (str === 'Infinity') {
        return Infinity
      }
      if (str === '-Infinity') {
        return -Infinity
      }
      if (str === 'NaN') {
        return NaN
      }
      return Number(str)
    }

    const result: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(obj)) {
      result[k] = humanizeEjson(v)
    }
    return result
  }

  return value
}

/**
 * Reverses humanization: detects ISO date strings and wraps them back into { "$date": "..." }.
 * Plain numbers are left as-is since mongosh accepts them directly.
 */
export function dehumanizeEjson(value: unknown): unknown {
  if (value === null || value === undefined) {
    return value
  }

  if (Array.isArray(value)) {
    return value.map(dehumanizeEjson)
  }

  if (typeof value === 'string' && isIsoDateString(value)) {
    return { $date: value }
  }

  if (typeof value === 'object') {
    const obj = value as Record<string, unknown>
    const result: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(obj)) {
      result[k] = dehumanizeEjson(v)
    }
    return result
  }

  return value
}

const ISO_DATE_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/

function isIsoDateString(str: string): boolean {
  return ISO_DATE_RE.test(str)
}
