import type { DocumentRow } from './types'
import { i18nGlobal } from '@/i18n'
import {
  BINARY_SUBTYPE_MD5,
  BINARY_SUBTYPE_UUID,
  BINARY_SUBTYPE_UUID_LEGACY,
  binaryToText,
  epochMsToText,
  getBsonWrapperType,
} from './bsonValue'
import { humanizeEjson } from './humanizeEjson'

const primitiveTypeLookup: Record<string, string> = {
  string: 'string',
  boolean: 'boolean',
}

export function getTypeKey(val: unknown): string {
  if (val === null) {
    return 'null'
  }
  if (Array.isArray(val)) {
    return 'array'
  }
  if (typeof val === 'object') {
    const wrapperType = getBsonWrapperType(val)
    if (wrapperType !== 'binary') {
      return wrapperType ?? 'object'
    }
    const subType = ((val as Record<string, unknown>).$binary as Record<string, unknown>).subType
    if (subType === BINARY_SUBTYPE_UUID) {
      return 'uuid'
    }
    if (subType === BINARY_SUBTYPE_UUID_LEGACY) {
      return 'uuidLegacy'
    }
    return subType === BINARY_SUBTYPE_MD5 ? 'md5' : 'binary'
  }
  if (typeof val === 'number') {
    return Number.isInteger(val) ? 'int32' : 'double'
  }
  return primitiveTypeLookup[typeof val] ?? typeof val
}

export function getTypeName(typeKey: string): string {
  return i18nGlobal.t(`query.types.${typeKey}`)
}

export function getDisplayValue(val: unknown): string {
  if (val === null) {
    return 'null'
  }
  if (Array.isArray(val)) {
    return i18nGlobal.t('query.arrayElements', { count: val.length })
  }
  if (typeof val === 'object') {
    const obj = val as Record<string, unknown>
    if (!getBsonWrapperType(obj)) {
      return i18nGlobal.t('query.objectFields', { count: Object.keys(obj).length })
    }
    if ('$oid' in obj) {
      return String(obj.$oid)
    }
    if ('$date' in obj) {
      const dateVal = obj.$date
      if (typeof dateVal === 'object' && dateVal !== null && '$numberLong' in dateVal) {
        return epochMsToText(String((dateVal as Record<string, unknown>).$numberLong))
      }
      return String(dateVal)
    }
    if ('$numberDecimal' in obj) {
      return String(obj.$numberDecimal)
    }
    if ('$numberLong' in obj) {
      return String(obj.$numberLong)
    }
    if ('$numberInt' in obj) {
      return String(obj.$numberInt)
    }
    if ('$numberDouble' in obj) {
      return String(obj.$numberDouble)
    }
    if ('$binary' in obj) {
      const binary = obj.$binary as Record<string, unknown>
      const text = binaryToText(String(binary.base64), binary.subType)
      if (text !== undefined) {
        return text
      }
      return i18nGlobal.t('query.binaryValue', { subType: binary.subType })
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
      return i18nGlobal.t('query.timestampValue', { t: ts.t, i: ts.i })
    }
    if ('$minKey' in obj) {
      return 'MinKey'
    }
    if ('$maxKey' in obj) {
      return 'MaxKey'
    }
    if ('$code' in obj) {
      return String(obj.$code)
    }
    if ('$symbol' in obj) {
      return String(obj.$symbol)
    }
    if ('$uuid' in obj) {
      return String(obj.$uuid)
    }
    if ('$dbPointer' in obj) {
      const pointer = obj.$dbPointer as Record<string, unknown>
      return `DBPointer(${pointer.$ref}, ${getDisplayValue(pointer.$id)})`
    }
    return 'undefined'
  }
  if (typeof val === 'string') {
    return val
  }
  return String(val)
}

export function getDocLabel(doc: unknown, index: number): string {
  if (typeof doc !== 'object' || doc === null) {
    return `(${index + 1})`
  }
  const record = doc as Record<string, unknown>
  const id = record._id
  if (id !== undefined) {
    const idStr = typeof id === 'object' && id !== null && '$oid' in id
      ? (id as Record<string, unknown>).$oid
      : getDisplayValue(id)
    return `(${index + 1}) {_id: ${idStr}}`
  }
  return `(${index + 1})`
}

export function buildTreeData(documents: unknown[]): DocumentRow[] {
  if (!documents || documents.length === 0) {
    return []
  }

  return documents.map((doc, i) => {
    const row = buildValueRow(`__doc_${i}`, getDocLabel(doc, i), doc)

    // A script can return any value, such as the strings from getCollectionNames()
    if (row.type !== 'object') {
      return { ...row, kind: 'value' }
    }

    return { ...row, kind: 'document', type: 'document', typeLabel: getTypeName('document') }
  })
}

function buildValueRow(key: string, field: string, val: unknown): DocumentRow {
  const typeKey = getTypeKey(val)
  const hasChildren = typeKey === 'object' || typeKey === 'array'
  const children = hasChildren ? buildChildRows(val, key) : []

  return {
    key,
    field,
    value: getDisplayValue(val),
    type: typeKey,
    typeLabel: getTypeName(typeKey),
    kind: 'field',
    children: children.length > 0 ? children : undefined,
  }
}

function buildChildRows(
  obj: unknown,
  prefix: string,
): DocumentRow[] {
  if (typeof obj !== 'object' || obj === null) {
    return []
  }

  const entries = Array.isArray(obj)
    ? obj.map((v, i) => [String(i), v] as [string, unknown])
    : Object.entries(obj)

  return entries.map(([key, val]) => buildValueRow(prefix ? `${prefix}.${key}` : key, key, val))
}

/**
 * The text to copy for a value. Scalars copy as shown, since converting them can
 * change them: a long past 2^53 rounds, and NaN becomes null in JSON.
 */
export function getCopyText(val: unknown, pretty = false): string {
  const typeKey = getTypeKey(val)
  if (typeKey !== 'array' && typeKey !== 'object' && typeKey !== 'binary') {
    return getDisplayValue(val)
  }
  return JSON.stringify(humanizeEjson(val), null, pretty ? 2 : undefined)
}
