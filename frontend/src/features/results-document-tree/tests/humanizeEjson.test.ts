import { describe, it, expect } from 'vitest'
import { humanizeEjson } from '../humanizeEjson'

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

describe('humanizeEjson with documents that only look like BSON wrappers', () => {
  it('keeps a document that has a field named like a wrapper', () => {
    const doc = { _id: 1, name: 'x', $date: '2024-01-01' }

    expect(humanizeEjson(doc)).toEqual(doc)
  })

  it.each([[{ $regularExpression: null }], [{ $binary: null }]])('does not throw on %j', (doc) => {
    expect(humanizeEjson(doc)).toEqual(doc)
  })

  it('keeps a UUID with invalid base64 instead of throwing', () => {
    const value = { $binary: { base64: '!!!', subType: '04' } }

    expect(humanizeEjson(value)).toEqual(value)
  })
})

describe('humanizeEjson with dates beyond a JS Date', () => {
  it('keeps the date as EJSON, so it does not become a string', () => {
    const value = { $date: { $numberLong: '99999999999999999' } }

    expect(humanizeEjson(value)).toEqual(value)
  })

  it('still converts a date in range', () => {
    expect(humanizeEjson({ $date: { $numberLong: '0' } })).toBe('1970-01-01T00:00:00.000Z')
  })
})
