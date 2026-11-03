import {
  binaryToText,
  dateEpochMs,
  epochMsToText,
  getBsonWrapperType,
  hasUnwritableDate,
} from './bsonValue'

/**
 * Converts canonical Extended JSON (EJSON) values into human-readable equivalents,
 * matching the display logic in documentTreeUtils.getDisplayValue(). This is for
 * reading only and loses BSON types: the edit dialog uses toEditableEjson instead.
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

    if (!getBsonWrapperType(obj)) {
      return humanizeFields(obj)
    }

    if ('$oid' in obj && typeof obj.$oid === 'string') {
      return obj.$oid
    }

    if ('$date' in obj) {
      // No text form can hold this date
      if (hasUnwritableDate(obj)) {
        return value
      }
      const epochMs = dateEpochMs(obj.$date)
      return epochMs === undefined ? String(obj.$date) : epochMsToText(epochMs)
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
      // Non-UUID/MD5 binary: keep as-is
      return binaryToText(String(binary.base64), binary.subType) ?? value
    }

    if ('$regularExpression' in obj) {
      const re = obj.$regularExpression as Record<string, unknown>
      return `/${re.pattern}/${re.options || ''}`
    }

    if ('$regex' in obj) {
      return `/${obj.$regex}/${obj.$options || ''}`
    }

    // $timestamp, $minKey, $maxKey: keep as EJSON (not cleanly representable as a primitive)
    return humanizeFields(obj)
  }

  return value
}

function humanizeFields(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [k, v] of Object.entries(obj)) {
    result[k] = humanizeEjson(v)
  }
  return result
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
