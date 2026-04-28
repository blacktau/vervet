export const MAX_INLINE_CHARS = 600
export const MAX_INLINE_LINES = 8

export interface DetailSummary {
  head: string
  truncated: boolean
}

export function summariseDetail(detail: string | undefined): DetailSummary {
  if (!detail) {
    return { head: '', truncated: false }
  }

  const lines = detail.split('\n')
  const exceedsLines = lines.length > MAX_INLINE_LINES
  const exceedsChars = detail.length > MAX_INLINE_CHARS

  if (!exceedsLines && !exceedsChars) {
    return { head: detail, truncated: false }
  }

  const headByLines = lines.slice(0, MAX_INLINE_LINES).join('\n')
  const head = headByLines.length > MAX_INLINE_CHARS
    ? headByLines.slice(0, MAX_INLINE_CHARS)
    : headByLines

  return { head, truncated: true }
}
