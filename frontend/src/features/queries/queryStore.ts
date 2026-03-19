import { defineStore } from 'pinia'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import * as filesProxy from 'wailsjs/go/api/FilesProxy'
import { useTabStore } from '@/features/tabs/tabs'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { i18nGlobal } from '@/i18n'

export interface QueryState {
  loading: boolean
  documents: unknown[]
  rawJson: string
  rawOutput: string
  error: string
  selectedDatabase: string
  messages: string
  activeResultTab: string
  resultView: 'table' | 'json'
  selectedDocIndex: number
  filePath: string | null
  isDirty: boolean
  savedContent: string | null
  currentContent: string
}

interface QueryStoreState {
  queries: Record<string, QueryState>
  mongoshAvailable: boolean | null
}

function createQueryState(database: string): QueryState {
  return {
    loading: false,
    documents: [],
    rawJson: '',
    rawOutput: '',
    error: '',
    selectedDatabase: database,
    messages: '',
    activeResultTab: 'results',
    resultView: 'table',
    selectedDocIndex: 0,
    filePath: null,
    isDirty: false,
    savedContent: null,
    currentContent: '',
  }
}

export const useQueryStore = defineStore('query', {
  state: (): QueryStoreState => ({
    queries: {},
    mongoshAvailable: null,
  }),
  actions: {
    getQueryState(queryId: string): QueryState {
      if (!this.queries[queryId]) {
        this.queries[queryId] = createQueryState('')
      }
      return this.queries[queryId]
    },

    initQueryState(queryId: string, database: string) {
      if (!this.queries[queryId]) {
        this.queries[queryId] = createQueryState(database)
      }

      this.queries[queryId].selectedDatabase = database
    },

    removeQueryState(queryId: string) {
      delete this.queries[queryId]
    },

    async checkMongosh() {
      const result = await shellProxy.CheckMongosh()
      if (result.isSuccess) {
        this.mongoshAvailable = result.data
      }
    },

    async executeQuery(queryId: string, query: string) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }

      const state = this.getQueryState(queryId)

      if (!state.selectedDatabase) {
        state.error = i18nGlobal.t('query.noDatabaseSelected')
        return
      }

      const settingsStore = useSettingsStore()
      if (settingsStore.editor.queryEngine === 'mongosh' && this.mongoshAvailable === false) {
        state.error = i18nGlobal.t('query.mongoshNotFound')
        return
      }

      state.loading = true
      state.documents = []
      state.rawJson = ''
      state.rawOutput = ''
      state.error = ''
      state.selectedDocIndex = 0

      try {
        const result = await shellProxy.ExecuteQuery(
          serverId,
          state.selectedDatabase,
          query,
        )

        if (result.isSuccess) {
          const data = result.data
          if (data.documents && data.documents.length > 0) {
            state.documents = data.documents
            state.rawJson = JSON.stringify(data.documents, null, 2)
          } else if (data.rawOutput) {
            state.rawOutput = data.rawOutput
          }
          state.activeResultTab = 'results'
        } else {
          const timestamp = new Date().toLocaleTimeString()
          state.error = result.error
          state.messages += `${timestamp} [ERROR] ${result.error}\n`
          state.activeResultTab = 'messages'
        }
      } catch (e) {
        const notifier = useNotifier()
        notifier.error(`Query execution failed: ${e}`)
        state.error = String(e)
        const timestamp = new Date().toLocaleTimeString()
        state.messages += `${timestamp} [ERROR] ${String(e)}\n`
        state.activeResultTab = 'messages'
      } finally {
        state.loading = false
      }
    },

    async cancelQuery(queryId: string) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }

      await shellProxy.CancelQuery(serverId)
      const state = this.getQueryState(queryId)
      state.loading = false
      state.error = i18nGlobal.t('query.queryCancelled')
    },

    setFilePath(queryId: string, filePath: string | null) {
      const state = this.getQueryState(queryId)
      state.filePath = filePath
    },

    setSavedContent(queryId: string, content: string) {
      const state = this.getQueryState(queryId)
      state.savedContent = content
      state.isDirty = false
    },

    setDirty(queryId: string, isDirty: boolean) {
      const state = this.getQueryState(queryId)
      state.isDirty = isDirty
    },

    setCurrentContent(queryId: string, content: string) {
      const state = this.getQueryState(queryId)
      state.currentContent = content
    },

    async openFile(queryId: string): Promise<string | null> {
      const result = await filesProxy.SelectFile(
        i18nGlobal.t('query.openFileDialogTitle'),
        ['*.js', '*.mongodb', '*.*'],
      )
      if (!result.isSuccess || !result.data) {
        return null
      }

      const readResult = await filesProxy.ReadFile(result.data)
      if (!readResult.isSuccess) {
        return null
      }

      this.setFilePath(queryId, result.data)
      this.setSavedContent(queryId, readResult.data)
      this.setCurrentContent(queryId, readResult.data)
      return readResult.data
    },

    async saveFile(queryId: string, content: string): Promise<boolean> {
      const state = this.getQueryState(queryId)
      if (!state.filePath) {
        return this.saveFileAs(queryId, content)
      }

      const result = await filesProxy.WriteFile(state.filePath, content)
      if (!result.isSuccess) {
        return false
      }

      this.setSavedContent(queryId, content)
      return true
    },

    async saveFileAs(queryId: string, content: string): Promise<boolean> {
      const state = this.getQueryState(queryId)
      const defaultName = state.filePath
        ? (state.filePath.split('/').pop() ?? 'query.js')
        : 'query.js'

      const pathResult = await filesProxy.SaveFile(
        i18nGlobal.t('query.saveFileDialogTitle'),
        defaultName,
        ['*.js', '*.mongodb', '*.*'],
      )
      if (!pathResult.isSuccess || !pathResult.data) {
        return false
      }

      const writeResult = await filesProxy.WriteFile(pathResult.data, content)
      if (!writeResult.isSuccess) {
        return false
      }

      this.setFilePath(queryId, pathResult.data)
      this.setSavedContent(queryId, content)
      return true
    },
  },
})
