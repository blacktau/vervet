import { type BrowserSubTabType } from '@/consts/BrowserSubTabType'

export type QueryTabItem = {
  id: string
  database: string
  initialText?: string
}

export type IndexTabItem = {
  id: string
  serverId: string
  dbName: string
  collectionName: string
}

export type ServerTabItem = {
  title: string
  blank: boolean
  icon?: string
  serverId: string
  subTab: BrowserSubTabType
  queryOpen?: boolean
  queries: QueryTabItem[]
  activeQueryId?: string
  indexTabs?: IndexTabItem[]
  activeIndexTabId?: string
}
