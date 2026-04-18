import { defineStore } from 'pinia'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import * as filesProxy from 'wailsjs/go/api/FilesProxy'
import { useTabStore } from '@/features/tabs/tabs'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { i18nGlobal } from '@/i18n'

function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(1)}s`
}

function translateError(errorCode: string, errorDetail: string): string {
  const key = `errors.${errorCode}`
  const translated = i18nGlobal.t(key)
  if (translated === key) {
    return errorDetail || errorCode
  }
  return translated
}

function resultMessage(operationType: string, count: number, duration: string): string {
  const key = `query.messages.${operationType}Result`
  const translated = i18nGlobal.t(key, { count, time: duration })
  if (translated === key) {
    return i18nGlobal.t('query.messages.genericResult', { time: duration })
  }
  return translated
}

export interface QueryState {
  loading: boolean
  cancelled: boolean
  executionId: number
  documents: unknown[]
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
  /** The limit() from the last executed query, when the result may be truncated */
  activeLimit: number | null
  /** Lazily computed JSON — only populated when the JSON view is first accessed */
  _rawJsonCache: string | null
}

interface QueryStoreState {
  queries: Record<string, QueryState>
  mongoshAvailable: boolean | null
}

let nextExecutionId = 0

function createQueryState(database: string): QueryState {
  return {
    loading: false,
    cancelled: false,
    executionId: 0,
    documents: [],
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
    activeLimit: null,
    _rawJsonCache: null,
  }
}

function parseLimit(query: string): number | null {
  const match = query.match(/\.limit\s*\(\s*(\d+)\s*\)/)
  if (!match) {
    return null
  }
  const n = Number(match[1])
  return Number.isFinite(n) && n > 0 ? n : null
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

    getRawJson(queryId: string): string {
      const state = this.getQueryState(queryId)
      if (state._rawJsonCache === null && state.documents.length > 0) {
        state._rawJsonCache = JSON.stringify(state.documents, null, 2)
      }
      return state._rawJsonCache ?? ''
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

    async executeQuery(queryId: string, rawQuery: string) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }

      const state = this.getQueryState(queryId)
      const timestamp = new Date().toLocaleTimeString()

      // Handle "use <database>" command — extract it and switch database
      let query = rawQuery
      const useMatch = query.match(/(?:^|\n)\s*use\s+(\S+)\s*(?:\n|$)/)
      if (useMatch) {
        const dbName = useMatch[1]!
        state.selectedDatabase = dbName
        state.messages += `${timestamp} [INFO] ${i18nGlobal.t('query.messages.switchedDatabase', { name: dbName })}\n`

        // Remove the use line and run any remaining query
        const remaining = query.replace(/(?:^|\n)\s*use\s+\S+\s*(?:\n|$)/, '\n').trim()
        if (!remaining || /^\s*(\/\/.*\s*)*$/.test(remaining)) {
          return
        }
        query = remaining
      }

      // Skip if query is empty or only comments
      const stripped = query
        .split('\n')
        .filter((line) => !/^\s*(\/\/|$)/.test(line))
        .join('')
        .trim()
      if (!stripped) {
        return
      }

      if (!state.selectedDatabase) {
        const msg = i18nGlobal.t('errors.no_database_selected')
        state.error = msg
        state.messages += `${timestamp} [ERROR] ${msg}\n`
        return
      }

      const settingsStore = useSettingsStore()
      if (settingsStore.editor.queryEngine === 'mongosh' && this.mongoshAvailable === false) {
        const msg = i18nGlobal.t('errors.shell_not_found')
        state.error = msg
        state.messages += `${timestamp} [ERROR] ${msg}\n`
        return
      }

      const thisExecution = ++nextExecutionId
      state.loading = true
      state.cancelled = false
      state.executionId = thisExecution
      state.documents = []
      state._rawJsonCache = null
      state.rawOutput = ''
      state.error = ''
      state.selectedDocIndex = 0
      state.activeLimit = parseLimit(query)
      state.messages += `${timestamp} [INFO] ${i18nGlobal.t('query.messages.executing')}\n`

      const startTime = Date.now()

      try {
        const result = await shellProxy.ExecuteQuery(serverId, state.selectedDatabase, query)

        if (state.cancelled || state.executionId !== thisExecution) {
          return
        }

        if (result.isSuccess) {
          const data = result.data
          if (data.documents && data.documents.length > 0) {
            state.documents = data.documents
            state._rawJsonCache = null
          } else if (data.rawOutput) {
            state.rawOutput = data.rawOutput
          }
          state.activeResultTab = 'results'

          const elapsed = formatDuration(Date.now() - startTime)
          const docCount = data.documents?.length ?? 0
          const opType = data.operationType || 'find'
          const msg = resultMessage(opType, data.affectedCount || docCount, elapsed)
          const ts = new Date().toLocaleTimeString()
          state.messages += `${ts} [INFO] ${msg}\n`
        } else {
          const translated = translateError(result.errorCode, result.errorDetail)
          state.error = translated
          const ts = new Date().toLocaleTimeString()
          state.messages += `${ts} [ERROR] ${translated}\n`
          if (result.errorDetail && result.errorDetail !== translated) {
            state.messages += `${ts} [ERROR] ${result.errorDetail}\n`
          }
          state.activeResultTab = 'messages'
        }
      } catch (e) {
        if (state.cancelled || state.executionId !== thisExecution) {
          return
        }
        const notifier = useNotifier()
        notifier.error(String(e))
        state.error = String(e)
        const ts = new Date().toLocaleTimeString()
        state.messages += `${ts} [ERROR] ${String(e)}\n`
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

      const state = this.getQueryState(queryId)
      state.cancelled = true
      state.loading = false
      state.error = ''
      const ts = new Date().toLocaleTimeString()
      state.messages += `${ts} [WARNING] ${i18nGlobal.t('errors.query_cancelled')}\n`
      await shellProxy.CancelQuery(serverId)
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
      const result = await filesProxy.SelectFile(i18nGlobal.t('query.openFileDialogTitle'), [
        { displayName: i18nGlobal.t('query.filterJavascript'), pattern: '*.js' },
        { displayName: i18nGlobal.t('query.filterMongodb'), pattern: '*.mongodb' },
        { displayName: i18nGlobal.t('query.filterAllFiles'), pattern: '*.*' },
      ])
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

    async loadFileByPath(queryId: string, filePath: string): Promise<boolean> {
      const result = await filesProxy.ReadFile(filePath)
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(result.errorDetail || result.errorCode)
        return false
      }

      const state = this.getQueryState(queryId)
      state.filePath = filePath
      state.savedContent = result.data
      state.currentContent = result.data
      state.isDirty = false
      return true
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
        [
          { displayName: i18nGlobal.t('query.filterJavascript'), pattern: '*.js' },
          { displayName: i18nGlobal.t('query.filterMongodb'), pattern: '*.mongodb' },
          { displayName: i18nGlobal.t('query.filterAllFiles'), pattern: '*.*' },
        ],
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
