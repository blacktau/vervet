import { defineStore } from 'pinia'
import { type IndexTabItem, type QueryTabItem, type ServerTabItem, type StatisticsTabItem } from '@/types/ServerTabItem.ts'
import { useDialoger } from '@/utils/dialog.ts'
import { i18nGlobal } from '@/i18n'
import { findIndex } from 'lodash'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { BrowserSubTabType } from '@/consts/BrowserSubTabType.ts'
import { useQueryStore } from '@/features/queries/queryStore'

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

function innerTabType(id: string): BrowserSubTabType {
  if (id.startsWith('index-')) {
    return BrowserSubTabType.Indexes
  }
  if (id.startsWith('stats-')) {
    return BrowserSubTabType.Statistics
  }
  return BrowserSubTabType.Query
}

function findFallbackInnerTabId(tab: ServerTabItem): string | undefined {
  if (tab.queries.length > 0) {
    return tab.queries[tab.queries.length - 1]!.id
  }
  if (tab.indexTabs && tab.indexTabs.length > 0) {
    return tab.indexTabs[tab.indexTabs.length - 1]!.id
  }
  if (tab.statisticsTabs && tab.statisticsTabs.length > 0) {
    return tab.statisticsTabs[tab.statisticsTabs.length - 1]!.id
  }
  return undefined
}

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
      const tab = state.tabItems[state.activeTabIndex]
      if (!tab?.activeInnerTabId) {
        return undefined
      }
      return innerTabType(tab.activeInnerTabId)
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
      if (!tab?.statisticsTabs || !tab.activeInnerTabId) {
        return undefined
      }
      return tab.statisticsTabs.find((t) => t.id === tab.activeInnerTabId)
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
    async closeTab(serverId: string) {
      const d = useDialoger()
      const tab = this.tabItems.find((x) => x.serverId === serverId)
      if (tab == null) {
        return
      }

      // Check for dirty queries before disconnecting
      const queryStore = useQueryStore()
      for (const query of tab.queries) {
        const state = queryStore.getQueryState(query.id)
        if (state.isDirty) {
          const filename = state.filePath?.split('/').pop() ?? 'Untitled'
          const shouldContinue = await new Promise<boolean>((resolve) => {
            d.show({
              type: 'warning',
              title: i18nGlobal.t('query.unsavedChangesTitle'),
              content: i18nGlobal.t('query.unsavedChangesMessage', { filename }),
              positiveText: i18nGlobal.t('query.unsavedChangesSave'),
              negativeText: i18nGlobal.t('query.unsavedChangesDontSave'),
              onPositiveClick: async () => {
                const saved = await queryStore.saveFile(query.id, state.currentContent)
                resolve(saved)
              },
              onNegativeClick: () => {
                resolve(true)
              },
              onClose: () => {
                resolve(false)
              },
            })
          })
          if (!shouldContinue) {
            return
          }
        }
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

    setActiveTab(tab: ServerTabItem) {
      this.activeTabIndex = this.tabItems.indexOf(tab)
    },

    setActiveTabIndex(index: number) {
      this.activeTabIndex = index
    },

    setActiveInnerTab(innerTabId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeInnerTabId = innerTabId
    },

    openQuery(serverId: string, database: string, initialText?: string, collectionName?: string) {
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
        collectionName,
      }

      tab.queries.push(queryItem)
      tab.activeInnerTabId = queryItem.id
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

      if (tab.activeInnerTabId === queryId) {
        if (tab.queries.length > 0) {
          const newIndex = Math.min(queryIndex, tab.queries.length - 1)
          tab.activeInnerTabId = tab.queries[newIndex]?.id
        } else {
          tab.activeInnerTabId = findFallbackInnerTabId(tab)
        }
      }
    },

    queryTabLabel(tab: ServerTabItem, query: QueryTabItem): string {
      if (query.filePath) {
        return query.filePath.split('/').pop() ?? query.database
      }
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
        tab.activeInnerTabId = existing.id
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
      tab.activeInnerTabId = indexTab.id
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

      if (tab.activeInnerTabId === indexTabId) {
        if (tab.indexTabs.length > 0) {
          const newIdx = Math.min(idx, tab.indexTabs.length - 1)
          tab.activeInnerTabId = tab.indexTabs[newIdx]?.id
        } else {
          tab.activeInnerTabId = findFallbackInnerTabId(tab)
        }
      }
    },

    setActiveIndexTab(indexTabId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeInnerTabId = indexTabId
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
        tab.activeInnerTabId = existing.id
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
      tab.activeInnerTabId = statsTab.id
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

      if (tab.activeInnerTabId === statisticsTabId) {
        if (tab.statisticsTabs.length > 0) {
          const newIdx = Math.min(idx, tab.statisticsTabs.length - 1)
          tab.activeInnerTabId = tab.statisticsTabs[newIdx]?.id
        } else {
          tab.activeInnerTabId = findFallbackInnerTabId(tab)
        }
      }
    },

    setActiveStatisticsTab(statisticsTabId: string) {
      const tab = this.tabItems[this.activeTabIndex]
      if (!tab) {
        return
      }
      tab.activeInnerTabId = statisticsTabId
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
