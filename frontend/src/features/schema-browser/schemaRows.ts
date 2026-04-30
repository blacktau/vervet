import type { models } from 'wailsjs/go/models'

export interface SchemaRow {
  key: string
  name: string
  path: string
  count: number
  isArrayElement: boolean
  presence: number
  avgPerParent: number
  types: models.TypeStat[]
  hasChildren: boolean
  children?: SchemaRow[]
}

export function buildSchemaRows(
  fields: models.FieldInfo[],
  sampledCount: number,
  parentCount: number = sampledCount,
): SchemaRow[] {
  return fields.map((f) => {
    const isArrayElement = f.name === '[]'
    const presence = isArrayElement
      ? 0
      : sampledCount > 0
        ? Math.min((f.count / sampledCount) * 100, 100)
        : 0
    const avgPerParent =
      isArrayElement && parentCount > 0 ? f.count / parentCount : 0
    return {
      key: f.path,
      name: f.name,
      path: f.path,
      count: f.count,
      isArrayElement,
      presence,
      avgPerParent,
      types: f.types,
      hasChildren: !!(f.children && f.children.length > 0),
      children:
        f.children && f.children.length > 0
          ? buildSchemaRows(f.children, sampledCount, f.count)
          : undefined,
    }
  })
}
