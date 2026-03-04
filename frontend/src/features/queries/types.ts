export interface FlatRow {
  key: string
  field: string
  value: string
  type: string
  depth: number
  hasChildren: boolean
  expanded: boolean
  childKeys: string[]
  isDocRoot: boolean
}
