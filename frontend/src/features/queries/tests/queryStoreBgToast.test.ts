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

const info = vi.fn()
const success = vi.fn()
const errorFn = vi.fn()
const warning = vi.fn()

vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ info, success, error: errorFn, warning }),
  useDialoger: () => ({}),
  useMessager: () => ({}),
}))

import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'

const SERVER_ID = 'srv-1'
const QUERY_ID = 'q-1'

function setupTabs(activeInnerTabId: string | undefined) {
  const tabStore = useTabStore()
  // Stub the getters by spying.
  vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
  vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
    serverId: SERVER_ID,
    activeInnerTabId,
  } as never)
  return tabStore
}

describe('queryStore background toast', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    info.mockReset()
    success.mockReset()
    errorFn.mockReset()
    warning.mockReset()
    ;(shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>).mockReset()
  })

  test('fires info when query tab not active on success', async () => {
    setupTabs('other-tab')
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')
    ;(shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'find', affectedCount: 1 },
    })

    await store.executeQuery(QUERY_ID, {
      text: 'db.foo.find()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    expect(info).toHaveBeenCalledTimes(1)
    expect(info.mock.calls[0]![0]).toMatch(/mydb/)
  })

  test('does NOT fire when query tab IS active', async () => {
    setupTabs(QUERY_ID)
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')
    ;(shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'find', affectedCount: 1 },
    })

    await store.executeQuery(QUERY_ID, {
      text: 'db.foo.find()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    expect(info).not.toHaveBeenCalled()
    expect(errorFn).not.toHaveBeenCalled()
  })

  test('fires error on error result when backgrounded', async () => {
    setupTabs('other-tab')
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')
    ;(shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: false,
      errorCode: 'something_broke',
      errorDetail: 'detail',
    })

    await store.executeQuery(QUERY_ID, {
      text: 'db.foo.find()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    expect(errorFn).toHaveBeenCalledTimes(1)
    expect(errorFn.mock.calls[0]![0]).toMatch(/mydb/)
  })

  test('does NOT fire when the query was cancelled', async () => {
    setupTabs('other-tab')
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')

    let resolveExec: (value: unknown) => void = () => {}
    ;(shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>).mockReturnValue(
      new Promise((resolve) => {
        resolveExec = resolve
      }),
    )

    const execPromise = store.executeQuery(QUERY_ID, {
      text: 'db.foo.find()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    // Mark cancelled before result arrives.
    const state = store.getQueryState(QUERY_ID)
    state.cancelled = true

    resolveExec({
      isSuccess: true,
      data: { documents: [{ a: 1 }], operationType: 'find', affectedCount: 1 },
    })
    await execPromise

    expect(info).not.toHaveBeenCalled()
    expect(errorFn).not.toHaveBeenCalled()
  })
})
