import { describe, it, expect } from 'vitest'
import { buildSchemaRows } from '../schemaRows'
import type { models } from 'wailsjs/go/models'

function field(over: Partial<models.FieldInfo>): models.FieldInfo {
  return {
    path: 'x',
    name: 'x',
    count: 0,
    types: [],
    children: [],
    ...over,
  } as models.FieldInfo
}

describe('buildSchemaRows', () => {
  it('computes presence as percent of sampled count', () => {
    const rows = buildSchemaRows(
      [field({ path: 'a', name: 'a', count: 50 })],
      100,
    )
    expect(rows[0]!.presence).toBe(50)
    expect(rows[0]!.isArrayElement).toBe(false)
  })

  it('caps presence at 100%', () => {
    const rows = buildSchemaRows([field({ count: 250 })], 100)
    expect(rows[0]!.presence).toBe(100)
  })

  it('returns 0 presence when sampledCount is zero', () => {
    const rows = buildSchemaRows([field({ count: 5 })], 0)
    expect(rows[0]!.presence).toBe(0)
  })

  it('computes avgPerParent for [] virtual children', () => {
    const rows = buildSchemaRows(
      [
        field({
          path: 'tags',
          name: 'tags',
          count: 100,
          children: [field({ path: 'tags[]', name: '[]', count: 320 })],
        }),
      ],
      100,
    )
    const arrayElem = rows[0]!.children![0]!
    expect(arrayElem.isArrayElement).toBe(true)
    expect(arrayElem.presence).toBe(0)
    expect(arrayElem.avgPerParent).toBeCloseTo(3.2)
  })

  it('flags hasChildren only when children non-empty', () => {
    const rows = buildSchemaRows(
      [
        field({ path: 'a', name: 'a', children: [] }),
        field({
          path: 'b',
          name: 'b',
          children: [field({ path: 'b.c', name: 'c' })],
        }),
      ],
      1,
    )
    expect(rows[0]!.hasChildren).toBe(false)
    expect(rows[1]!.hasChildren).toBe(true)
    expect(rows[1]!.children).toHaveLength(1)
  })

  it('uses field path as row key', () => {
    const rows = buildSchemaRows(
      [field({ path: 'nested.deep.x', name: 'x' })],
      1,
    )
    expect(rows[0]!.key).toBe('nested.deep.x')
  })
})
