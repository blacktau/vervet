import type { FlatRow } from './types'
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

export function buildRowMap(obj: unknown, prefix: string, depth: number, target: Map<string, FlatRow>) {
  if (typeof obj !== 'object' || obj === null) {
    return
  }

  const entries = Array.isArray(obj)
    ? obj.map((v, i) => [String(i), v] as [string, unknown])
    : Object.entries(obj)

  for (const [key, val] of entries) {
    const nodeKey = prefix ? `${prefix}.${key}` : key
    const typeKey = getTypeKey(val)
    const hasChildren = typeKey === 'document' || typeKey === 'array'

    const childKeys: string[] = []
    if (hasChildren && typeof val === 'object' && val !== null) {
      const childEntries = Array.isArray(val)
        ? val.map((_v, i) => String(i))
        : Object.keys(val)
      for (const ck of childEntries) {
        childKeys.push(`${nodeKey}.${ck}`)
      }
    }

    target.set(nodeKey, {
      key: nodeKey,
      field: key,
      value: getDisplayValue(val),
      type: typeKey,
      depth,
      hasChildren,
      expanded: false,
      childKeys,
      isDocRoot: false,
    })

    if (hasChildren) {
      buildRowMap(val, nodeKey, depth + 1, target)
    }
  }
}

const typeClassLookup: Record<string, string> = {
  string: 'type-string',
  int32: 'type-number',
  double: 'type-number',
  long: 'type-number',
  decimal128: 'type-number',
  boolean: 'type-boolean',
  null: 'type-null',
  objectId: 'type-objectid',
  date: 'type-date',
  document: 'type-composite',
  array: 'type-composite',
}

export function getTypeClass(typeKey: string): string {
  return typeClassLookup[typeKey] ?? ''
}
