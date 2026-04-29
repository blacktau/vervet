import { describe, it, expect } from 'vitest'
import { computeSegments, TYPE_PALETTE } from '../typeBarHelpers'

describe('computeSegments', () => {
  it('returns one segment per type with correct percent', () => {
    const segs = computeSegments(
      [
        { type: 'string', count: 80 },
        { type: 'int', count: 20 },
      ],
      100,
      '#000',
    )
    expect(segs).toHaveLength(2)
    expect(segs[0]!.pct).toBe(80)
    expect(segs[1]!.pct).toBe(20)
  })

  it('uses palette color when type known', () => {
    const segs = computeSegments([{ type: 'string', count: 1 }], 1, '#000')
    expect(segs[0]!.color).toBe(TYPE_PALETTE.string)
  })

  it('falls back to fallbackColor for unknown type', () => {
    const segs = computeSegments([{ type: 'unknown-type-xyz', count: 1 }], 1, '#abc')
    expect(segs[0]!.color).toBe('#abc')
  })

  it('returns 0 percent when total is 0', () => {
    const segs = computeSegments([{ type: 'string', count: 5 }], 0, '#000')
    expect(segs[0]!.pct).toBe(0)
  })
})
