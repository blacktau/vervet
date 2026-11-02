export interface DocumentRow {
  key: string
  field: string
  value: string
  type: string
  typeLabel: string
  /**
   * document: a top-level document.
   * value: a top-level result that isn't a document, such as a string or an array.
   * field: anything nested inside those.
   */
  kind: 'document' | 'value' | 'field'
  children?: DocumentRow[]
}
