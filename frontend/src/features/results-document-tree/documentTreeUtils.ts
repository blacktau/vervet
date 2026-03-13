import type { DocumentRow } from './types'
import { i18nGlobal } from '@/i18n'

const bsonKeyLookup: Record<string, string> = {
  $oid: 'objectId',
  $date: 'date',
  $numberDecimal: 'decimal128',
  $numberLong: 'long',
  $numberInt: 'int32',
  $numberDouble: 'double',
  $binary: 'binary',
  $regex: 'regex',
  $timestamp: 'timestamp',
  $minKey: 'minKey',
  $maxKey: 'maxKey',
}

const BINARY_SUBTYPE_UUID_LEGACY = '03'
const BINARY_SUBTYPE_UUID = '04'
const BINARY_SUBTYPE_MD5 = '05'

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
    for (const bsonKey in bsonKeyLookup) {
      if (bsonKey in val) {
        if (bsonKey === '$binary') {
          const binary = (val as Record<string, unknown>).$binary as Record<string, unknown>
          if (binary?.subType === BINARY_SUBTYPE_UUID || binary?.subType === BINARY_SUBTYPE_UUID_LEGACY) {
            return binary.subType === BINARY_SUBTYPE_UUID ? 'uuid' : 'uuidLegacy'
          }
          if (binary?.subType === BINARY_SUBTYPE_MD5) {
            return 'md5'
          }
        }
        return bsonKeyLookup[bsonKey]
      }
    }
    return 'document'
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
    if ('$oid' in obj) {
      return String(obj.$oid)
    }
    if ('$date' in obj) {
      return String(obj.$date)
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
      if (typeof binary?.base64 === 'string') {
        if (binary.subType === BINARY_SUBTYPE_UUID || binary.subType === BINARY_SUBTYPE_UUID_LEGACY) {
          return base64ToUUID(binary.base64)
        }
        if (binary.subType === BINARY_SUBTYPE_MD5) {
          return base64ToHex(binary.base64)
        }
      }
      return i18nGlobal.t('query.binaryValue', { subType: binary.subType })
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
    const keys = Object.keys(obj)
    return i18nGlobal.t('query.objectFields', { count: keys.length })
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
      : String(id)
    return `(${index + 1}) {_id: ${idStr}}`
  }
  return `(${index + 1})`
}

export function buildTreeData(documents: unknown[]): DocumentRow[] {
  if (!documents || documents.length === 0) {
    return []
  }

  return documents.map((doc, i) => {
    const fieldCount =
      typeof doc === 'object' && doc !== null ? Object.keys(doc).length : 0

    const children: DocumentRow[] =
      typeof doc === 'object' && doc !== null
        ? buildChildRows(doc, `__doc_${i}`)
        : []

    return {
      key: `__doc_${i}`,
      field: getDocLabel(doc, i),
      value: i18nGlobal.t('query.objectFields', { count: fieldCount }),
      type: 'document',
      typeLabel: getTypeName('document'),
      isDocRoot: true,
      children: children.length > 0 ? children : undefined,
    }
  })
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

  return entries.map(([key, val]) => {
    const nodeKey = prefix ? `${prefix}.${key}` : key
    const typeKey = getTypeKey(val)
    const hasChildren = typeKey === 'document' || typeKey === 'array'

    const children =
      hasChildren && typeof val === 'object' && val !== null
        ? buildChildRows(val, nodeKey)
        : undefined

    return {
      key: nodeKey,
      field: key,
      value: getDisplayValue(val),
      type: typeKey,
      typeLabel: getTypeName(typeKey),
      isDocRoot: false,
      children: children && children.length > 0 ? children : undefined,
    }
  })
}
