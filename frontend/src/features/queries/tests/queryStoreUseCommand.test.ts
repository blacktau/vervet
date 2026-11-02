import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(async () => undefined),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(async () => ({ isSuccess: true, data: { count: 0, estimated: false } })),
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
const range = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 }
const executeQuery = shellProxy.ExecuteQuery as ReturnType<typeof vi.fn>

async function run(text: string) {
  const tabStore = useTabStore()
  vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
  vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
    serverId: SERVER_ID,
    activeInnerTabId: QUERY_ID,
  } as never)
  const store = useQueryStore()
  store.initQueryState(QUERY_ID, 'mydb')
  executeQuery.mockResolvedValue({
    isSuccess: true,
    data: { documents: [], rawOutput: '', output: [] },
  })
  await store.executeQuery(QUERY_ID, { text, range })
  return store.getQueryState(QUERY_ID)
}

describe('queryStore use command', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    executeQuery.mockReset()
  })

  test.each([
    ['nothing else', 'use other'],
    ['a comment', 'use other\n// just a note'],
    ['several comments and blank lines', 'use other\n\n// one\n\n  // two\n'],
  ])('switches database without running anything when followed by %s', async (_, text) => {
    const state = await run(text)
    expect(state.selectedDatabase).toBe('other')
    expect(executeQuery).not.toHaveBeenCalled()
  })

  test('runs the code that follows the use line', async () => {
    const state = await run('use other\n// note\ndb.c.find()')
    expect(state.selectedDatabase).toBe('other')
    expect(executeQuery).toHaveBeenCalledOnce()
    expect(executeQuery.mock.calls[0]).toContain('// note\ndb.c.find()')
  })

  // The slash run used to take exponential time in the comment check and freeze the UI.
  test('a long divider comment before code does not hang', async () => {
    const divider = '/'.repeat(200)
    await run(`use other\n${divider}\ndb.c.find()`)
    expect(executeQuery).toHaveBeenCalledOnce()
  })
})
