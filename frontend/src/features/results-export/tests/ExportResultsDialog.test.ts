import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { separatorFromChoice, buildExportPayload } from '../exportDialogHelpers'

vi.mock('wailsjs/go/api/ExportProxy', () => ({
  ExportResults: vi.fn().mockResolvedValue({ isSuccess: true, data: '/tmp/out.csv' }),
}))

vi.mock('@/utils/dialog.ts', () => ({
  useNotifier: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
}))

describe('separatorFromChoice', () => {
  test('returns comma for "comma"', () => {
    expect(separatorFromChoice('comma', '')).toBe(',')
  })

  test('returns tab for "tab"', () => {
    expect(separatorFromChoice('tab', '')).toBe('\t')
  })

  test('returns semicolon for "semicolon"', () => {
    expect(separatorFromChoice('semicolon', '')).toBe(';')
  })

  test('returns pipe for "pipe"', () => {
    expect(separatorFromChoice('pipe', '')).toBe('|')
  })

  test('returns custom value for "custom"', () => {
    expect(separatorFromChoice('custom', '|')).toBe('|')
    expect(separatorFromChoice('custom', '#')).toBe('#')
  })

  test('falls back to comma when custom value is empty', () => {
    expect(separatorFromChoice('custom', '')).toBe(',')
  })

  test('slices custom to first character only', () => {
    expect(separatorFromChoice('custom', 'abc')).toBe('a')
  })
})

describe('buildExportPayload', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('builds CSV payload with csv field populated', () => {
    const payload = buildExportPayload({
      format: 'csv',
      ejson: '[{"a":1}]',
      collectionName: 'users',
      defaultFilename: 'users-20260424-143022.csv',
      isCsv: true,
      separator: ',',
      includeHeader: true,
      utf8Bom: false,
    })

    expect(payload.format).toBe('csv')
    expect(payload.ejson).toBe('[{"a":1}]')
    expect(payload.collectionName).toBe('users')
    expect(payload.defaultFilename).toBe('users-20260424-143022.csv')
    expect(payload.csv).toEqual({ separator: ',', includeHeader: true, utf8Bom: false })
  })

  test('builds JSON payload without csv field', () => {
    const payload = buildExportPayload({
      format: 'json',
      ejson: '[{"a":1}]',
      collectionName: 'logs',
      defaultFilename: 'logs-20260424-143022.json',
      isCsv: false,
      separator: ',',
      includeHeader: true,
      utf8Bom: false,
    })

    expect(payload.format).toBe('json')
    expect(payload.csv).toBeUndefined()
  })

  test('builds NDJSON payload without csv field', () => {
    const payload = buildExportPayload({
      format: 'ndjson',
      ejson: '[{"a":1}]',
      collectionName: 'events',
      defaultFilename: 'events-20260424-143022.ndjson',
      isCsv: false,
      separator: ',',
      includeHeader: false,
      utf8Bom: false,
    })

    expect(payload.format).toBe('ndjson')
    expect(payload.csv).toBeUndefined()
  })

  test('uses empty string for collectionName when undefined', () => {
    const payload = buildExportPayload({
      format: 'csv',
      ejson: '[]',
      collectionName: undefined,
      defaultFilename: 'vervet-export-20260424-143022.csv',
      isCsv: true,
      separator: '\t',
      includeHeader: false,
      utf8Bom: true,
    })

    expect(payload.collectionName).toBe('')
    expect(payload.csv).toEqual({ separator: '\t', includeHeader: false, utf8Bom: true })
  })
})
