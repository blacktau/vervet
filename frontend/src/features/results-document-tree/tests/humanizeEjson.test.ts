import { describe, it, expect } from 'vitest'
import { humanizeEjson, toJsExpression } from '../humanizeEjson'

describe('humanizeEjson', () => {
  // --- ObjectId ---
  it('converts $oid to plain hex string', () => {
    expect(humanizeEjson({ $oid: '507f1f77bcf86cd799439011' })).toBe('507f1f77bcf86cd799439011')
  })

  // --- Date ---
  it('converts $date ISO string', () => {
    expect(humanizeEjson({ $date: '2024-01-23T10:13:20.000Z' })).toBe('2024-01-23T10:13:20.000Z')
  })

  it('converts $date with nested $numberLong (epoch ms)', () => {
    const input = { $date: { $numberLong: '1706004800000' } }
    const result = humanizeEjson(input)
    expect(result).toBe(new Date(1706004800000).toISOString())
  })

  // --- Numbers ---
  it('converts $numberLong to number', () => {
    expect(humanizeEjson({ $numberLong: '123456' })).toBe(123456)
  })

  it('converts $numberInt to number', () => {
    expect(humanizeEjson({ $numberInt: '42' })).toBe(42)
  })

  it('converts $numberDouble to number', () => {
    expect(humanizeEjson({ $numberDouble: '3.14' })).toBe(3.14)
  })

  it('handles special double values', () => {
    expect(humanizeEjson({ $numberDouble: 'Infinity' })).toBe(Infinity)
    expect(humanizeEjson({ $numberDouble: '-Infinity' })).toBe(-Infinity)
    expect(humanizeEjson({ $numberDouble: 'NaN' })).toBeNaN()
  })

  it('converts $numberDecimal to number when safe', () => {
    expect(humanizeEjson({ $numberDecimal: '42' })).toBe(42)
    expect(humanizeEjson({ $numberDecimal: '3.14' })).toBe(3.14)
  })

  it('keeps $numberDecimal as string when not safely representable', () => {
    expect(humanizeEjson({ $numberDecimal: '1234567890.123456789012345' })).toBe(
      '1234567890.123456789012345',
    )
  })

  // --- Binary: UUID ---
  it('converts $binary UUID (subType 04) to UUID string', () => {
    const base64 = btoa(
      String.fromCharCode(
        ...[0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00],
      ),
    )
    const input = { $binary: { base64, subType: '04' } }
    expect(humanizeEjson(input)).toBe('550e8400-e29b-41d4-a716-446655440000')
  })

  it('converts $binary legacy UUID (subType 03) to UUID string', () => {
    const base64 = btoa(
      String.fromCharCode(
        ...[0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00],
      ),
    )
    const input = { $binary: { base64, subType: '03' } }
    expect(humanizeEjson(input)).toBe('550e8400-e29b-41d4-a716-446655440000')
  })

  it('converts $binary MD5 (subType 05) to hex string', () => {
    const base64 = btoa(
      String.fromCharCode(
        ...[0xd4, 0x1d, 0x8c, 0xd9, 0x8f, 0x00, 0xb2, 0x04, 0xe9, 0x80, 0x09, 0x98, 0xec, 0xf8, 0x42, 0x7e],
      ),
    )
    const input = { $binary: { base64, subType: '05' } }
    expect(humanizeEjson(input)).toBe('d41d8cd98f00b204e9800998ecf8427e')
  })

  it('keeps non-UUID/MD5 $binary as-is', () => {
    const input = { $binary: { base64: 'AQID', subType: '00' } }
    expect(humanizeEjson(input)).toEqual(input)
  })

  // --- Regex ---
  it('converts $regularExpression to /pattern/options', () => {
    const input = { $regularExpression: { pattern: '^test', options: 'i' } }
    expect(humanizeEjson(input)).toBe('/^test/i')
  })

  it('converts $regex to /pattern/options', () => {
    const input = { $regex: '^test', $options: 'gi' }
    expect(humanizeEjson(input)).toBe('/^test/gi')
  })

  it('handles $regularExpression with no options', () => {
    const input = { $regularExpression: { pattern: 'foo' } }
    expect(humanizeEjson(input)).toBe('/foo/')
  })

  // --- Timestamp, MinKey, MaxKey: kept as-is ---
  it('keeps $timestamp as-is', () => {
    const input = { $timestamp: { t: 1234567890, i: 1 } }
    expect(humanizeEjson(input)).toEqual({ $timestamp: { t: 1234567890, i: 1 } })
  })

  it('keeps $minKey as-is', () => {
    const input = { $minKey: 1 }
    expect(humanizeEjson(input)).toEqual({ $minKey: 1 })
  })

  it('keeps $maxKey as-is', () => {
    const input = { $maxKey: 1 }
    expect(humanizeEjson(input)).toEqual({ $maxKey: 1 })
  })

  // --- Recursion and primitives ---
  it('recurses into nested objects', () => {
    const input = {
      name: 'Alice',
      userId: { $oid: '507f1f77bcf86cd799439011' },
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      age: { $numberInt: '30' },
      nested: {
        updatedAt: { $date: '2024-06-15T12:00:00.000Z' },
      },
    }
    expect(humanizeEjson(input)).toEqual({
      name: 'Alice',
      userId: '507f1f77bcf86cd799439011',
      createdAt: '2024-01-01T00:00:00.000Z',
      age: 30,
      nested: {
        updatedAt: '2024-06-15T12:00:00.000Z',
      },
    })
  })

  it('recurses into arrays', () => {
    const input = [
      { $date: '2024-01-01T00:00:00.000Z' },
      { $numberLong: '99' },
      'plain string',
    ]
    expect(humanizeEjson(input)).toEqual(['2024-01-01T00:00:00.000Z', 99, 'plain string'])
  })

  it('handles null and undefined', () => {
    expect(humanizeEjson(null)).toBeNull()
    expect(humanizeEjson(undefined)).toBeUndefined()
  })

  it('passes through primitives unchanged', () => {
    expect(humanizeEjson('hello')).toBe('hello')
    expect(humanizeEjson(42)).toBe(42)
    expect(humanizeEjson(true)).toBe(true)
  })
})

describe('toJsExpression', () => {
  // --- EJSON types → BSON constructors ---
  it('converts $oid to ObjectId()', () => {
    expect(toJsExpression({ $oid: '507f1f77bcf86cd799439011' })).toBe(
      'ObjectId("507f1f77bcf86cd799439011")',
    )
  })

  it('converts $date string to ISODate()', () => {
    expect(toJsExpression({ $date: '2024-01-23T10:13:20.000Z' })).toBe(
      'ISODate("2024-01-23T10:13:20.000Z")',
    )
  })

  it('converts $date with $numberLong to ISODate()', () => {
    const input = { $date: { $numberLong: '1706004800000' } }
    expect(toJsExpression(input)).toBe(
      `ISODate("${new Date(1706004800000).toISOString()}")`,
    )
  })

  it('converts $numberLong to NumberLong()', () => {
    expect(toJsExpression({ $numberLong: '9007199254740993' })).toBe(
      'NumberLong("9007199254740993")',
    )
  })

  it('converts $numberInt to plain number', () => {
    expect(toJsExpression({ $numberInt: '42' })).toBe('42')
  })

  it('converts $numberDouble to plain number', () => {
    expect(toJsExpression({ $numberDouble: '3.14' })).toBe('3.14')
  })

  it('converts $numberDecimal to NumberDecimal()', () => {
    expect(toJsExpression({ $numberDecimal: '123.456' })).toBe('NumberDecimal("123.456")')
  })

  it('converts $binary UUID to UUID()', () => {
    const base64 = btoa(
      String.fromCharCode(
        ...[0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00],
      ),
    )
    expect(toJsExpression({ $binary: { base64, subType: '04' } })).toBe(
      'UUID("550e8400-e29b-41d4-a716-446655440000")',
    )
  })

  it('converts $binary generic to BinData()', () => {
    expect(toJsExpression({ $binary: { base64: 'AQID', subType: '00' } })).toBe(
      'BinData(0, "AQID")',
    )
  })

  it('converts $regularExpression to regex literal', () => {
    expect(toJsExpression({ $regularExpression: { pattern: '^test', options: 'i' } })).toBe(
      '/^test/i',
    )
  })

  it('converts $timestamp to Timestamp()', () => {
    expect(toJsExpression({ $timestamp: { t: 1700000000, i: 1 } })).toBe(
      'Timestamp(1700000000, 1)',
    )
  })

  it('converts $minKey to MinKey()', () => {
    expect(toJsExpression({ $minKey: 1 })).toBe('MinKey()')
  })

  it('converts $maxKey to MaxKey()', () => {
    expect(toJsExpression({ $maxKey: 1 })).toBe('MaxKey()')
  })

  // --- Humanized values → BSON constructors ---
  it('detects ISO date strings and wraps with ISODate()', () => {
    expect(toJsExpression('2024-01-23T10:13:20.000Z')).toBe(
      'ISODate("2024-01-23T10:13:20.000Z")',
    )
  })

  it('detects 24-char hex and wraps with ObjectId()', () => {
    expect(toJsExpression('507f1f77bcf86cd799439011')).toBe(
      'ObjectId("507f1f77bcf86cd799439011")',
    )
  })

  it('detects UUID strings and wraps with UUID()', () => {
    expect(toJsExpression('550e8400-e29b-41d4-a716-446655440000')).toBe(
      'UUID("550e8400-e29b-41d4-a716-446655440000")',
    )
  })

  it('wraps plain strings in quotes', () => {
    expect(toJsExpression('hello')).toBe('"hello"')
  })

  it('escapes special characters in strings', () => {
    expect(toJsExpression('say "hello"')).toBe('"say \\"hello\\""')
  })

  // --- Primitives ---
  it('handles numbers', () => {
    expect(toJsExpression(42)).toBe('42')
    expect(toJsExpression(3.14)).toBe('3.14')
  })

  it('handles booleans', () => {
    expect(toJsExpression(true)).toBe('true')
    expect(toJsExpression(false)).toBe('false')
  })

  it('handles null', () => {
    expect(toJsExpression(null)).toBe('null')
  })

  // --- Containers ---
  it('generates array expression', () => {
    expect(toJsExpression([1, 'hello', null])).toBe('[1, "hello", null]')
  })

  it('generates object expression', () => {
    const result = toJsExpression({ name: 'Alice', age: 30 })
    expect(result).toBe('{ name: "Alice", age: 30 }')
  })

  it('quotes keys that need quoting', () => {
    const result = toJsExpression({ 'my-key': 1, 'has space': 2, normal: 3 })
    expect(result).toBe('{ "my-key": 1, "has space": 2, normal: 3 }')
  })

  // --- Full document round-trip ---
  it('generates correct JS for a humanized document', () => {
    const humanized = {
      name: 'Alice',
      createdAt: '2024-01-01T00:00:00.000Z',
      userId: '507f1f77bcf86cd799439011',
      age: 30,
      tags: ['a', 'b'],
    }
    const result = toJsExpression(humanized)
    expect(result).toBe(
      '{ name: "Alice", createdAt: ISODate("2024-01-01T00:00:00.000Z"), userId: ObjectId("507f1f77bcf86cd799439011"), age: 30, tags: ["a", "b"] }',
    )
  })

  it('generates correct JS for a raw EJSON _id filter', () => {
    // Binary UUID _id
    const base64 = btoa(
      String.fromCharCode(
        ...[0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00],
      ),
    )
    const rawId = { $binary: { base64, subType: '04' } }
    expect(toJsExpression(rawId)).toBe('UUID("550e8400-e29b-41d4-a716-446655440000")')

    // ObjectId _id
    const oid = { $oid: '507f1f77bcf86cd799439011' }
    expect(toJsExpression(oid)).toBe('ObjectId("507f1f77bcf86cd799439011")')
  })
})
