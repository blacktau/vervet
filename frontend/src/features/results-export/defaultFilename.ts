export type ExportFormat = 'csv' | 'json' | 'ndjson'

export function buildDefaultFilename(
  collection: string | undefined,
  format: ExportFormat,
): string {
  const base = collection && collection.length > 0 ? collection : 'vervet-export'
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const ts =
    `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}` +
    `-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  return `${base}-${ts}.${format}`
}
