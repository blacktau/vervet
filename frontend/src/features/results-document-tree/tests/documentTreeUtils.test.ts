import { describe, test, expect, vi } from 'vitest'
import { buildTreeData } from '../documentTreeUtils'

vi.mock('@/i18n', () => ({
  i18nGlobal: {
    t: (key: string, params?: Record<string, unknown>) => {
      if (params) {
        return `${key}:${JSON.stringify(params)}`
      }
      return key
    },
  },
}))

describe('buildTreeData', () => {
  test('returns empty array for empty input', () => {
    expect(buildTreeData([])).toEqual([])
  })

  test('returns empty array for undefined-like input', () => {
    expect(buildTreeData(undefined as unknown as unknown[])).toEqual([])
    expect(buildTreeData(null as unknown as unknown[])).toEqual([])
  })

  test('builds root document nodes', () => {
    const docs = [{ _id: { $oid: 'abc123' }, name: 'Alice' }]
    const result = buildTreeData(docs)

    expect(result).toHaveLength(1)
    expect(result[0].key).toBe('__doc_0')
    expect(result[0].isDocRoot).toBe(true)
    expect(result[0].field).toContain('abc123')
    expect(result[0].children).toHaveLength(2)
  })

  test('builds nested children for subdocuments', () => {
    const docs = [{ address: { city: 'London', zip: '12345' } }]
    const result = buildTreeData(docs)

    const addressChild = result[0].children?.find((c) => c.field === 'address')
    expect(addressChild).toBeDefined()
    expect(addressChild!.children).toHaveLength(2)
    expect(addressChild!.children![0].field).toBe('city')
    expect(addressChild!.children![0].value).toBe('London')
    expect(addressChild!.children![0].type).toBe('string')
  })

  test('builds array children with numeric keys', () => {
    const docs = [{ tags: ['a', 'b', 'c'] }]
    const result = buildTreeData(docs)

    const tagsChild = result[0].children?.find((c) => c.field === 'tags')
    expect(tagsChild).toBeDefined()
    expect(tagsChild!.type).toBe('array')
    expect(tagsChild!.children).toHaveLength(3)
    expect(tagsChild!.children![0].field).toBe('0')
    expect(tagsChild!.children![0].value).toBe('a')
  })

  test('detects BSON types correctly', () => {
    const docs = [
      {
        _id: { $oid: 'abc123' },
        created: { $date: '2024-01-01T00:00:00Z' },
        count: { $numberLong: '42' },
        active: true,
        score: 3.14,
        name: 'test',
        removed: null,
      },
    ]
    const result = buildTreeData(docs)
    const children = result[0].children!

    const findChild = (field: string) => children.find((c) => c.field === field)

    expect(findChild('_id')!.type).toBe('objectId')
    expect(findChild('created')!.type).toBe('date')
    expect(findChild('count')!.type).toBe('long')
    expect(findChild('active')!.type).toBe('boolean')
    expect(findChild('score')!.type).toBe('double')
    expect(findChild('name')!.type).toBe('string')
    expect(findChild('removed')!.type).toBe('null')
  })

  test('leaf nodes have no children property', () => {
    const docs = [{ name: 'Alice' }]
    const result = buildTreeData(docs)
    const nameChild = result[0].children?.find((c) => c.field === 'name')
    expect(nameChild!.children).toBeUndefined()
  })

  test('handles multiple documents', () => {
    const docs = [{ a: 1 }, { b: 2 }, { c: 3 }]
    const result = buildTreeData(docs)
    expect(result).toHaveLength(3)
    expect(result[0].key).toBe('__doc_0')
    expect(result[1].key).toBe('__doc_1')
    expect(result[2].key).toBe('__doc_2')
  })
})
