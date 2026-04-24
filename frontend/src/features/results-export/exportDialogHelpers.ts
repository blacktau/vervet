import type { ExportFormat } from './defaultFilename'

export type SeparatorChoice = 'comma' | 'tab' | 'semicolon' | 'pipe' | 'custom'

export interface ExportPayloadOptions {
  format: ExportFormat
  ejson: string
  collectionName: string | undefined
  defaultFilename: string
  isCsv: boolean
  separator: string
  includeHeader: boolean
  utf8Bom: boolean
}

export interface ExportPayload {
  format: string
  ejson: string
  collectionName: string
  defaultFilename: string
  csv?: {
    separator: string
    includeHeader: boolean
    utf8Bom: boolean
  }
}

export function separatorFromChoice(choice: SeparatorChoice, custom: string): string {
  switch (choice) {
    case 'comma':
      return ','
    case 'tab':
      return '\t'
    case 'semicolon':
      return ';'
    case 'pipe':
      return '|'
    case 'custom':
      return custom.slice(0, 1) || ','
  }
}

export function separatorChoiceFromValue(separator: string): SeparatorChoice {
  switch (separator) {
    case ',':
      return 'comma'
    case '\t':
      return 'tab'
    case ';':
      return 'semicolon'
    case '|':
      return 'pipe'
    default:
      return 'custom'
  }
}

export function buildExportPayload(opts: ExportPayloadOptions): ExportPayload {
  return {
    format: opts.format,
    ejson: opts.ejson,
    collectionName: opts.collectionName ?? '',
    defaultFilename: opts.defaultFilename,
    csv: opts.isCsv
      ? {
          separator: opts.separator,
          includeHeader: opts.includeHeader,
          utf8Bom: opts.utf8Bom,
        }
      : undefined,
  }
}
