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

async function run(response: unknown) {
  const tabStore = useTabStore()
  vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue(SERVER_ID)
  vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
    serverId: SERVER_ID,
    activeInnerTabId: QUERY_ID,
  } as never)
  const store = useQueryStore()
  store.initQueryState(QUERY_ID, 'mydb')
  executeQuery.mockResolvedValue(response)
  await store.executeQuery(QUERY_ID, { text: 'script', range })
  return store.getQueryState(QUERY_ID)
}

const printed = [
  { level: 'info', text: 'Top users:' },
  { level: 'warning', text: 'slow' },
]

describe('queryStore printed output', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    executeQuery.mockReset()
  })

  test('printed lines land in Messages, in order, between Executing and the result summary', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [{ a: 1 }], rawOutput: '', output: printed },
    })
    const texts = state.messages.map((m) => [m.level, m.text])
    expect(texts[1]).toEqual(['info', 'Top users:'])
    expect(texts[2]).toEqual(['warning', 'slow'])
    expect(texts).toHaveLength(4)
    expect(state.messages[1]!.query?.text).toBe('script')
  })

  test('with a result, Results stays open and the Messages tab gets a badge', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [{ a: 1 }], rawOutput: '', output: printed },
    })
    expect(state.activeResultTab).toBe('results')
    expect(state.documents).toEqual([{ a: 1 }])
    expect(state.unseenOutput).toEqual({ count: 2, level: 'warning' })
  })

  test('a text result is kept alongside the printed lines', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [], rawOutput: '5', output: [{ level: 'info', text: 'checking' }] },
    })
    expect(state.rawOutput).toBe('5')
    expect(state.activeResultTab).toBe('results')
    expect(state.unseenOutput).toEqual({ count: 1, level: 'info' })
  })

  test('an empty cursor is still a result', async () => {
    const state = await run({
      isSuccess: true,
      data: {
        documents: [],
        rawOutput: '',
        output: [{ level: 'info', text: 'x' }],
        pageContext: { database: 'mydb', collection: 'none' },
      },
    })
    expect(state.activeResultTab).toBe('results')
    expect(state.unseenOutput).toEqual({ count: 1, level: 'info' })
  })

  test('a print-only script opens Messages, with no badge', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [], rawOutput: '', output: [{ level: 'error', text: 'bad' }] },
    })
    expect(state.activeResultTab).toBe('messages')
    expect(state.unseenOutput).toBeNull()
  })

  test('a script with no result is summarised as completed, not as 0 documents', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [], rawOutput: '', output: [{ level: 'info', text: 'done' }] },
    })
    expect(state.messages.at(-1)!.text).toMatch(/^Query completed in /)
  })

  test('no output means no badge', async () => {
    const state = await run({ isSuccess: true, data: { documents: [{ a: 1 }], rawOutput: '' } })
    expect(state.unseenOutput).toBeNull()
  })

  test('a failed script shows what it printed before the error', async () => {
    const state = await run({
      isSuccess: false,
      errorCode: 'unknown',
      errorDetail: 'script error: Wrong database',
      data: { documents: [], rawOutput: '', output: [{ level: 'info', text: 'checked 25' }] },
    })
    const levels = state.messages.map((m) => m.level)
    expect(state.messages[1]!.text).toBe('checked 25')
    expect(levels.slice(2).every((l) => l === 'error')).toBe(true)
    expect(state.activeResultTab).toBe('messages')
  })

  test('opening Messages clears the badge, and a new run resets it', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [{ a: 1 }], rawOutput: '', output: printed },
    })
    const store = useQueryStore()
    store.markOutputSeen(QUERY_ID)
    expect(state.unseenOutput).toBeNull()

    state.unseenOutput = { count: 1, level: 'info' }
    executeQuery.mockResolvedValue({ isSuccess: true, data: { documents: [{ a: 1 }], rawOutput: '' } })
    await store.executeQuery(QUERY_ID, { text: 'script', range })
    expect(state.unseenOutput).toBeNull()
  })

  test('clearing messages clears the badge', async () => {
    const state = await run({
      isSuccess: true,
      data: { documents: [{ a: 1 }], rawOutput: '', output: printed },
    })
    useQueryStore().clearMessages(QUERY_ID)
    expect(state.unseenOutput).toBeNull()
  })
})
