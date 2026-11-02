interface BsonWrapper {
  type: string
  isPayload: (payload: unknown) => boolean
  /** A second key the wrapper may carry, as in { $regex, $options } */
  companion?: string
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isString(value: unknown): boolean {
  return typeof value === 'string'
}

const bsonWrappers = new Map<string, BsonWrapper>([
  ['$oid', { type: 'objectId', isPayload: isString }],
  [
    '$date',
    {
      type: 'date',
      isPayload: (v) =>
        isString(v) || typeof v === 'number' || (isRecord(v) && isString(v.$numberLong)),
    },
  ],
  ['$numberDecimal', { type: 'decimal128', isPayload: isString }],
  ['$numberLong', { type: 'long', isPayload: isString }],
  ['$numberInt', { type: 'int32', isPayload: isString }],
  ['$numberDouble', { type: 'double', isPayload: isString }],
  ['$binary', { type: 'binary', isPayload: (v) => isRecord(v) && isString(v.base64) }],
  ['$regex', { type: 'regex', isPayload: isString, companion: '$options' }],
  ['$regularExpression', { type: 'regex', isPayload: (v) => isRecord(v) && isString(v.pattern) }],
  ['$timestamp', { type: 'timestamp', isPayload: (v) => isRecord(v) && 't' in v && 'i' in v }],
  ['$minKey', { type: 'minKey', isPayload: (v) => v === 1 }],
  ['$maxKey', { type: 'maxKey', isPayload: (v) => v === 1 }],
  ['$code', { type: 'code', isPayload: isString, companion: '$scope' }],
  ['$symbol', { type: 'symbol', isPayload: isString }],
  ['$undefined', { type: 'undefined', isPayload: (v) => v === true }],
  ['$uuid', { type: 'uuid', isPayload: isString }],
  [
    '$dbPointer',
    { type: 'dbPointer', isPayload: (v) => isRecord(v) && isString(v.$ref) && isRecord(v.$id) },
  ],
])

/**
 * The BSON type an EJSON wrapper holds, or undefined for an ordinary object. Only
 * the exact wrapper shape counts: MongoDB can store a field named $date, and
 * { _id: 1, $date: 'x' } or { $timestamp: null } are documents.
 */
export function getBsonWrapperType(value: object): string | undefined {
  const keys = Object.keys(value)
  const wrapperKey = keys.find((key) => bsonWrappers.has(key))
  if (!wrapperKey) {
    return undefined
  }

  const wrapper = bsonWrappers.get(wrapperKey)!
  const isOnlyWrapperKeys = keys.every((key) => key === wrapperKey || key === wrapper.companion)
  if (!isOnlyWrapperKeys || !wrapper.isPayload((value as Record<string, unknown>)[wrapperKey])) {
    return undefined
  }

  return wrapper.type
}

export const BINARY_SUBTYPE_UUID_LEGACY = '03'
export const BINARY_SUBTYPE_UUID = '04'
export const BINARY_SUBTYPE_MD5 = '05'

function base64ToHex(b64: string): string {
  const raw = atob(b64)
  return Array.from(raw, (ch) => ch.charCodeAt(0).toString(16).padStart(2, '0')).join('')
}

/** The UUID or MD5 text of a binary value; undefined for other subtypes or invalid base64. */
export function binaryToText(base64: string, subType: unknown): string | undefined {
  const isUuid = subType === BINARY_SUBTYPE_UUID || subType === BINARY_SUBTYPE_UUID_LEGACY
  if (!isUuid && subType !== BINARY_SUBTYPE_MD5) {
    return undefined
  }

  try {
    const hex = base64ToHex(base64)
    return isUuid
      ? `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20, 32)}`
      : hex
  } catch {
    return undefined
  }
}

/** ISO text for epoch milliseconds, or the raw value when a JS Date can't hold it. */
export function epochMsToText(epochMs: string): string {
  const date = new Date(Number(epochMs))
  return Number.isNaN(date.getTime()) ? epochMs : date.toISOString()
}
