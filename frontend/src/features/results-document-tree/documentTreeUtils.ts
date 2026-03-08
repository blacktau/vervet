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
      return i18nGlobal.t('query.binaryValue', { subType: binary.subType })
    }
    if ('$regex' in obj) {
      return `/${obj.$regex}/${obj.$options || ''}`
    }
    if ('$timestamp' in obj) {
      const ts = obj.$timestamp as Record<string, unknown>
      return i18nGlobal.t('query.timestampValue', { t: ts.t, i: ts.i })
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
