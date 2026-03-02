import { defineStore } from 'pinia'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { useTabStore } from '@/features/tabs/tabs'
import { useNotifier } from '@/utils/dialog'

interface QueryState {
  loading: boolean
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
  rawJson: string
  rawOutput: string
  error: string
  selectedDatabase: string
  messages: string
  activeResultTab: string
  resultView: 'table' | 'json'
  selectedDocIndex: number
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
      const result = await connectionsProxy.CheckMongosh()
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
        state.error = 'No database selected'
        return
      }

      if (this.mongoshAvailable === false) {
        state.error = 'mongosh is not installed or not in PATH'
        return
      }

      state.loading = true
      state.documents = []
      state.rawJson = ''
      state.rawOutput = ''
      state.error = ''
      state.selectedDocIndex = 0

      try {
        const result = await connectionsProxy.ExecuteQuery(
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
        } else {
          const timestamp = new Date().toLocaleTimeString()
          state.error = result.error
          state.messages += `--- ${timestamp} [ERROR] ---\n${result.error}\n\n`
          state.activeResultTab = 'messages'
        }
      } catch (e) {
        const notifier = useNotifier()
        notifier.error(`Query execution failed: ${e}`)
        state.error = String(e)
        const timestamp = new Date().toLocaleTimeString()
        state.messages += `--- ${timestamp} [ERROR] ---\n${String(e)}\n\n`
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

      await connectionsProxy.CancelQuery(serverId)
      const state = this.getQueryState(queryId)
      state.loading = false
      state.error = 'Query cancelled'
    },
  },
})
