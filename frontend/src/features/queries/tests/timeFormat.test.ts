import { describe, expect, test } from 'vitest'
import { formatElapsed } from '../timeFormat'

describe('formatElapsed', () => {
  test('zero ms returns 0:00', () => {
    expect(formatElapsed(0)).toBe('0:00')
  })

  test('under one second truncates to 0:00', () => {
    expect(formatElapsed(500)).toBe('0:00')
  })

  test('under a minute pads seconds (1500ms -> 0:01)', () => {
    expect(formatElapsed(1500)).toBe('0:01')
  })

  test('exact minute returns 1:00', () => {
    expect(formatElapsed(60000)).toBe('1:00')
  })

  test('minutes and seconds (65000ms -> 1:05)', () => {
    expect(formatElapsed(65000)).toBe('1:05')
  })

  test('just under an hour returns 59:59', () => {
    expect(formatElapsed(3599000)).toBe('59:59')
  })

  test('nine minutes fifty-nine seconds (599000ms -> 9:59)', () => {
    expect(formatElapsed(599000)).toBe('9:59')
  })

  test('exact hour returns 1:00:00', () => {
    expect(formatElapsed(3600000)).toBe('1:00:00')
  })

  test('hour minutes and seconds (3661000ms -> 1:01:01)', () => {
    expect(formatElapsed(3661000)).toBe('1:01:01')
  })

  test('truncates sub-second remainder (no rounding up)', () => {
    expect(formatElapsed(1999)).toBe('0:01')
    expect(formatElapsed(59999)).toBe('0:59')
  })

  test('NaN returns 0:00', () => {
    expect(formatElapsed(NaN)).toBe('0:00')
  })

  test('negative returns 0:00', () => {
    expect(formatElapsed(-1000)).toBe('0:00')
  })
})
