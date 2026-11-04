import { defineStore } from 'pinia'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import * as filesProxy from 'wailsjs/go/api/FilesProxy'
import type { models } from 'wailsjs/go/models'
import { useTabStore } from '@/features/tabs/tabs'
import { useNotifier } from '@/utils/dialog'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { i18nGlobal } from '@/i18n'
import { errorCodeText, notifyResultError, resultErrorText } from '@/utils/resultError'

export type PageContext = models.PageContext
export type ResultNamespace = models.ResultNamespace

function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`
  }
  return `${(ms / 1000).toFixed(1)}s`
}

function resultMessage(operationType: string, count: number, duration: string): string {
  const key = `query.messages.${operationType}Result`
  const translated = i18nGlobal.t(key, { count, time: duration })
  if (translated === key) {
    return i18nGlobal.t('query.messages.genericResult', { time: duration })
  }
  return translated
}

/** Operations whose summary counts the documents they returned. */
const documentOperations = new Set(['find', 'findOne', 'aggregate', 'listIndexes'])

/**
 * The count for an operation's summary. Every other operation counts what it
 * affected, not its result document. A zero is left out of the JSON, so a write
 * that changed nothing arrives without a count.
 */
function resultCount(operationType: string, affectedCount: number | undefined, docCount: number) {
  return documentOperations.has(operationType) ? docCount : (affectedCount ?? 0)
}

export type LogLevel = 'info' | 'warning' | 'error'

const levelRank: Record<LogLevel, number> = { info: 0, warning: 1, error: 2 }

/** The backend sends levels as plain strings; anything unknown is info. */
function toLogLevel(level: string): LogLevel {
  return level === 'warning' || level === 'error' ? level : 'info'
}

/** Printed lines added to Messages while the Results tab was showing, by level. */
export type UnseenOutput = Record<LogLevel, number>

/** What the Messages tab badge shows: a count and the level that colours it. */
export interface UnseenBadge {
  count: number
  level: LogLevel
}

/**
 * Printed lines past this many are combined into one Messages entry, so a
 * script that prints in a long loop does not add tens of thousands of rows.
 */
export const MAX_OUTPUT_ENTRIES = 1000

export interface LogMessageQuery {
  text: string
  range: {
    startLineNumber: number
    startColumn: number
    endLineNumber: number
    endColumn: number
  }
}

export interface LogMessage {
  id: string
  timestamp: string
  level: LogLevel
  text: string
  query?: LogMessageQuery
}

export interface MessageFilter {
  info: boolean
  warning: boolean
  error: boolean
}

function makeId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export interface QueryState {
  loading: boolean
  cancelled: boolean
  executionId: number
  documents: unknown[]
  rawOutput: string
  error: string
  selectedDatabase: string
  messages: LogMessage[]
  messageFilter: MessageFilter
  /** Printed lines the user has not seen yet, shown as a badge on the Messages tab */
  unseenOutput: UnseenOutput | null
  /** The "Query cancelled" message, so output arriving after it can go above it. */
  cancelNoticeId: string | null
  activeResultTab: string
  resultView: 'table' | 'json'
  selectedDocIndex: number
  filePath: string | null
  isDirty: boolean
  savedContent: string | null
  currentContent: string
  /** Timestamp (ms since epoch) when the current execution started; null when idle */
  runStartedAt: number | null
  /** The limit() from the last executed query, when the result may be truncated */
  activeLimit: number | null
  /** Lazily computed JSON — only populated when the JSON view is first accessed */
  _rawJsonCache: string | null
  /** Server-side paging state. Populated when the engine returns a PageContext. */
  pageContext: PageContext | null
  /** Where the current results are stored, when the engine knows. Edits and deletes go here. */
  resultsNamespace: ResultNamespace | null
  /**
   * A document was edited and the results can't be re-fetched, so its row is out of
   * date. Saving from that row would replace the document with the old values.
   */
  resultsStale: boolean
  page: number
  pageSize: number
  total: number | null
  totalEstimated: boolean
  loadingPage: boolean
  loadingCount: boolean
  /** Latest page and count requests; older responses are discarded */
  pageRequestId: number
  countRequestId: number
  /** The page the latest page request asked for, which may not have arrived yet */
  requestedPage: number
  requestedPageSize: number
}

interface QueryStoreState {
  queries: Record<string, QueryState>
  mongoshAvailable: boolean | null
}

let nextExecutionId = 0
let nextRequestId = 0

function createQueryState(database: string): QueryState {
  return {
    loading: false,
    cancelled: false,
    executionId: 0,
    documents: [],
    rawOutput: '',
    error: '',
    selectedDatabase: database,
    messages: [],
    messageFilter: { info: true, warning: true, error: true },
    unseenOutput: null,
    cancelNoticeId: null,
    activeResultTab: 'results',
    resultView: 'table',
    selectedDocIndex: 0,
    filePath: null,
    isDirty: false,
    savedContent: null,
    currentContent: '',
    runStartedAt: null,
    activeLimit: null,
    _rawJsonCache: null,
    pageContext: null,
    resultsNamespace: null,
    resultsStale: false,
    page: 0,
    pageSize: 25,
    total: null,
    totalEstimated: false,
    loadingPage: false,
    loadingCount: false,
    pageRequestId: 0,
    countRequestId: 0,
    requestedPage: 0,
    requestedPageSize: 25,
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

    setDatabase(queryId: string, database: string) {
      const state = this.getQueryState(queryId)
      state.selectedDatabase = database
    },

    appendMessage(
      queryId: string,
      msg: { level: LogLevel; text: string; query?: LogMessageQuery },
    ) {
      const state = this.getQueryState(queryId)
      const id = makeId()
      state.messages.push({
        id,
        timestamp: new Date().toLocaleTimeString(),
        level: msg.level,
        text: msg.text,
        query: msg.query,
      })
      return id
    },

    clearMessages(queryId: string) {
      const state = this.getQueryState(queryId)
      state.messages = []
      state.unseenOutput = null
    },

    /**
     * Adds a script's output to Messages in one update and returns how many
     * printed lines of each level there were, or null when there were none.
     * A notice (text from the engine, not the script) is added but not counted.
     * Past MAX_OUTPUT_ENTRIES the rest are combined into one entry at their most
     * severe level, after a notice saying so; each combined line still counts.
     * With before, they go above that message, or at the end if it has been
     * cleared.
     */
    appendOutput(
      queryId: string,
      output: models.OutputLine[] | undefined,
      query: LogMessageQuery,
      before: string | null = null,
    ): UnseenOutput | null {
      if (!output || output.length === 0) {
        return null
      }
      const state = this.getQueryState(queryId)
      const timestamp = new Date().toLocaleTimeString()
      const counts: UnseenOutput = { info: 0, warning: 0, error: 0 }
      const count = (line: models.OutputLine, level: LogLevel) => {
        if (!line.notice) {
          counts[level]++
        }
      }
      const entry = (level: LogLevel, text: string): LogMessage => ({
        id: makeId(),
        timestamp,
        level,
        text,
        query,
      })

      const added: LogMessage[] = output.slice(0, MAX_OUTPUT_ENTRIES).map((line) => {
        const level = toLogLevel(line.level)
        count(line, level)
        return entry(level, line.text)
      })

      const rest = output.slice(MAX_OUTPUT_ENTRIES)
      if (rest.length > 0) {
        let level: LogLevel = 'info'
        for (const line of rest) {
          const l = toLogLevel(line.level)
          count(line, l)
          if (levelRank[l] > levelRank[level]) {
            level = l
          }
        }
        added.push(
          entry(
            'warning',
            i18nGlobal.t('query.messages.outputCombined', {
              shown: MAX_OUTPUT_ENTRIES,
              count: rest.length,
            }),
          ),
        )
        added.push(entry(level, rest.map((line) => line.text).join('\n')))
      }

      // One assignment, not a push per line: each push is a reactive update.
      const at = before === null ? -1 : state.messages.findIndex((m) => m.id === before)
      state.messages =
        at < 0
          ? state.messages.concat(added)
          : [...state.messages.slice(0, at), ...added, ...state.messages.slice(at)]
      return counts.info + counts.warning + counts.error > 0 ? counts : null
    },

    /** Called when the Messages tab is opened: its badge has been seen. */
    markOutputSeen(queryId: string) {
      this.getQueryState(queryId).unseenOutput = null
    },

    /**
     * The Messages tab badge: unseen printed lines at the levels the Messages
     * filter shows, coloured by the most severe of them. Null when there are none.
     */
    unseenBadge(queryId: string): UnseenBadge | null {
      const state = this.getQueryState(queryId)
      const unseen = state.unseenOutput
      if (!unseen) {
        return null
      }
      let count = 0
      let level: LogLevel = 'info'
      for (const l of ['info', 'warning', 'error'] as const) {
        if (!state.messageFilter[l] || unseen[l] === 0) {
          continue
        }
        count += unseen[l]
        level = l
      }
      return count > 0 ? { count, level } : null
    },

    setMessageFilter(queryId: string, level: LogLevel, enabled: boolean) {
      const state = this.getQueryState(queryId)
      state.messageFilter[level] = enabled
    },

    async checkMongosh() {
      const result = await shellProxy.CheckMongosh()
      if (result.isSuccess) {
        this.mongoshAvailable = result.data
      }
    },

    async executeQuery(
      queryId: string,
      payload: { text: string; range: LogMessageQuery['range'] },
    ) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }

      const isBackgrounded = (): boolean => {
        const tab = tabStore.currentTab
        if (!tab) {
          return true
        }
        if (tabStore.currentTabId !== serverId) {
          return true
        }
        return tab.activeInnerTabId !== queryId
      }

      const state = this.getQueryState(queryId)
      let query = payload.text
      const queryPayload: LogMessageQuery = { text: payload.text, range: payload.range }

      // Handle "use <database>" command — extract it and switch database
      const useMatch = query.match(/(?:^|\n)\s*use\s+(\S+)\s*(?:\n|$)/)
      if (useMatch) {
        const dbName = useMatch[1]!
        state.selectedDatabase = dbName
        this.appendMessage(queryId, {
          level: 'info',
          text: i18nGlobal.t('query.messages.switchedDatabase', { name: dbName }),
          query: queryPayload,
        })

        // Remove the use line and run any remaining query
        query = query.replace(/(?:^|\n)\s*use\s+\S+\s*(?:\n|$)/, '\n').trim()
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
        this.appendMessage(queryId, { level: 'error', text: msg, query: queryPayload })
        return
      }

      const settingsStore = useSettingsStore()
      if (settingsStore.query.queryEngine === 'mongosh' && this.mongoshAvailable === false) {
        const msg = i18nGlobal.t('errors.shell_not_found')
        state.error = msg
        this.appendMessage(queryId, { level: 'error', text: msg, query: queryPayload })
        return
      }

      const thisExecution = ++nextExecutionId
      state.loading = true
      state.cancelled = false
      state.executionId = thisExecution
      state.documents = []
      state.resultsNamespace = null
      state.resultsStale = false
      state._rawJsonCache = null
      state.rawOutput = ''
      state.error = ''
      state.unseenOutput = null
      state.cancelNoticeId = null
      state.selectedDocIndex = 0
      state.activeLimit = parseLimit(query)
      state.pageContext = null
      state.page = 0
      state.pageSize = settingsStore.query.defaultPageSize
      state.total = null
      state.totalEstimated = false
      state.loadingPage = false
      state.loadingCount = false
      state.pageRequestId = 0
      state.countRequestId = 0
      state.requestedPage = 0
      state.requestedPageSize = state.pageSize
      this.appendMessage(queryId, {
        level: 'info',
        text: i18nGlobal.t('query.messages.executing'),
        query: queryPayload,
      })

      try {
        state.runStartedAt = Date.now()
        const result = await shellProxy.ExecuteQuery(
          serverId,
          queryId,
          state.selectedDatabase,
          query,
          state.filePath ?? '',
        )

        if (state.executionId !== thisExecution) {
          return
        }
        if (state.cancelled) {
          // What the script printed came before the cancel, so it goes above the notice.
          const added = this.appendOutput(
            queryId,
            result.data?.output,
            queryPayload,
            state.cancelNoticeId,
          )
          if (state.activeResultTab !== 'messages') {
            state.unseenOutput = added
          }
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

          // What the script printed goes to Messages. A script with no result
          // opens Messages; otherwise Results stays open and the Messages tab
          // shows a badge for the lines waiting there. An empty cursor
          // (pageContext) is still a result.
          const added = this.appendOutput(queryId, data.output, queryPayload)
          // hasValue covers an empty result ([] or an empty cursor), which
          // both engines report the same way; the rest covers older shapes.
          const hasResult =
            !!data.hasValue ||
            (data.documents?.length ?? 0) > 0 ||
            !!data.rawOutput ||
            !!data.pageContext
          if (!hasResult && added !== null) {
            state.activeResultTab = 'messages'
          } else {
            state.activeResultTab = 'results'
            state.unseenOutput = added
          }

          state.resultsNamespace = data.namespace ?? null
          if (data.pageContext) {
            state.pageContext = data.pageContext
            state.page = 0
            void this.fetchCount(queryId)
          }

          const elapsed = formatDuration(Date.now() - (state.runStartedAt ?? Date.now()))
          const docCount = data.documents?.length ?? 0
          const operationType = data.operationType || 'find'
          const msg = hasResult
            ? resultMessage(
                operationType,
                resultCount(operationType, data.affectedCount, docCount),
                elapsed,
              )
            : i18nGlobal.t('query.messages.genericResult', { time: elapsed })
          this.appendMessage(queryId, { level: 'info', text: msg, query: queryPayload })

          if (isBackgrounded() && state.executionId === thisExecution && !state.cancelled) {
            useNotifier().info(
              i18nGlobal.t('query.messages.bgFinished', {
                db: state.selectedDatabase,
                elapsed,
              }),
            )
          }
        } else {
          // What the script printed before it failed comes first.
          this.appendOutput(queryId, result.data?.output, queryPayload)
          const translated = resultErrorText(result)
          state.error = translated
          this.appendMessage(queryId, {
            level: 'error',
            text: translated,
            query: queryPayload,
          })
          if (result.errorDetail && result.errorDetail !== translated) {
            this.appendMessage(queryId, {
              level: 'error',
              text: result.errorDetail,
              query: queryPayload,
            })
          }
          state.activeResultTab = 'messages'

          if (isBackgrounded() && state.executionId === thisExecution && !state.cancelled) {
            useNotifier().error(
              i18nGlobal.t('query.messages.bgFailed', { db: state.selectedDatabase }),
            )
          }
        }
      } catch (e) {
        if (state.executionId !== thisExecution) {
          return
        }
        if (state.cancelled) {
          return
        }
        const notifier = useNotifier()
        const backgrounded = isBackgrounded()
        if (backgrounded) {
          notifier.error(i18nGlobal.t('query.messages.bgFailed', { db: state.selectedDatabase }))
        } else {
          notifier.error(String(e))
        }
        state.error = String(e)
        this.appendMessage(queryId, { level: 'error', text: String(e), query: queryPayload })
        state.activeResultTab = 'messages'
      } finally {
        state.loading = false
        state.runStartedAt = null
      }
    },

    async fetchPage(queryId: string, page: number, pageSize: number) {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return
      }
      const state = this.getQueryState(queryId)
      if (!state.pageContext) {
        return
      }
      const requestId = ++nextRequestId
      state.pageRequestId = requestId
      state.requestedPage = page
      state.requestedPageSize = pageSize
      state.loadingPage = true
      try {
        const result = await shellProxy.FetchPage(serverId, state.pageContext, page, pageSize)
        // A newer page request or query has taken over
        if (state.pageRequestId !== requestId) {
          return
        }
        if (result.isSuccess && result.data) {
          state.documents = result.data.documents ?? []
          state.page = page
          state.pageSize = pageSize
          state._rawJsonCache = null
        } else {
          state.requestedPage = state.page
          state.requestedPageSize = state.pageSize
          notifyResultError(result)
        }
      } catch (e) {
        if (state.pageRequestId === requestId) {
          state.requestedPage = state.page
          state.requestedPageSize = state.pageSize
          useNotifier().error(String(e))
        }
      } finally {
        if (state.pageRequestId === requestId) {
          state.loadingPage = false
        }
      }
    },

    /** Returns true when a fresh count was stored. */
    async fetchCount(queryId: string): Promise<boolean> {
      const tabStore = useTabStore()
      const serverId = tabStore.currentTabId
      if (!serverId) {
        return false
      }
      const state = this.getQueryState(queryId)
      if (!state.pageContext) {
        return false
      }
      const requestId = ++nextRequestId
      state.countRequestId = requestId
      state.loadingCount = true
      try {
        const result = await shellProxy.CountForPage(serverId, state.pageContext)
        if (state.countRequestId !== requestId) {
          return false
        }
        if (!result.isSuccess || !result.data) {
          return false
        }
        state.total = result.data.count
        state.totalEstimated = result.data.estimated
        return true
      } catch {
        // The count is advisory: the pager keeps its previous total
        return false
      } finally {
        if (state.countRequestId === requestId) {
          state.loadingCount = false
        }
      }
    },

    /**
     * Refresh the results after a document was edited, inserted or deleted. Only paged
     * results can be re-fetched safely; re-running the query text could repeat writes.
     */
    async refreshResults(queryId: string, edited = false) {
      const state = this.getQueryState(queryId)
      if (state.loading) {
        return
      }
      if (!state.pageContext) {
        state.resultsStale = state.resultsStale || edited
        this.appendMessage(queryId, {
          level: 'info',
          text: i18nGlobal.t('query.messages.resultsStale'),
        })
        return
      }
      const pageRequestBefore = state.pageRequestId
      const counted = await this.fetchCount(queryId)
      // Use the page last asked for: a click whose response hasn't arrived still counts
      const pageSize = state.requestedPageSize
      let page = state.requestedPage
      // Deletes can leave the page, or several, past the end. Clamp before fetching
      // so an empty page never shows; without a fresh count, keep the page.
      if (counted && state.total !== null) {
        page = Math.min(page, Math.max(Math.ceil(state.total / pageSize) - 1, 0))
      }
      // A pager click during the count was sent after the change, so it's already
      // fresh, unless it asked for a page the change has emptied
      const clickedDuringCount = state.pageRequestId !== pageRequestBefore
      if (clickedDuringCount && page === state.requestedPage) {
        return
      }
      await this.fetchPage(queryId, page, pageSize)
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
      state.cancelNoticeId = this.appendMessage(queryId, {
        level: 'warning',
        text: i18nGlobal.t('errors.query_cancelled'),
      })
      await shellProxy.CancelQuery(serverId, queryId)
    },

    setFilePath(queryId: string, filePath: string | null) {
      const state = this.getQueryState(queryId)
      state.filePath = filePath
    },

    renameFilePath(oldPath: string, newPath: string) {
      const prefix = oldPath + '/'
      for (const state of Object.values(this.queries)) {
        if (state.filePath === oldPath) {
          state.filePath = newPath
        } else if (state.filePath && state.filePath.startsWith(prefix)) {
          state.filePath = newPath + state.filePath.slice(oldPath.length)
        }
      }
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
        notifier.error(errorCodeText(result.errorCode), {
          title: i18nGlobal.t('errorTitles.loadFile'),
          detail: result.errorDetail,
        })
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
