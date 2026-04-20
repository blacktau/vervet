export type DbImpact = {
  collectionCount?: number
  documentCount?: number
}

export type CollectionImpact = {
  documentCount?: number
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

export function readDbImpact(stats: Record<string, unknown>): DbImpact {
  return {
    collectionCount: asNumber(stats.collections),
    documentCount: asNumber(stats.objects),
  }
}

export function readCollectionImpact(stats: Record<string, unknown>): CollectionImpact {
  return {
    documentCount: asNumber(stats.count),
  }
}

export function shouldEscalateCollectionDrop(input: {
  isView: boolean
  documentCount: number | undefined
}): boolean {
  if (input.isView) {
    return false
  }
  if (input.documentCount === 0) {
    return false
  }
  return true
}
