import { describe, expect, test } from 'vitest'
import { filterMessages, formatLogLine } from '../messageFormatters'
import type { LogMessage } from '../queryStore'

function msg(level: LogMessage['level'], text: string): LogMessage {
  return { id: `${level}-${text}`, timestamp: '12:00:00', level, text }
}

describe('filterMessages', () => {
  test('keeps only enabled levels', () => {
    const result = filterMessages(
      [msg('info', 'a'), msg('warning', 'b'), msg('error', 'c')],
      { info: true, warning: false, error: true },
    )
    expect(result.map((m) => m.text)).toEqual(['a', 'c'])
  })

  test('returns empty when all levels are disabled', () => {
    const result = filterMessages(
      [msg('info', 'a'), msg('error', 'b')],
      { info: false, warning: false, error: false },
    )
    expect(result).toEqual([])
  })
})

describe('formatLogLine', () => {
  test('formats as "{ts} [{LEVEL}] {text}"', () => {
    expect(formatLogLine(msg('error', 'boom'))).toBe('12:00:00 [ERROR] boom')
    expect(formatLogLine(msg('warning', 'soft'))).toBe('12:00:00 [WARNING] soft')
    expect(formatLogLine(msg('info', 'hello'))).toBe('12:00:00 [INFO] hello')
  })
})
