import { defineStore } from 'pinia'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'
import { useTabStore } from '@/features/tabs/tabs'
import { useNotifier } from '@/utils/dialog'

interface QueryStoreState {
  loading: boolean
  result: string
  error: string
  mongoshAvailable: boolean | null
  selectedDatabase: string
}

export const useQueryStore = defineStore('query', {
  state: (): QueryStoreState => ({
    loading: false,
    result: '',
    error: '',
    mongoshAvailable: null,
    selectedDatabase: '',
  }),
  actions: {
    async checkMongosh() {
      const result = await connectionsProxy.CheckMongosh()
      if (result.isSuccess) {
        this.mongoshAvailable = result.data
      }
    },

    async executeQuery(query: string) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        this.error = 'No server selected'
        return
      }

      if (!this.selectedDatabase) {
        this.error = 'No database selected'
        return
      }

      if (this.mongoshAvailable === false) {
        this.error = 'mongosh is not installed or not in PATH'
        return
      }

      this.loading = true
      this.result = ''
      this.error = ''

      try {
        const result = await connectionsProxy.ExecuteQuery(
          serverId,
          this.selectedDatabase,
          query,
        )

        if (result.isSuccess) {
          this.result = result.data
        } else {
          this.error = result.error
        }
      } catch (e) {
        const notifier = useNotifier()
        notifier.error(`Query execution failed: ${e}`)
        this.error = String(e)
      } finally {
        this.loading = false
      }
    },

    async cancelQuery() {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) return

      await connectionsProxy.CancelQuery(serverId)
      this.loading = false
      this.error = 'Query cancelled'
    },

    setDatabase(dbName: string) {
      this.selectedDatabase = dbName
    },
  },
})
