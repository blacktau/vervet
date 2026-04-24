import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { buildDefaultFilename } from '../defaultFilename'

describe('buildDefaultFilename', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-24T14:30:22'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('uses collection name + timestamp for csv', () => {
    expect(buildDefaultFilename('users', 'csv')).toBe('users-20260424-143022.csv')
  })
  test('uses collection name + timestamp for json', () => {
    expect(buildDefaultFilename('users', 'json')).toBe('users-20260424-143022.json')
  })
  test('uses collection name + timestamp for ndjson', () => {
    expect(buildDefaultFilename('users', 'ndjson')).toBe('users-20260424-143022.ndjson')
  })
  test('falls back to vervet-export when collection missing', () => {
    expect(buildDefaultFilename(undefined, 'csv')).toBe('vervet-export-20260424-143022.csv')
    expect(buildDefaultFilename('', 'csv')).toBe('vervet-export-20260424-143022.csv')
  })
})
