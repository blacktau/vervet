import { defineStore } from 'pinia'
import { type TabItem } from '@/types/TabItem'
import { useDialoger } from '@/utils/dialog'
import { i18nGlobal } from '@/i18n'
import { find, findIndex, set } from 'lodash'
import { useDataBrowserStore } from '@/components/data-browser/browserStore.ts'

interface TabStoreState {
  nav?: string
  asideWidth: number
  tabItems: TabItem[]
  activeTab: number
}

type TabStoreGetters = {
  tabs: () => TabItem[]
  currentTab: () => TabItem | undefined
  currentTabName: () => string
}

interface TabUpsertOptions {
  subTabIdx?: number,
  subTab?: string,
  server: string,
  database?: string,
  type?: number,
  clearValue?: boolean,
  format?: string,
  collection?: string,
  query?: string,
  forceSwitch?: boolean,
}

export const useTabStore = defineStore('tabs', {
  state: (): TabStoreState => ({
    nav: 'servers',
    asideWidth: 300,
    tabItems: [],
    activeTab: 0,
  }),
  getters: {
    tabs(state: TabStoreState) {
      return state.tabItems
    },
    currentTab(state: TabStoreState) {
      return state.tabItems[state.activeTab]
    },
    currentTabName(state: TabStoreState) {
      return state.tabItems[state.activeTab].name
    },
  } as TabStoreGetters,
  actions: {
    _setActivatedIndex(index: number, switchNav: boolean, subTabIdx: number = 0) {
      this.activeTab = index
      if (switchNav) {
        this.nav = index >= 0 ? 'connections' : 'server'
        set(this.tabItems, [index, 'subTabIdx'], subTabIdx)
      } else {
        if (index < 0) {
          this.nav = 'server'
        }
      }
    },
    openEmptyTab(name: string) {
      this.upsertTab({ server: name, clearValue: true })
    },
    closeTab(name: string) {
      const d = useDialoger()
      d.warning(i18nGlobal.t('dialog.closeConfirm', { name: name }), () => {
        const connectionStore = useDataBrowserStore()
        connectionStore.disconnect(name)
      })
    },
    upsertTab: function (options: TabUpsertOptions) {
      let tabIndex = findIndex(this.tabItems, { name: options.server })
      if (tabIndex === -1) {
        options.subTabIdx = options.subTabIdx || 0
        const tabItem: TabItem = {
          name: options.server,
          title: options.server,
          subTab: options.subTab,
          subTabIdx: options.subTabIdx,
          database: options.database,
        }
        this.tabItems.push(tabItem)
        tabIndex = this.tabItems.length - 1
        this._setActivatedIndex(tabIndex, true, options.subTabIdx)
      } else {
        const tab = this.tabItems[tabIndex]
        tab.blank = false
        tab.subTabIdx = options.subTabIdx || tab.subTabIdx
        tab.title = options.server
        tab.server = options.server
        tab.database = tab.database || options.database

        if (options.forceSwitch === true) {
          this._setActivatedIndex(tabIndex, true, tab.subTabIdx)
        }
      }
    },
    updateLoading(server: string, database: string, loading: boolean) {
      const tab = find(this.tabItems, { name: server, database: database })
      if (!tab) {
        return
      }

      tab.loading = loading
    },
    removeTab(tabIndex: number) {
      const len = this.tabItems.length
      if (len === 1 && this.tabs[0].blank) {
        return undefined
      }

      if (tabIndex < 0 || tabIndex >= len) {
        return undefined
      }

      const removed = this.tabItems.splice(tabIndex, 1)
      this.activeTab -= 1
      if (this.activeTab < 0) {
        if (this.tabs.length > 0) {
          this._setActivatedIndex(0, false)
        } else {
          this._setActivatedIndex(-1, false)
        }
      } else {
        this._setActivatedIndex(this.activeTab, false)
      }

      return removed.length > 0 ? removed[0] : undefined
    },
    removeTabByName(name: string) {
      const tabIndex = findIndex(this.tabItems, { name: name })
      if (tabIndex !== -1) {
        this.removeTab(tabIndex)
      }
    },
    removeAllTabs() {
      this.tabItems = []
      this._setActivatedIndex(-1, false)
    },

    setActiveTab(tab: TabItem) {
      this.activeTab = this.tabItems.indexOf(tab)
    },
  },
})
