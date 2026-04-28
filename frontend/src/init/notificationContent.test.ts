import { describe, it, expect } from 'vitest'
import { summariseDetail, MAX_INLINE_CHARS, MAX_INLINE_LINES } from './notificationContent'

describe('summariseDetail', () => {
  it('returns the input unchanged when short and few lines', () => {
    const result = summariseDetail('connection refused')
    expect(result).toEqual({ head: 'connection refused', truncated: false })
  })

  it('returns truncated=false for empty/undefined input', () => {
    expect(summariseDetail('')).toEqual({ head: '', truncated: false })
    expect(summariseDetail(undefined)).toEqual({ head: '', truncated: false })
  })

  it('flags truncated when detail exceeds the character threshold', () => {
    const long = 'x'.repeat(MAX_INLINE_CHARS + 50)
    const result = summariseDetail(long)
    expect(result.truncated).toBe(true)
    expect(result.head.length).toBeLessThanOrEqual(MAX_INLINE_CHARS)
    expect(long.startsWith(result.head)).toBe(true)
  })

  it('flags truncated when detail exceeds the line threshold', () => {
    const lines = Array.from({ length: MAX_INLINE_LINES + 3 }, (_, i) => `line ${i}`).join('\n')
    const result = summariseDetail(lines)
    expect(result.truncated).toBe(true)
    expect(result.head.split('\n').length).toBeLessThanOrEqual(MAX_INLINE_LINES)
  })

  it('does not truncate at exactly the line threshold', () => {
    const lines = Array.from({ length: MAX_INLINE_LINES }, (_, i) => `line ${i}`).join('\n')
    const result = summariseDetail(lines)
    expect(result.truncated).toBe(false)
    expect(result.head).toBe(lines)
  })
})
