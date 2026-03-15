import { defineStore } from 'pinia'
import { type IndexTabItem, type QueryTabItem, type ServerTabItem, type StatisticsTabItem } from '@/types/ServerTabItem.ts'
import { useDialoger } from '@/utils/dialog.ts'
import { i18nGlobal } from '@/i18n'
import { findIndex } from 'lodash'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { BrowserSubTabType } from '@/consts/BrowserSubTabType.ts'

interface TabStoreState {
  nav: NavType
  asideWidth: number
  tabItems: ServerTabItem[]
  activeTabIndex: number
}

type TabStoreGetters = {
  tabs: () => ServerTabItem[]
  currentTab: () => ServerTabItem | undefined
  currentTabId: () => string
  currentSubTab: () => BrowserSubTabType | undefined
  currentQueries: () => QueryTabItem[]
  currentIndexTabs: () => IndexTabItem[]
  currentStatisticsTabs: () => StatisticsTabItem[]
  currentStatisticsTab: () => StatisticsTabItem | undefined
}

type TabUpsertOptions = ServerTabItem & {
  format?: string
  collection?: string
  query?: string
  forceSwitch?: boolean
}

export const enum NavType {
  Servers = 'servers',
  Browser = 'browser',
}

let queryIdCounter = 0
let indexTabIdCounter = 0
let statisticsTabIdCounter = 0

export const useTabStore = defineStore('tabs', {
  state: (): TabStoreState => ({
    nav: NavType.Servers,
    asideWidth: 300,
    tabItems: [],
    activeTabIndex: 0,
  }),
  getters: {
    tabs(state: TabStoreState) {
      return state.tabItems
    },
    currentTab(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]
    },
    currentTabId(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]?.serverId
    },
    currentSubTab(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]?.subTab
    },
    currentQueries(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]?.queries ?? []
    },
    currentIndexTabs(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]?.indexTabs ?? ([] as IndexTabItem[])
    },
    currentStatisticsTabs(state: TabStoreState) {
      return state.tabItems[state.activeTabIndex]?.statisticsTabs ?? ([] as StatisticsTabItem[])
    },
    currentStatisticsTab(state: TabStoreState) {
      const tab = state.tabItems[state.activeTabIndex]
      if (!tab?.statisticsTabs || !tab.activeStatisticsTabId) {
        return undefined
      }
      return tab.statisticsTabs.find((t) => t.id === tab.activeStatisticsTabId)
    },
  } as TabStoreGetters,
  actions: {
    _setActivatedIndex(index: number, switchNav: boolean) {
      this.activeTabIndex = index
      if (switchNav) {
        this.nav = index >= 0 ? NavType.Browser : NavType.Servers
        return
      }
      if (index < 0) {
        this.nav = NavType.Servers
      }
    },
    closeTab(serverId: string) {
      const d = useDialoger()
      const tab = this.tabItems.find((x) => x.serverId === serverId)
      if (tab == null) {
        return
      }
      d.warning(i18nGlobal.t('common.dialog.closeConfirm', { name: tab.title }), async () => {
        const connectionStore = useDataBrowserStore()
        await connectionStore.disconnect(tab.serverId)
      })
    },
    upsertTab: function (options: TabUpsertOptions) {
      let tabIndex = findIndex(this.tabItems, { serverId: options.serverId })
      if (tabIndex === -1) {
        const tabItem: ServerTabItem = {
          serverId: options.serverId,
          title: options.title,
          blank: false,
          subTab: BrowserSubTabType.Query,
          queryOpen: false,
          queries: [],
          indexTabs: [],
          statisticsTabs: [],
        }
        this.tabItems.push(tabItem)
        tabIndex = this.tabItems.length - 1
        this._setActivatedIndex(tabIndex, true)
        return
      }

      const tab = this.tabItems[tabIndex]
      if (tab == null) {
        return
      }

      tab.blank = false
      tab.title = options.title
      tab.serverId = options.serverId

      if (options.forceSwitch === true) {
        this._setActivatedIndex(tabIndex, true)
      }
    },
    removeTabById(serverId: string) {
      const tabIndex = findIndex(this.tabItems, { serverId: serverId })
      this.removeTab(tabIndex)
    },
    removeTab(tabIndex: number) {
      const len = this.tabItems.length
      if (len === 1 && this.tabs[0]?.blank) {
        return undefined
      }

      if (tabIndex < 0 || tabIndex >= len) {
        return undefined
      }

      const removed = this.tabItems.splice(tabIndex, 1)
      this.activeTabIndex -= 1
      if (this.activeTabIndex < 0) {
        this._setActivatedIndex(this.tabs.length > 0 ? 0 : -1, false)
      } else {
        this._setActivatedIndex(this.activeTabIndex, false)
      }

      return removed.length > 0 ? removed[0] : undefined
    },
    removeAllTabs() {
      this.tabItems = []
      this._setActivatedIndex(-1, false)
    },

    switchSubTab(name: BrowserSubTabType) {
      const tab = this.tabItems[this.activeTabIndex]
      if (tab == null) {
        return
      }
      tab.subTab = name
    },

    setActiveTab(tab: ServerTabItem) {
      this.activeTabIndex = this.tabItems.indexOf(tab)
    },

    setActiveTabIndex(index: number) {
      this.activeTabIndex = index
    },

    openQuery(serverId: string, database: string, initialText?: string) {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab) {
        return
      }

      const queryItem: QueryTabItem = {
        id: `query-${++queryIdCounter}`,
        database,
        initialText,
      }

      tab.queries.push(queryItem)
      tab.queryOpen = true
      tab.activeQueryId = queryItem.id
      this._setActivatedIndex(tabIndex, true)
    },

    closeQuery(serverId: string, queryId: string) {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab) {
        return
      }

      const queryIndex = tab.queries.findIndex((q) => q.id === queryId)
      if (queryIndex === -1) {
        return
      }

      tab.queries.splice(queryIndex, 1)

      if (tab.queries.length === 0) {
        tab.queryOpen = false
        tab.activeQueryId = undefined
      } else if (tab.activeQueryId === queryId) {
        const newIndex = Math.min(queryIndex, tab.queries.length - 1)
        tab.activeQueryId = tab.queries[newIndex]?.id
      }
    },

    setActiveQuery(queryId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeQueryId = queryId
    },

    queryTabLabel(tab: ServerTabItem, query: QueryTabItem): string {
      const sameDbQueries = tab.queries.filter((q) => q.database === query.database)
      if (sameDbQueries.length <= 1) {
        return query.database
      }
      const index = sameDbQueries.indexOf(query)
      return index === 0 ? query.database : `${query.database} ${index + 1}`
    },

    openIndexTab(serverId: string, dbName: string, collectionName: string) {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab) {
        return
      }

      if (!tab.indexTabs) {
        tab.indexTabs = []
      }

      const existing = tab.indexTabs.find(
        (t) => t.dbName === dbName && t.collectionName === collectionName,
      )
      if (existing) {
        tab.activeIndexTabId = existing.id
        tab.subTab = BrowserSubTabType.Indexes
        this._setActivatedIndex(tabIndex, true)
        return
      }

      const indexTab: IndexTabItem = {
        id: `index-${++indexTabIdCounter}`,
        serverId,
        dbName,
        collectionName,
      }

      tab.indexTabs.push(indexTab)
      tab.activeIndexTabId = indexTab.id
      tab.subTab = BrowserSubTabType.Indexes
      this._setActivatedIndex(tabIndex, true)
    },

    closeIndexTab(serverId: string, indexTabId: string) {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab || !tab.indexTabs) {
        return
      }

      const idx = tab.indexTabs.findIndex((t) => t.id === indexTabId)
      if (idx === -1) {
        return
      }

      tab.indexTabs.splice(idx, 1)

      if (tab.indexTabs.length === 0) {
        tab.activeIndexTabId = undefined
        tab.subTab = BrowserSubTabType.Query
      } else if (tab.activeIndexTabId === indexTabId) {
        const newIdx = Math.min(idx, tab.indexTabs.length - 1)
        tab.activeIndexTabId = tab.indexTabs[newIdx]?.id
      }
    },

    setActiveIndexTab(indexTabId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeIndexTabId = indexTabId
    },

    indexTabLabel(tab: ServerTabItem, indexTab: IndexTabItem): string {
      return `${indexTab.collectionName} indexes`
    },

    openStatisticsTab(serverId: string, dbName: string, collectionName: string, level: 'collection' | 'database' | 'server') {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab) {
        return
      }

      if (!tab.statisticsTabs) {
        tab.statisticsTabs = []
      }

      const existing = tab.statisticsTabs.find(
        (t) => t.dbName === dbName && t.collectionName === collectionName && t.level === level,
      )
      if (existing) {
        tab.activeStatisticsTabId = existing.id
        tab.subTab = BrowserSubTabType.Statistics
        this._setActivatedIndex(tabIndex, true)
        return
      }

      const statsTab: StatisticsTabItem = {
        id: `stats-${++statisticsTabIdCounter}`,
        serverId,
        dbName,
        collectionName,
        level,
      }

      tab.statisticsTabs.push(statsTab)
      tab.activeStatisticsTabId = statsTab.id
      tab.subTab = BrowserSubTabType.Statistics
      this._setActivatedIndex(tabIndex, true)
    },

    closeStatisticsTab(serverId: string, statisticsTabId: string) {
      const tabIndex = findIndex(this.tabItems, { serverId })
      if (tabIndex === -1) {
        return
      }

      const tab = this.tabItems[tabIndex]
      if (!tab || !tab.statisticsTabs) {
        return
      }

      const idx = tab.statisticsTabs.findIndex((t) => t.id === statisticsTabId)
      if (idx === -1) {
        return
      }

      tab.statisticsTabs.splice(idx, 1)

      if (tab.statisticsTabs.length === 0) {
        tab.activeStatisticsTabId = undefined
        tab.subTab = BrowserSubTabType.Query
      } else if (tab.activeStatisticsTabId === statisticsTabId) {
        const newIdx = Math.min(idx, tab.statisticsTabs.length - 1)
        tab.activeStatisticsTabId = tab.statisticsTabs[newIdx]?.id
      }
    },

    setActiveStatisticsTab(statisticsTabId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeStatisticsTabId = statisticsTabId
    },

    statisticsTabLabel(tab: ServerTabItem, statsTab: StatisticsTabItem): string {
      if (statsTab.level === 'server') {
        return i18nGlobal.t('statistics.serverTabLabel', { server: tab.title })
      }
      if (statsTab.level === 'database') {
        return i18nGlobal.t('statistics.databaseTabLabel', { database: statsTab.dbName })
      }
      return i18nGlobal.t('statistics.tabLabel', { collection: statsTab.collectionName })
    },
  },
})
