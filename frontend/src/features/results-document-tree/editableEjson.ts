import { binaryToText, BINARY_SUBTYPE_UUID, dateEpochMs, getBsonWrapperType } from './bsonValue'

const INT32_TEXT = /^-?\d+$/
const MIN_ISO_YEAR = 1
const MAX_ISO_YEAR = 9999

/**
 * Converts canonical Extended JSON into the text the edit dialog shows. Only values
 * that come back with the same BSON type become plain JSON: strings, booleans, null,
 * int32 and fractional doubles. A whole-number double has to stay a wrapper, as
 * plain 5 is stored as an int32.
 */
export function toEditableEjson(value: unknown): unknown {
  if (Array.isArray(value)) {
    return value.map(toEditableEjson)
  }
  if (typeof value !== 'object' || value === null) {
    return value
  }

  const obj = value as Record<string, unknown>
  switch (getBsonWrapperType(obj)) {
    case undefined:
      return Object.fromEntries(Object.entries(obj).map(([k, v]) => [k, toEditableEjson(v)]))
    case 'int32':
      return INT32_TEXT.test(String(obj.$numberInt)) ? Number(obj.$numberInt) : value
    case 'double': {
      const number = Number(obj.$numberDouble)
      return Number.isFinite(number) && !Number.isInteger(number) ? number : value
    }
    case 'date':
      return readableDate(obj) ?? value
    case 'binary':
      return readableUuid(obj) ?? value
    default:
      return value
  }
}

function readableDate(wrapper: Record<string, unknown>): unknown {
  const epochMs = dateEpochMs(wrapper.$date)
  if (epochMs === undefined) {
    return undefined
  }
  const date = new Date(Number(epochMs))
  const year = date.getUTCFullYear()
  // Outside these years, ISO text isn't four digits and parsers disagree on it
  if (Number.isNaN(year) || year < MIN_ISO_YEAR || year > MAX_ISO_YEAR) {
    return undefined
  }
  return { $date: date.toISOString() }
}

function readableUuid(wrapper: Record<string, unknown>): unknown {
  const { base64, subType } = wrapper.$binary as { base64: string; subType: unknown }
  if (subType !== BINARY_SUBTYPE_UUID) {
    return undefined
  }
  const text = binaryToText(base64, subType)
  // binaryToText has shown the base64 is valid. A UUID is exactly 16 bytes.
  return text !== undefined && atob(base64).length === 16 ? { $uuid: text } : undefined
}

/**
 * A script expression for a document written as Extended JSON. Both query engines
 * read every type back from this form, and JSON.stringify means nothing in the
 * text can run as script, whatever a stored document holds.
 */
export function ejsonParseExpression(json: string): string {
  return `EJSON.parse(${JSON.stringify(json)}, { relaxed: false })`
}

/** A filter expression matching a document by its _id, given as canonical Extended JSON. */
export function documentIdFilter(id: unknown): string {
  // JSON.stringify would drop it, leaving an empty filter that matches any document
  if (id === undefined) {
    throw new Error('A document filter needs an _id')
  }
  return ejsonParseExpression(JSON.stringify({ _id: id }))
}

/**
 * A count from a write's reply, or undefined if the reply doesn't have it. The
 * builtin engine sends plain numbers and mongosh sends canonical Extended JSON.
 */
export function writeCount(documents: unknown, field: string): number | undefined {
  const reply = Array.isArray(documents) ? (documents[0] as Record<string, unknown>) : undefined
  const count = reply?.[field]
  if (typeof count === 'number') {
    return count
  }
  const wrapped = (count as { $numberInt?: unknown } | null | undefined)?.$numberInt
  return typeof wrapped === 'string' ? Number(wrapped) : undefined
}
