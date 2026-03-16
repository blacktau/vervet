/**
 * Converts canonical Extended JSON (EJSON) values into human-readable equivalents,
 * matching the display logic in documentTreeUtils.getDisplayValue().
 *
 * - { "$oid": "..." }                        → "..." (plain hex string)
 * - { "$date": "..." }                       → "..." (ISO string)
 * - { "$date": { "$numberLong": "N" } }      → ISO string from epoch ms
 * - { "$numberLong": "N" }                   → N (number)
 * - { "$numberInt": "N" }                    → N (number)
 * - { "$numberDouble": "N" }                 → N (number)
 * - { "$numberDecimal": "N" }                → N (number, or string if not representable)
 * - { "$binary": { base64, subType: "04" } } → UUID string
 * - { "$binary": { base64, subType: "03" } } → UUID string (legacy)
 * - { "$binary": { base64, subType: "05" } } → hex string (MD5)
 * - { "$regularExpression": { pattern, options } } → "/pattern/options"
 * - { "$regex": "...", "$options": "..." }    → "/pattern/options"
 * - { "$timestamp": { t, i } }               → kept as-is (no clean string representation)
 * - { "$minKey": 1 }                         → kept as-is
 * - { "$maxKey": 1 }                         → kept as-is
 * - Other objects/arrays                      → recursed
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

    if ('$oid' in obj && typeof obj.$oid === 'string') {
      return obj.$oid
    }

    if ('$date' in obj) {
      const dateVal = obj.$date
      if (typeof dateVal === 'string') {
        return dateVal
      }
      if (typeof dateVal === 'object' && dateVal !== null && '$numberLong' in dateVal) {
        const ms = Number((dateVal as Record<string, unknown>).$numberLong)
        return new Date(ms).toISOString()
      }
      return String(dateVal)
    }

    if ('$numberLong' in obj && typeof obj.$numberLong === 'string') {
      return Number(obj.$numberLong)
    }

    if ('$numberInt' in obj && typeof obj.$numberInt === 'string') {
      return Number(obj.$numberInt)
    }

    if ('$numberDouble' in obj && typeof obj.$numberDouble === 'string') {
      return parseSpecialDouble(obj.$numberDouble)
    }

    if ('$numberDecimal' in obj && typeof obj.$numberDecimal === 'string') {
      const num = Number(obj.$numberDecimal)
      if (!isNaN(num) && isFinite(num) && String(num) === obj.$numberDecimal) {
        return num
      }
      return obj.$numberDecimal
    }

    if ('$binary' in obj) {
      const binary = obj.$binary as Record<string, unknown>
      if (typeof binary?.base64 === 'string') {
        if (binary.subType === '04' || binary.subType === '03') {
          return base64ToUUID(binary.base64)
        }
        if (binary.subType === '05') {
          return base64ToHex(binary.base64)
        }
      }
      // Non-UUID/MD5 binary: keep as-is
      return value
    }

    if ('$regularExpression' in obj) {
      const re = obj.$regularExpression as Record<string, unknown>
      return `/${re.pattern}/${re.options || ''}`
    }

    if ('$regex' in obj) {
      return `/${obj.$regex}/${obj.$options || ''}`
    }

    // $timestamp, $minKey, $maxKey: keep as EJSON (not cleanly representable as a primitive)
    const result: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(obj)) {
      result[k] = humanizeEjson(v)
    }
    return result
  }

  return value
}

/**
 * Converts a value (humanized or raw EJSON) into a JavaScript expression string
 * using the BSON constructor functions available in the Go query engine:
 * ObjectId(), ISODate(), UUID(), NumberLong(), NumberDecimal(), Timestamp(),
 * MinKey(), MaxKey(), BinData().
 *
 * Handles both:
 * - Raw EJSON values (e.g. { $oid: "..." } → ObjectId("..."))
 * - Humanized values (e.g. ISO date string → ISODate("..."))
 */
export function toJsExpression(value: unknown): string {
  if (value === null) {
    return 'null'
  }
  if (value === undefined) {
    return 'undefined'
  }

  if (Array.isArray(value)) {
    const items = value.map(toJsExpression)
    return `[${items.join(', ')}]`
  }

  if (typeof value === 'object') {
    const obj = value as Record<string, unknown>

    // EJSON types → BSON constructors
    if ('$oid' in obj && typeof obj.$oid === 'string') {
      return `ObjectId("${obj.$oid}")`
    }

    if ('$date' in obj) {
      const dateVal = obj.$date
      if (typeof dateVal === 'string') {
        return `ISODate("${dateVal}")`
      }
      if (typeof dateVal === 'object' && dateVal !== null && '$numberLong' in dateVal) {
        const ms = Number((dateVal as Record<string, unknown>).$numberLong)
        return `ISODate("${new Date(ms).toISOString()}")`
      }
      return `ISODate("${String(dateVal)}")`
    }

    if ('$numberLong' in obj && typeof obj.$numberLong === 'string') {
      return `NumberLong("${obj.$numberLong}")`
    }

    if ('$numberInt' in obj && typeof obj.$numberInt === 'string') {
      return obj.$numberInt
    }

    if ('$numberDouble' in obj && typeof obj.$numberDouble === 'string') {
      return obj.$numberDouble
    }

    if ('$numberDecimal' in obj && typeof obj.$numberDecimal === 'string') {
      return `NumberDecimal("${obj.$numberDecimal}")`
    }

    if ('$binary' in obj) {
      const binary = obj.$binary as Record<string, unknown>
      if (typeof binary?.base64 === 'string') {
        if (binary.subType === '04' || binary.subType === '03') {
          return `UUID("${base64ToUUID(binary.base64)}")`
        }
        return `BinData(${Number(binary.subType)}, "${binary.base64}")`
      }
    }

    if ('$regularExpression' in obj) {
      const re = obj.$regularExpression as Record<string, unknown>
      return `/${re.pattern}/${re.options || ''}`
    }

    if ('$regex' in obj) {
      return `/${obj.$regex}/${obj.$options || ''}`
    }

    if ('$timestamp' in obj) {
      const ts = obj.$timestamp as Record<string, unknown>
      return `Timestamp(${ts.t}, ${ts.i})`
    }

    if ('$minKey' in obj) {
      return 'MinKey()'
    }

    if ('$maxKey' in obj) {
      return 'MaxKey()'
    }

    // Regular object
    const entries = Object.entries(obj).map(([k, v]) => {
      const key = needsQuoting(k) ? JSON.stringify(k) : k
      return `${key}: ${toJsExpression(v)}`
    })
    return `{ ${entries.join(', ')} }`
  }

  if (typeof value === 'string') {
    // Detect humanized BSON types
    if (isIsoDateString(value)) {
      return `ISODate("${value}")`
    }
    if (isObjectIdHex(value)) {
      return `ObjectId("${value}")`
    }
    if (isUuidString(value)) {
      return `UUID("${value}")`
    }
    return JSON.stringify(value)
  }

  if (typeof value === 'number') {
    if (!isFinite(value)) {
      if (value === Infinity) {
        return 'Infinity'
      }
      if (value === -Infinity) {
        return '-Infinity'
      }
      return 'NaN'
    }
    return String(value)
  }

  if (typeof value === 'boolean') {
    return String(value)
  }

  return String(value)
}

const JS_IDENTIFIER_RE = /^[a-zA-Z_$][a-zA-Z0-9_$]*$/

function needsQuoting(key: string): boolean {
  return !JS_IDENTIFIER_RE.test(key)
}

// --- Helpers ---

const ISO_DATE_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/
const OBJECT_ID_RE = /^[0-9a-f]{24}$/
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

function isIsoDateString(str: string): boolean {
  return ISO_DATE_RE.test(str)
}

function isObjectIdHex(str: string): boolean {
  return OBJECT_ID_RE.test(str)
}

function isUuidString(str: string): boolean {
  return UUID_RE.test(str)
}

function parseSpecialDouble(str: string): number {
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

function base64ToHex(b64: string): string {
  const raw = atob(b64)
  return Array.from(raw, (ch) => ch.charCodeAt(0).toString(16).padStart(2, '0')).join('')
}

function hexToUUID(hex: string): string {
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`
}

function base64ToUUID(b64: string): string {
  return hexToUUID(base64ToHex(b64))
}

