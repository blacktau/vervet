import { parse } from '@babel/parser'
import type { editor } from 'monaco-editor'

const SHELL_SUGAR_LINE = /^\s*(?:show\s+\w+|use\s+\w+|it)\s*;?\s*$/

function preprocess(source: string): string {
  return source
    .split('\n')
    .map((line) => (SHELL_SUGAR_LINE.test(line) ? ' '.repeat(line.length) : line))
    .join('\n')
}

export function validate(source: string): editor.IMarkerData[] {
  const masked = preprocess(source)
  try {
    const ast = parse(masked, {
      errorRecovery: true,
      sourceType: 'script',
      allowReturnOutsideFunction: true,
      allowAwaitOutsideFunction: true,
      allowImportExportEverywhere: false,
    })
    const errors = (ast as unknown as { errors?: Array<{ loc?: { line: number; column: number }; message: string }> }).errors ?? []
    return errors.map(toMarker)
  } catch (err) {
    const e = err as { loc?: { line: number; column: number }; message?: string }
    if (!e.loc) {
      return []
    }
    return [toMarker({ loc: e.loc, message: e.message ?? 'Syntax error' })]
  }
}

function toMarker(e: { loc?: { line: number; column: number }; message: string }): editor.IMarkerData {
  const line = e.loc?.line ?? 1
  const col = (e.loc?.column ?? 0) + 1
  return {
    severity: 8, // monaco.MarkerSeverity.Error — hardcoded to avoid importing the runtime in tests
    message: e.message,
    source: 'vervet',
    startLineNumber: line,
    startColumn: col,
    endLineNumber: line,
    endColumn: col + 1,
  }
}
