import { beforeEach, describe, expect, test } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useExportStore } from '../exportStore'

describe('exportStore', () => {
  beforeEach(() => setActivePinia(createPinia()))

  test('defaults to CSV with comma separator, header on, BOM off', () => {
    const s = useExportStore()
    expect(s.format).toBe('csv')
    expect(s.csv.separator).toBe(',')
    expect(s.csv.includeHeader).toBe(true)
    expect(s.csv.utf8Bom).toBe(false)
  })

  test('remembers the last-chosen options', () => {
    const s = useExportStore()
    s.setFormat('ndjson')
    s.setCsv({ separator: '\t', includeHeader: false, utf8Bom: true })
    expect(s.format).toBe('ndjson')
    expect(s.csv.separator).toBe('\t')
    expect(s.csv.includeHeader).toBe(false)
    expect(s.csv.utf8Bom).toBe(true)
  })
})
