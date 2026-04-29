import type { models } from 'wailsjs/go/models'

export const TYPE_PALETTE: Record<string, string> = {
  string: '#4f9cff',
  int: '#7bd88f',
  long: '#7bd88f',
  double: '#7bd88f',
  decimal: '#7bd88f',
  bool: '#f5a623',
  date: '#c678dd',
  objectId: '#56b6c2',
  null: '#888',
  array: '#e06c75',
  object: '#d19a66',
}

export interface Segment {
  type: string
  pct: number
  color: string
  count: number
}

export function computeSegments(
  types: models.TypeStat[],
  total: number,
  fallbackColor: string,
): Segment[] {
  return types.map((t) => ({
    type: t.type,
    pct: total > 0 ? (t.count / total) * 100 : 0,
    color: TYPE_PALETTE[t.type] ?? fallbackColor,
    count: t.count,
  }))
}
