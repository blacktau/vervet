import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(async () => undefined),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(),
  CheckMongosh: vi.fn(async () => ({ isSuccess: true, data: true })),
}))

vi.mock('wailsjs/go/api/FilesProxy', () => ({
  SelectFile: vi.fn(),
  ReadFile: vi.fn(),
  WriteFile: vi.fn(),
  SaveFile: vi.fn(),
}))

vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ info: vi.fn(), success: vi.fn(), error: vi.fn(), warning: vi.fn() }),
  useDialoger: () => ({}),
  useMessager: () => ({}),
}))

import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'

const SERVER_ID = 'srv-1'
const QUERY_ID = 'q-1'
const RANGE = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 12 }
const PAGE_CONTEXT = { database: 'dbA', collection: 'c' }

const executeQuery = shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>
const fetchPage = shellProxy.FetchPage as ReturnType<typeof vi.fn>
const countForPage = shellProxy.CountForPage as ReturnType<typeof vi.fn>

function pageOf(...docs: unknown[]) {
  return { isSuccess: true, data: { documents: docs } }
}

async function runPagedQuery() {
  const store = useQueryStore()
  store.initQueryState(QUERY_ID, 'dbA')
  executeQuery.mockResolvedValueOnce({
    isSuccess: true,
    data: { documents: [{ a: 1 }], operationType: 'find', pageContext: PAGE_CONTEXT },
  })
  await store.executeQuery(QUERY_ID, { text: 'db.c.find()', range: RANGE })
  return store
}

describe('queryStore refreshResults', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const tabStore = useTabStore()
    vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
    vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
      serverId: SERVER_ID,
      activeInnerTabId: QUERY_ID,
    } as never)
    executeQuery.mockReset()
    fetchPage.mockReset()
    countForPage.mockReset()
    countForPage.mockResolvedValue({ isSuccess: true, data: { count: 60, estimated: false } })
  })

  test('records the database the results were read from', async () => {
    const store = await runPagedQuery()
    executeQuery.mockResolvedValueOnce({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'aggregate' },
    })
    await store.executeQuery(QUERY_ID, { text: 'use dbC\ndb.c.aggregate([])', range: RANGE })

    expect(store.getQueryState(QUERY_ID).resultsDatabase).toBe('dbC')
    store.setDatabase(QUERY_ID, 'dbB')
    expect(store.getQueryState(QUERY_ID).resultsDatabase).toBe('dbC')
  })

  test('re-fetches the current page and count without replaying the query', async () => {
    const store = await runPagedQuery()
    const state = store.getQueryState(QUERY_ID)
    state.page = 2
    state.pageSize = 20
    fetchPage.mockResolvedValue(pageOf({ a: 2 }))
    countForPage.mockClear()

    await store.refreshResults(QUERY_ID)

    expect(executeQuery).toHaveBeenCalledTimes(1)
    expect(fetchPage).toHaveBeenCalledWith(SERVER_ID, 'dbA', PAGE_CONTEXT, 2, 20)
    expect(countForPage).toHaveBeenCalledTimes(1)
    expect(state.documents).toEqual([{ a: 2 }])
    expect(state.page).toBe(2)
    expect(state.pageSize).toBe(20)
  })

  test('uses the database the results came from, not the current selection', async () => {
    const store = await runPagedQuery()
    store.setDatabase(QUERY_ID, 'dbB')
    fetchPage.mockResolvedValue(pageOf({ a: 2 }))

    await store.refreshResults(QUERY_ID)

    expect(fetchPage.mock.calls[0]![1]).toBe('dbA')
    const countCalls = countForPage.mock.calls
    expect(countCalls[countCalls.length - 1]![1]).toBe('dbA')
  })

  test('reads pages from the database a sibling find ran on', async () => {
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'dbA')
    const siblingContext = { database: 'archive', collection: 'c' }
    executeQuery.mockResolvedValueOnce({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'find', pageContext: siblingContext },
    })
    await store.executeQuery(QUERY_ID, {
      text: "db.getSiblingDB('archive').c.find()",
      range: RANGE,
    })
    fetchPage.mockResolvedValue(pageOf({ a: 2 }))

    await store.refreshResults(QUERY_ID)

    expect(store.getQueryState(QUERY_ID).resultsDatabase).toBe('archive')
    expect(fetchPage.mock.calls[0]![1]).toBe('archive')
  })

  test('clamps to the last page when deletes leave several pages past the end', async () => {
    const store = await runPagedQuery()
    const state = store.getQueryState(QUERY_ID)
    state.page = 4
    state.pageSize = 20
    countForPage.mockResolvedValue({ isSuccess: true, data: { count: 45, estimated: false } })
    fetchPage.mockResolvedValueOnce(pageOf()).mockResolvedValueOnce(pageOf({ a: 9 }))

    await store.refreshResults(QUERY_ID)

    expect(fetchPage).toHaveBeenCalledTimes(2)
    expect(fetchPage.mock.calls[1]![3]).toBe(2)
    expect(state.page).toBe(2)
    expect(state.documents).toEqual([{ a: 9 }])
  })

  test('a pager click made during a refresh wins', async () => {
    const store = await runPagedQuery()
    const state = store.getQueryState(QUERY_ID)
    state.page = 1
    let resolveRefresh!: (v: unknown) => void
    fetchPage
      .mockReturnValueOnce(new Promise((r) => (resolveRefresh = r)))
      .mockResolvedValueOnce(pageOf({ page: 2 }))

    const refresh = store.refreshResults(QUERY_ID)
    await store.fetchPage(QUERY_ID, 2, state.pageSize)
    resolveRefresh(pageOf({ page: 1 }))
    await refresh

    expect(state.page).toBe(2)
    expect(state.documents).toEqual([{ page: 2 }])
  })

  test('a superseded response leaves the newer request loading', async () => {
    const store = await runPagedQuery()
    const state = store.getQueryState(QUERY_ID)
    const resolvers: ((v: unknown) => void)[] = []
    fetchPage.mockImplementation(() => new Promise((r) => resolvers.push(r)))

    const first = store.fetchPage(QUERY_ID, 1, 25)
    const second = store.fetchPage(QUERY_ID, 2, 25)
    resolvers[0]!(pageOf({ page: 1 }))
    await first
    expect(state.loadingPage).toBe(true)

    resolvers[1]!(pageOf({ page: 2 }))
    await second
    expect(state.loadingPage).toBe(false)
  })

  test('leaves unpaged results alone and says to re-run', async () => {
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'dbA')
    executeQuery.mockResolvedValueOnce({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'aggregate' },
    })
    await store.executeQuery(QUERY_ID, { text: 'db.c.aggregate([])', range: RANGE })

    await store.refreshResults(QUERY_ID)

    expect(executeQuery).toHaveBeenCalledTimes(1)
    expect(fetchPage).not.toHaveBeenCalled()
    const messages = store.getQueryState(QUERY_ID).messages
    const last = messages[messages.length - 1]!
    expect(last.text).toMatch(/run the query again/i)
    expect(last.query).toBeUndefined()
  })

  test('does nothing while a query is running', async () => {
    const store = await runPagedQuery()
    store.getQueryState(QUERY_ID).loading = true

    await store.refreshResults(QUERY_ID)

    expect(fetchPage).not.toHaveBeenCalled()
  })

  test('discards a page that arrives after a newer query started', async () => {
    const store = await runPagedQuery()
    let resolvePage!: (v: unknown) => void
    fetchPage.mockReturnValueOnce(new Promise((r) => (resolvePage = r)))
    const refresh = store.refreshResults(QUERY_ID)

    executeQuery.mockResolvedValueOnce({
      isSuccess: true,
      data: { documents: [{ fresh: true }], operationType: 'find' },
    })
    await store.executeQuery(QUERY_ID, { text: 'db.c.find({ x: 1 })', range: RANGE })
    resolvePage(pageOf({ stale: true }))
    await refresh

    expect(store.getQueryState(QUERY_ID).documents).toEqual([{ fresh: true }])
  })
})
