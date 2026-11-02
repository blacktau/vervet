export interface DocumentRow {
  key: string
  field: string
  value: string
  type: string
  typeLabel: string
  isDocRoot: boolean
  /** A top-level result that isn't a document, such as a string or an ObjectId */
  isRootValue: boolean
  children?: DocumentRow[]
}
