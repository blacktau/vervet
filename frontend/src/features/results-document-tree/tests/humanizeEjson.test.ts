import { describe, it, expect } from 'vitest'
import { humanizeEjson, dehumanizeEjson } from '../humanizeEjson'

describe('humanizeEjson', () => {
  it('converts $date to ISO string', () => {
    expect(humanizeEjson({ $date: '2024-01-23T10:13:20.000Z' })).toBe('2024-01-23T10:13:20.000Z')
  })

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

  it('preserves $oid as-is', () => {
    const oid = { $oid: '507f1f77bcf86cd799439011' }
    expect(humanizeEjson(oid)).toEqual(oid)
  })

  it('recurses into nested objects', () => {
    const input = {
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      age: { $numberInt: '30' },
      nested: {
        updatedAt: { $date: '2024-06-15T12:00:00.000Z' },
      },
    }
    expect(humanizeEjson(input)).toEqual({
      name: 'Alice',
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

  it('leaves numbers as-is', () => {
    expect(dehumanizeEjson(42)).toBe(42)
    expect(dehumanizeEjson(3.14)).toBe(3.14)
  })

  it('recurses into nested objects', () => {
    const input = {
      name: 'Alice',
      createdAt: '2024-01-01T00:00:00.000Z',
      age: 30,
    }
    expect(dehumanizeEjson(input)).toEqual({
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
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

  it('round-trips with humanizeEjson', () => {
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

    // _id ($oid) is preserved as-is through both transforms
    expect(restored).toEqual({
      _id: { $oid: '507f1f77bcf86cd799439011' },
      name: 'Alice',
      createdAt: { $date: '2024-01-01T00:00:00.000Z' },
      age: 30,
      score: 95.5,
      tags: ['a', 'b'],
    })
  })
})
