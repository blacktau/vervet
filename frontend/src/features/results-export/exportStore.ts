import { defineStore } from 'pinia'
import type { ExportFormat } from './defaultFilename'

export interface CsvOptions {
  separator: string
  includeHeader: boolean
  utf8Bom: boolean
}

interface State {
  format: ExportFormat
  csv: CsvOptions
}

export const useExportStore = defineStore('export', {
  state: (): State => ({
    format: 'csv',
    csv: { separator: ',', includeHeader: true, utf8Bom: false },
  }),
  actions: {
    setFormat(f: ExportFormat) {
      this.format = f
    },
    setCsv(c: CsvOptions) {
      this.csv = { ...c }
    },
  },
})
