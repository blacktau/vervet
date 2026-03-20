export type QueryTabItem = {
  id: string
  database: string
  initialText?: string
  collectionName?: string
  filePath?: string
}

export type IndexTabItem = {
  id: string
  serverId: string
  dbName: string
  collectionName: string
}

export type StatisticsTabItem = {
  id: string
  serverId: string
  dbName: string
  collectionName: string
  level: 'collection' | 'database' | 'server'
}

export type ServerTabItem = {
  title: string
  blank: boolean
  icon?: string
  serverId: string
  queries: QueryTabItem[]
  indexTabs?: IndexTabItem[]
  statisticsTabs?: StatisticsTabItem[]
  activeInnerTabId?: string
}
