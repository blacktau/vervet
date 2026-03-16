export interface DocumentRow {
  key: string
  field: string
  value: string
  type: string
  typeLabel: string
  isDocRoot: boolean
  children?: DocumentRow[]
}
