import { describe, expect, test } from 'vitest'
import {
  readCollectionImpact,
  readDbImpact,
  shouldEscalateCollectionDrop,
} from '@/features/data-browser/statsImpact.ts'

describe('readDbImpact', () => {
  test('extracts collections and objects when both are numbers', () => {
    expect(readDbImpact({ collections: 14, objects: 412532 })).toEqual({
      collectionCount: 14,
      documentCount: 412532,
    })
  })

  test('returns undefined fields when values are missing', () => {
    expect(readDbImpact({})).toEqual({
      collectionCount: undefined,
      documentCount: undefined,
    })
  })

  test('returns undefined fields when values are not numbers', () => {
    expect(readDbImpact({ collections: '14', objects: null })).toEqual({
      collectionCount: undefined,
      documentCount: undefined,
    })
  })

  test('handles only one field being present', () => {
    expect(readDbImpact({ collections: 3 })).toEqual({
      collectionCount: 3,
      documentCount: undefined,
    })
  })
})

describe('readCollectionImpact', () => {
  test('extracts count when it is a number', () => {
    expect(readCollectionImpact({ count: 42 })).toEqual({ documentCount: 42 })
  })

  test('returns undefined when count is missing', () => {
    expect(readCollectionImpact({})).toEqual({ documentCount: undefined })
  })

  test('returns undefined when count is not a number', () => {
    expect(readCollectionImpact({ count: 'many' })).toEqual({ documentCount: undefined })
  })
})

describe('shouldEscalateCollectionDrop', () => {
  test('does not escalate for views', () => {
    expect(shouldEscalateCollectionDrop({ isView: true, documentCount: 1_000_000 })).toBe(false)
  })

  test('does not escalate for empty collections', () => {
    expect(shouldEscalateCollectionDrop({ isView: false, documentCount: 0 })).toBe(false)
  })

  test('escalates for non-empty collections', () => {
    expect(shouldEscalateCollectionDrop({ isView: false, documentCount: 1 })).toBe(true)
  })

  test('escalates when count is unknown (fail-closed)', () => {
    expect(shouldEscalateCollectionDrop({ isView: false, documentCount: undefined })).toBe(true)
  })
})
