import { describe, it, expect } from 'vitest'
import { humanizeEjson, dehumanizeEjson } from '../humanizeEjson'

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
    // High-precision value that loses precision as a JS number
    expect(humanizeEjson({ $numberDecimal: '1234567890.123456789012345' })).toBe(
      '1234567890.123456789012345',
    )
  })

  // --- Binary: UUID ---
  it('converts $binary UUID (subType 04) to UUID string', () => {
    // UUID: 550e8400-e29b-41d4-a716-446655440000
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

describe('dehumanizeEjson', () => {
  // --- Dates ---
  it('converts ISO date strings back to $date', () => {
    expect(dehumanizeEjson('2024-01-23T10:13:20.000Z')).toEqual({
      $date: '2024-01-23T10:13:20.000Z',
    })
  })

  it('converts ISO dates with timezone offset', () => {
    expect(dehumanizeEjson('2024-01-23T10:13:20.000+05:30')).toEqual({
      $date: '2024-01-23T10:13:20.000+05:30',
    })
  })

  it('does not convert non-ISO strings', () => {
    expect(dehumanizeEjson('hello')).toBe('hello')
    expect(dehumanizeEjson('2024-01-23')).toBe('2024-01-23')
    expect(dehumanizeEjson('not a date at all')).toBe('not a date at all')
  })

  // --- ObjectId ---
  it('converts 24-char hex strings back to $oid', () => {
    expect(dehumanizeEjson('507f1f77bcf86cd799439011')).toEqual({
      $oid: '507f1f77bcf86cd799439011',
    })
  })

  it('does not convert non-24-char hex strings', () => {
    expect(dehumanizeEjson('abcdef')).toBe('abcdef')
    expect(dehumanizeEjson('not-hex-at-all-24-chars!')).toBe('not-hex-at-all-24-chars!')
  })

  // --- UUID ---
  it('converts UUID strings back to $binary', () => {
    const uuid = '550e8400-e29b-41d4-a716-446655440000'
    const result = dehumanizeEjson(uuid) as Record<string, unknown>
    expect(result.$binary).toBeDefined()
    const binary = result.$binary as Record<string, unknown>
    expect(binary.subType).toBe('04')
    // Round-trip: humanize the result and confirm we get the UUID back
    expect(humanizeEjson(result)).toBe(uuid)
  })

  // --- Numbers ---
  it('leaves numbers as-is', () => {
    expect(dehumanizeEjson(42)).toBe(42)
    expect(dehumanizeEjson(3.14)).toBe(3.14)
  })

  // --- Recursion ---
  it('recurses into nested objects', () => {
    const input = {
      name: 'Alice',
      createdAt: '2024-01-01T00:00:00.000Z',
      userId: '507f1f77bcf86cd799439011',
      age: 30,
    }
    expect(dehumanizeEjson(input)).toEqual({
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      userId: { $oid: '507f1f77bcf86cd799439011' },
      age: 30,
    })
  })

  it('recurses into arrays', () => {
    const input = ['2024-01-01T00:00:00.000Z', 99, 'plain string']
    expect(dehumanizeEjson(input)).toEqual([
      { $date: '2024-01-01T00:00:00.000Z' },
      99,
      'plain string',
    ])
  })

  it('handles null and undefined', () => {
    expect(dehumanizeEjson(null)).toBeNull()
    expect(dehumanizeEjson(undefined)).toBeUndefined()
  })

  // --- Round-trip ---
  it('round-trips with humanizeEjson for common types', () => {
    const original = {
      _id: { $oid: '507f1f77bcf86cd799439011' },
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      age: { $numberInt: '30' },
      score: { $numberDouble: '95.5' },
      tags: ['a', 'b'],
    }
    const humanized = humanizeEjson(original)
    const restored = dehumanizeEjson(humanized)

    // $oid round-trips correctly
    expect(restored).toEqual({
      _id: { $oid: '507f1f77bcf86cd799439011' },
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      age: 30,
      score: 95.5,
      tags: ['a', 'b'],
    })
  })

  it('round-trips UUID through humanize/dehumanize', () => {
    const original = {
      $binary: { base64: btoa(String.fromCharCode(
        ...[0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00],
      )), subType: '04' },
    }
    const humanized = humanizeEjson(original)
    expect(humanized).toBe('550e8400-e29b-41d4-a716-446655440000')
    const restored = dehumanizeEjson(humanized)
    // Re-humanize to verify round-trip
    expect(humanizeEjson(restored)).toBe('550e8400-e29b-41d4-a716-446655440000')
  })
})
