import { describe, test, expect, vi } from 'vitest'
import { buildTreeData, getCopyText, getDocLabel } from '../documentTreeUtils'

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
    expect(result[0]?.key).toBe('__doc_0')
    expect(result[0]?.kind).toBe('document')
    expect(result[0]?.field).toContain('abc123')
    expect(result[0]?.children).toHaveLength(2)
  })

  test('builds nested children for subdocuments', () => {
    const docs = [{ address: { city: 'London', zip: '12345' } }]
    const result = buildTreeData(docs)

    const addressChild = result[0]?.children?.find((c) => c.field === 'address')
    expect(addressChild).toBeDefined()
    expect(addressChild!.children).toHaveLength(2)
    expect(addressChild!.children![0]?.field).toBe('city')
    expect(addressChild!.children![0]?.value).toBe('London')
    expect(addressChild!.children![0]?.type).toBe('string')
  })

  test('builds array children with numeric keys', () => {
    const docs = [{ tags: ['a', 'b', 'c'] }]
    const result = buildTreeData(docs)

    const tagsChild = result[0]?.children?.find((c) => c.field === 'tags')
    expect(tagsChild).toBeDefined()
    expect(tagsChild!.type).toBe('array')
    expect(tagsChild!.children).toHaveLength(3)
    expect(tagsChild!.children![0]?.field).toBe('0')
    expect(tagsChild!.children![0]?.value).toBe('a')
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
    const children = result[0]!.children!

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
    const nameChild = result[0]?.children?.find((c) => c.field === 'name')
    expect(nameChild!.children).toBeUndefined()
  })

  test('handles multiple documents', () => {
    const docs = [{ a: 1 }, { b: 2 }, { c: 3 }]
    const result = buildTreeData(docs)
    expect(result).toHaveLength(3)
    expect(result[0]?.key).toBe('__doc_0')
    expect(result[1]?.key).toBe('__doc_1')
    expect(result[2]?.key).toBe('__doc_2')
  })

  describe('top-level values that are not documents', () => {
    test.each([
      ['comments', 'comments', 'string'],
      [42, '42', 'int32'],
      [2.5, '2.5', 'double'],
      [0, '0', 'int32'],
      [false, 'false', 'boolean'],
      [null, 'null', 'null'],
      ['', '', 'string'],
      [{ $oid: 'abc123' }, 'abc123', 'objectId'],
      [{ $numberLong: '7' }, '7', 'long'],
      [{ $date: '2024-01-01T00:00:00Z' }, '2024-01-01T00:00:00Z', 'date'],
      [{ $code: 'x()' }, 'x()', 'code'],
      [{ $code: 'x()', $scope: {} }, 'x()', 'code'],
      [{ $symbol: 's' }, 's', 'symbol'],
      [{ $undefined: true }, 'undefined', 'undefined'],
      [{ $uuid: 'abc' }, 'abc', 'uuid'],
      [
        { $dbPointer: { $ref: 'db.c', $id: { $oid: 'abc123' } } },
        'DBPointer(db.c, abc123)',
        'dbPointer',
      ],
    ])('shows %j as its value and type', (value, displayValue, type) => {
      const [row] = buildTreeData([value])

      expect(row).toMatchObject({
        key: '__doc_0',
        field: '(1)',
        value: displayValue,
        type,
        typeLabel: `query.types.${type}`,
        kind: 'value',
      })
      expect(row?.children).toBeUndefined()
    })

    test('numbers each row', () => {
      const rows = buildTreeData(['a', 'b'])

      expect(rows.map((r) => [r.field, r.value])).toEqual([
        ['(1)', 'a'],
        ['(2)', 'b'],
      ])
    })

    test('shows an array as an array, with its elements as children', () => {
      const [row] = buildTreeData([['x', 'y']])

      expect(row?.type).toBe('array')
      expect(row?.kind).toBe('value')
      expect(row?.children?.map((c) => [c.key, c.value])).toEqual([
        ['__doc_0.0', 'x'],
        ['__doc_0.1', 'y'],
      ])
    })

    test('keeps a mix of documents and values apart', () => {
      const rows = buildTreeData([{ a: 1 }, 'b'])

      expect(rows[0]).toMatchObject({ type: 'document', kind: 'document' })
      expect(rows[1]).toMatchObject({ type: 'string', kind: 'value' })
    })

    test.each([
      [{ _id: 1, $timestamp: null }],
      [{ _id: 1, name: 'x', $date: '2024-01-01' }],
      [{ $timestamp: null }],
      [{ $binary: null }],
      [{ $regularExpression: null }],
      [{ $date: null }],
      [{ $oid: 5 }],
      [{ $date: 'x', $oid: 'y' }],
    ])('treats %j as a document, since it is not an exact BSON wrapper', (doc) => {
      const [row] = buildTreeData([doc])

      expect(row).toMatchObject({ kind: 'document', type: 'document' })
      expect(row?.children).toHaveLength(Object.keys(doc).length)
    })

    test('shows a nested field that only looks like a BSON wrapper as an object', () => {
      const [row] = buildTreeData([{ meta: { $date: '2024-01-01', name: 'x' } }])

      expect(row?.children?.[0]).toMatchObject({ type: 'object' })
      expect(row?.children?.[0]?.children).toHaveLength(2)
    })

    test('does not throw on invalid base64 in a UUID', () => {
      const [row] = buildTreeData([{ $binary: { base64: '!!!', subType: '04' } }])

      expect(row).toMatchObject({ kind: 'value', type: 'uuid' })
    })

    test('does not throw on a date a JS Date cannot hold', () => {
      const [row] = buildTreeData([{ $date: { $numberLong: '99999999999999999' } }])

      expect(row).toMatchObject({ kind: 'value', value: '99999999999999999' })
    })

    test('still treats an empty object as a document', () => {
      const [row] = buildTreeData([{}])

      expect(row).toMatchObject({ type: 'document', kind: 'document' })
    })
  })
})

describe('getCopyText', () => {
  test.each([
    ['theaters', 'theaters'],
    [null, 'null'],
    [false, 'false'],
    [{ $numberLong: '9007199254740993' }, '9007199254740993'],
    [{ $numberDouble: 'NaN' }, 'NaN'],
    [{ $numberDouble: 'Infinity' }, 'Infinity'],
    [{ $oid: 'abc123' }, 'abc123'],
  ])('copies %j as it is shown', (value, text) => {
    expect(getCopyText(value)).toBe(text)
  })

  test('copies a timestamp as it is shown', () => {
    const value = { $timestamp: { t: 1, i: 2 } }

    expect(getCopyText(value)).toBe('query.timestampValue:{"t":1,"i":2}')
  })

  test('copies a binary as EJSON, since the text shown does not hold the data', () => {
    const value = { $binary: { base64: 'AA==', subType: '00' } }

    expect(getCopyText(value)).toBe(JSON.stringify(value))
  })

  test('copies Code with its scope, since the text shown leaves it out', () => {
    const value = { $code: 'x()', $scope: { a: 1 } }

    expect(getCopyText(value)).toBe(JSON.stringify(value))
  })

  test('copies a UUID with invalid base64 as EJSON, not the placeholder shown', () => {
    const value = { $binary: { base64: '!!!', subType: '04' } }

    expect(getCopyText(value)).toBe(JSON.stringify(value))
  })

  test('copies a valid UUID as shown', () => {
    const value = { $binary: { base64: 'AAAAAAAAAAAAAAAAAAAAAA==', subType: '04' } }

    expect(getCopyText(value)).toBe('00000000-0000-0000-0000-000000000000')
  })

  test('indents an array only when asked to', () => {
    expect(getCopyText(['a'])).toBe('["a"]')
    expect(getCopyText(['a'], true)).toBe('[\n  "a"\n]')
  })
})

describe('getDocLabel', () => {
  test('shows an ObjectId _id', () => {
    expect(getDocLabel({ _id: { $oid: 'abc123' } }, 0)).toBe('(1) {_id: abc123}')
  })

  test('agrees with the tree when the _id only looks like an ObjectId', () => {
    expect(getDocLabel({ _id: { $oid: 'abc123', x: 1 } }, 0)).toBe(
      '(1) {_id: query.objectFields:{"count":2}}',
    )
  })
})
