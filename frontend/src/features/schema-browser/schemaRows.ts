import type { models } from 'wailsjs/go/models'

export interface SchemaRow {
  key: string
  name: string
  path: string
  count: number
  presence: number
  types: models.TypeStat[]
  hasChildren: boolean
  children?: SchemaRow[]
}

export function buildSchemaRows(
  fields: models.FieldInfo[],
  sampledCount: number,
): SchemaRow[] {
  return fields.map((f) => ({
    key: f.path,
    name: f.name,
    path: f.path,
    count: f.count,
    presence: sampledCount > 0 ? (f.count / sampledCount) * 100 : 0,
    types: f.types,
    hasChildren: !!(f.children && f.children.length > 0),
    children:
      f.children && f.children.length > 0
        ? buildSchemaRows(f.children, sampledCount)
        : undefined,
  }))
}
