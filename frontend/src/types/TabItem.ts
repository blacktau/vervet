export type TabItem = {
  name: string
  title: string
  blank: boolean
  subTabIdx?: number
  subTab?: string
  icon?: string
  server?: string
  loading: boolean
  connectionId: string
  database?: string
  value?: unknown
}
