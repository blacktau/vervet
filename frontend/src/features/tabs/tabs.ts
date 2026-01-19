import { defineStore } from 'pinia'
import { type ServerTabItem } from '@/types/ServerTabItem.ts'
import { useDialoger } from '@/utils/dialog.ts'
import { i18nGlobal } from '@/i18n'
import { findIndex, set } from 'lodash'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

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
  } as TabStoreGetters,
  actions: {
    _setActivatedIndex(index: number, switchNav: boolean, subTabIdx: number = 0) {
      console.log('_setActivatedIndex', index, switchNav, subTabIdx)
      this.activeTabIndex = index
      if (switchNav) {
        this.nav = index >= 0 ? NavType.Browser : NavType.Servers
        set(this.tabItems, [index, 'subTabIdx'], subTabIdx)
      } else {
        if (index < 0) {
          this.nav = NavType.Servers
        }
      }
    },
    closeTab(serverId: string) {
      console.log('closeTab', serverId)
      const d = useDialoger()
      const tab = this.tabItems.find((x) => x.serverId === serverId)
      console.log('closeTab', serverId, tab)
      if (tab == null) {
        return
      }
      d.warning(i18nGlobal.t('dialog.closeConfirm', { name: tab.title }), async () => {
        console.log('closeTab confirmed', serverId)
        const connectionStore = useDataBrowserStore()
        await connectionStore.disconnect(tab.serverId)
      })
    },
    upsertTab: function (options: TabUpsertOptions) {
      console.log('upsertTab', options)
      let tabIndex = findIndex(this.tabItems, { serverId: options.serverId })
      if (tabIndex === -1) {
        const tabItem: ServerTabItem = {
          serverId: options.serverId,
          title: options.title,
          blank: false,
        }
        this.tabItems.push(tabItem)
        tabIndex = this.tabItems.length - 1
        this._setActivatedIndex(tabIndex, true, -1)
      } else {
        const tab = this.tabItems[tabIndex]
        if (tab == null) {
          return
        }

        tab.blank = false
        tab.title = options.title
        tab.serverId = options.serverId

        if (options.forceSwitch === true) {
          this._setActivatedIndex(tabIndex, true, -1)
        }
      }
    },
    removeTabById(serverId: string) {
      const tabIndex = findIndex(this.tabItems, { serverId: serverId })

      console.log('removeTabById', serverId, tabIndex)

      this.removeTab(tabIndex)
    },
    removeTab(tabIndex: number) {
      console.log('removeTab', tabIndex)
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
        if (this.tabs.length > 0) {
          this._setActivatedIndex(0, false)
        } else {
          this._setActivatedIndex(-1, false)
        }
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
  },
})
