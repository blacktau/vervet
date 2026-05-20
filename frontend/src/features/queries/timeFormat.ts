/**
 * Format an elapsed duration in milliseconds as a clock-style string.
 *
 * - Under 1 hour: `m:ss` (seconds zero-padded to 2)
 * - 1 hour or more: `h:mm:ss` (minutes and seconds zero-padded to 2)
 * - Negative or NaN input: `"0:00"`
 * - Sub-second remainder is truncated (floored), never rounded up.
 */
export function formatElapsed(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return '0:00'
  }

  const totalSeconds = Math.floor(ms / 1000)
  const seconds = totalSeconds % 60
  const totalMinutes = Math.floor(totalSeconds / 60)
  const minutes = totalMinutes % 60
  const hours = Math.floor(totalMinutes / 60)

  const ss = seconds.toString().padStart(2, '0')

  if (hours > 0) {
    const mm = minutes.toString().padStart(2, '0')
    return `${hours}:${mm}:${ss}`
  }

  return `${totalMinutes}:${ss}`
}
