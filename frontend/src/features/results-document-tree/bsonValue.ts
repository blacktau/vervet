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

function isEpochMsInRange(epochMs: unknown): boolean {
  return !Number.isNaN(new Date(Number(epochMs)).getTime())
}

/** The epoch milliseconds of a $date payload, or undefined for a date already in text. */
export function dateEpochMs(payload: unknown): string | undefined {
  if (typeof payload === 'number') {
    return String(payload)
  }
  if (isRecord(payload) && typeof payload.$numberLong === 'string') {
    return payload.$numberLong
  }
  return undefined
}

/**
 * Whether a value holds a date beyond a JS Date's range. Scripts handle dates as
 * JS Dates, so a query can't write such a value back or match it.
 */
export function hasUnwritableDate(value: unknown): boolean {
  if (Array.isArray(value)) {
    return value.some(hasUnwritableDate)
  }
  if (!isRecord(value)) {
    return false
  }
  if (getBsonWrapperType(value) === 'date') {
    const epochMs = dateEpochMs(value.$date)
    return epochMs !== undefined && !isEpochMsInRange(epochMs)
  }
  return Object.values(value).some(hasUnwritableDate)
}

const INTEGER_TEXT = /^-?\d+$/
const INT32_MIN = -2147483648
const INT32_MAX = 2147483647

/** Whether text is an int32 as canonical Extended JSON writes it, so it fits and converts exactly. */
export function isInt32Text(text: unknown): boolean {
  if (typeof text !== 'string' || !INTEGER_TEXT.test(text)) {
    return false
  }
  const number = Number(text)
  return number >= INT32_MIN && number <= INT32_MAX && String(number) === text
}

const SPECIAL_DOUBLE_TEXT = /^(NaN|-?Infinity)$/
const DECIMAL_TEXT = /^[-+]?(\d|\.\d|NaN|Inf)/i
const OBJECT_ID_TEXT = /^[0-9a-f]{24}$/
const BINARY_SUBTYPE_TEXT = /^[0-9a-f]{1,2}$/i

function hasExactKeys(value: Record<string, unknown>, ...keys: string[]): boolean {
  const actual = Object.keys(value)
  return actual.length === keys.length && keys.every((key) => key in value)
}

/** How each wrapper looks when the server's value is written as canonical Extended JSON. */
const canonicalPayloads = new Map<string, (v: unknown) => boolean>([
  ['$oid', (v) => typeof v === 'string' && OBJECT_ID_TEXT.test(v)],
  [
    '$date',
    (v) =>
      isRecord(v) &&
      hasExactKeys(v, '$numberLong') &&
      typeof v.$numberLong === 'string' &&
      INTEGER_TEXT.test(v.$numberLong),
  ],
  ['$numberInt', isInt32Text],
  ['$numberLong', (v) => typeof v === 'string' && INTEGER_TEXT.test(v)],
  [
    '$numberDouble',
    (v) =>
      typeof v === 'string' &&
      v.trim() !== '' &&
      (SPECIAL_DOUBLE_TEXT.test(v) || !Number.isNaN(Number(v))),
  ],
  ['$numberDecimal', (v) => typeof v === 'string' && DECIMAL_TEXT.test(v)],
  [
    '$binary',
    (v) =>
      isRecord(v) &&
      hasExactKeys(v, 'base64', 'subType') &&
      typeof v.base64 === 'string' &&
      typeof v.subType === 'string' &&
      BINARY_SUBTYPE_TEXT.test(v.subType),
  ],
  [
    '$regularExpression',
    (v) =>
      isRecord(v) &&
      hasExactKeys(v, 'pattern', 'options') &&
      typeof v.pattern === 'string' &&
      typeof v.options === 'string',
  ],
  [
    '$timestamp',
    (v) =>
      isRecord(v) &&
      hasExactKeys(v, 't', 'i') &&
      typeof v.t === 'number' &&
      typeof v.i === 'number',
  ],
  ['$minKey', (v) => v === 1],
  ['$maxKey', (v) => v === 1],
  ['$code', (v) => typeof v === 'string'],
  ['$symbol', (v) => typeof v === 'string'],
  ['$undefined', (v) => v === true],
  [
    '$dbPointer',
    (v) =>
      isRecord(v) &&
      hasExactKeys(v, '$ref', '$id') &&
      typeof v.$ref === 'string' &&
      isRecord(v.$id) &&
      hasExactKeys(v.$id, '$oid'),
  ],
])

/** Keys the Extended JSON parsers read as a type. These aren't all canonical output. */
const EJSON_TYPE_KEYS = new Set([...canonicalPayloads.keys(), '$uuid', '$regex', '$scope'])

function isCanonicalWrapper(value: Record<string, unknown>): boolean {
  const keys = Object.keys(value)
  if (keys.length === 2 && '$code' in value && '$scope' in value) {
    return typeof value.$code === 'string' && isRecord(value.$scope)
  }
  const isPayload = canonicalPayloads.get(keys[0]!)
  return keys.length === 1 && isPayload !== undefined && isPayload(value[keys[0]!])
}

/**
 * Whether a value holds an object with a field named like an Extended JSON type that
 * isn't that type, such as { $uuid: '...' } or { $oid: 5 }. Real values always arrive
 * in canonical form, so anything else is an ordinary object, which EJSON.parse would
 * read as the type or reject. Saving it can't give back what is stored.
 */
export function hasAmbiguousEjsonField(value: unknown): boolean {
  if (Array.isArray(value)) {
    return value.some(hasAmbiguousEjsonField)
  }
  if (!isRecord(value) || isCanonicalWrapper(value)) {
    return false
  }
  return (
    Object.keys(value).some((key) => EJSON_TYPE_KEYS.has(key)) ||
    Object.values(value).some(hasAmbiguousEjsonField)
  )
}

const INTEGER_KEY = /^(0|[1-9]\d*)$/

/**
 * Whether a value holds an object whose field order may have been lost: JS objects
 * put integer-like keys first. An _id only matches with its fields in stored order.
 */
export function hasReorderedKeys(value: unknown): boolean {
  if (Array.isArray(value)) {
    return value.some(hasReorderedKeys)
  }
  if (!isRecord(value)) {
    return false
  }
  const keys = Object.keys(value)
  return (
    (keys.length > 1 && keys.some((key) => INTEGER_KEY.test(key))) ||
    Object.values(value).some(hasReorderedKeys)
  )
}
