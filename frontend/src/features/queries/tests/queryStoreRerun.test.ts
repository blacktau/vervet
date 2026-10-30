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

const executeQuery = shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>

describe('queryStore rerunLastQuery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const tabStore = useTabStore()
    vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
    vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
      serverId: SERVER_ID,
      activeInnerTabId: QUERY_ID,
    } as never)
    executeQuery.mockReset()
    executeQuery.mockResolvedValue({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'find', affectedCount: 1 },
    })
  })

  test('does nothing when no query has run', async () => {
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')

    await store.rerunLastQuery(QUERY_ID)

    expect(executeQuery).not.toHaveBeenCalled()
  })

  test('re-runs the last executed text and attributes it to the same range', async () => {
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')
    const range = { startLineNumber: 3, startColumn: 1, endLineNumber: 3, endColumn: 14 }
    await store.executeQuery(QUERY_ID, { text: 'db.c.find()', range })

    await store.rerunLastQuery(QUERY_ID)

    expect(executeQuery).toHaveBeenCalledTimes(2)
    expect(executeQuery.mock.calls[1]![3]).toBe('db.c.find()')
    const messages = store.getQueryState(QUERY_ID).messages
    expect(messages.at(-1)!.query).toEqual({ text: 'db.c.find()', range })
  })
})
