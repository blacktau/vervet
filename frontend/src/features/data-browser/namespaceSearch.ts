import { type DataNodeType } from '@/features/data-browser/types.ts'

export type NamespaceRow = {
  serverID: string
  db: string
  name: string
  type: DataNodeType
  // What matching scores against: `db.name` for a collection or view, and the
  // bare db for a database, so a database is not matched against `users.users`.
  path: string
}

const DEFAULT_LIMIT = 50

// Scoring weights. Contiguity matters most: a user typing "orders" means the
// collection called orders, not one whose letters happen to appear in order.
const CONTIGUOUS_BONUS = 8
const BOUNDARY_BONUS = 6
const LEADING_PENALTY = 0.5

const isBoundary = (target: string, index: number): boolean => {
  if (index === 0) {
    return true
  }
  const previous = target[index - 1]!
  if (previous === '.' || previous === '_' || previous === '-') {
    return true
  }
  const current = target[index]!
  return previous === previous.toLowerCase() && current !== current.toLowerCase()
}

// Returns a score for query as a subsequence of target, or null if it is not
// one. Higher is better. An empty query scores 0 so callers can treat it as a
// trivial match rather than a special case.
export const scoreMatch = (query: string, target: string): number | null => {
  if (query.length === 0) {
    return 0
  }

  const lowerQuery = query.toLowerCase()
  const lowerTarget = target.toLowerCase()

  let score = 0
  let targetIndex = 0
  let previousMatchIndex = -1

  for (const char of lowerQuery) {
    const found = lowerTarget.indexOf(char, targetIndex)
    if (found === -1) {
      return null
    }

    if (previousMatchIndex !== -1 && found === previousMatchIndex + 1) {
      score += CONTIGUOUS_BONUS
    }
    if (isBoundary(target, found)) {
      score += BOUNDARY_BONUS
    }

    previousMatchIndex = found
    targetIndex = found + 1
  }

  // Prefer matches that start early in the target.
  const firstMatchPenalty = lowerTarget.indexOf(lowerQuery[0]!) * LEADING_PENALTY
  return score - firstMatchPenalty
}

export const searchNamespaces = (
  query: string,
  rows: NamespaceRow[],
  limit: number = DEFAULT_LIMIT,
): NamespaceRow[] => {
  const trimmed = query.trim()
  if (trimmed.length === 0) {
    return []
  }

  const scored: { row: NamespaceRow; score: number; length: number }[] = []
  for (const row of rows) {
    const score = scoreMatch(trimmed, row.path)
    if (score === null) {
      continue
    }
    scored.push({ row, score, length: row.path.length })
  }

  scored.sort((a, b) => {
    if (b.score !== a.score) {
      return b.score - a.score
    }
    return a.length - b.length
  })

  return scored.slice(0, limit).map((entry) => entry.row)
}
