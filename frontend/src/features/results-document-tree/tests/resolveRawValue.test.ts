import { describe, it, expect } from 'vitest'
import { resolveRawValue } from '../resolveRawValue'

describe('resolveRawValue', () => {
  const documents = [
    {
      _id: { $oid: 'abc123' },
      name: 'Alice',
      address: {
        city: 'London',
        codes: [10, 20, 30],
      },
      tags: ['admin', 'user'],
    },
    {
      _id: { $oid: 'def456' },
      name: 'Bob',
    },
  ]

  it('returns full document for root key', () => {
    expect(resolveRawValue(documents, '__doc_0')).toBe(documents[0])
  })

  it('returns full document for second root key', () => {
    expect(resolveRawValue(documents, '__doc_1')).toBe(documents[1])
  })

  it('returns top-level field value', () => {
    expect(resolveRawValue(documents, '__doc_0.name')).toBe('Alice')
  })

  it('returns nested object field', () => {
    expect(resolveRawValue(documents, '__doc_0.address.city')).toBe('London')
  })

  it('returns array element', () => {
    expect(resolveRawValue(documents, '__doc_0.tags.0')).toBe('admin')
  })

  it('returns nested array element', () => {
    expect(resolveRawValue(documents, '__doc_0.address.codes.1')).toBe(20)
  })

  it('returns BSON extended JSON object', () => {
    expect(resolveRawValue(documents, '__doc_0._id')).toEqual({ $oid: 'abc123' })
  })

  it('returns undefined for invalid doc index', () => {
    expect(resolveRawValue(documents, '__doc_99')).toBeUndefined()
  })

  it('returns undefined for invalid field path', () => {
    expect(resolveRawValue(documents, '__doc_0.nonexistent')).toBeUndefined()
  })

  it('returns undefined for empty documents', () => {
    expect(resolveRawValue([], '__doc_0')).toBeUndefined()
  })
})
