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
const SCRIPT_PATH = '/home/someone/scripts/import.js'

function executeQueryMock() {
  return shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>
}

// The backend derives __dirname, load() and relative file paths from this
// argument, so a saved tab must send its own path.
describe('queryStore script path', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const tabStore = useTabStore()
    vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
    vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
      serverId: SERVER_ID,
      activeInnerTabId: QUERY_ID,
    } as never)
    executeQueryMock().mockReset()
    executeQueryMock().mockResolvedValue({
      isSuccess: true,
      data: { documents: [], operationType: 'find', affectedCount: 0 },
    })
  })

  async function run() {
    const store = useQueryStore()
    store.initQueryState(QUERY_ID, 'mydb')
    return store
  }

  test('sends the saved tab file path', async () => {
    const store = await run()
    store.setFilePath(QUERY_ID, SCRIPT_PATH)

    await store.executeQuery(QUERY_ID, {
      text: 'load("helper.js")',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    expect(executeQueryMock().mock.calls[0]![4]).toBe(SCRIPT_PATH)
  })

  test('sends an empty path for an unsaved tab', async () => {
    const store = await run()

    await store.executeQuery(QUERY_ID, {
      text: 'db.foo.find()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })

    expect(executeQueryMock().mock.calls[0]![4]).toBe('')
  })
})
