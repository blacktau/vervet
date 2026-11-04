import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(async () => undefined),
  CheckMongosh: vi.fn(async () => ({ isSuccess: true, data: true })),
}))

import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import { useQueryStore } from '../queryStore'
import type { LogMessage } from '../queryStore'
import { useTabStore } from '@/features/tabs/tabs'

describe('queryStore message model', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('starts with an empty messages array and default filter', () => {
    const store = useQueryStore()
    const state = store.getQueryState('q1')
    expect(state.messages).toEqual([])
    expect(state.messageFilter).toEqual({ info: true, warning: true, error: true })
  })

  test('appendMessage adds a typed LogMessage with id and timestamp', () => {
    const store = useQueryStore()
    store.appendMessage('q1', { level: 'info', text: 'hello' })
    const messages = store.getQueryState('q1').messages
    expect(messages).toHaveLength(1)
    const msg = messages[0] as LogMessage
    expect(msg.level).toBe('info')
    expect(msg.text).toBe('hello')
    expect(typeof msg.id).toBe('string')
    expect(msg.id.length).toBeGreaterThan(0)
    expect(typeof msg.timestamp).toBe('string')
    expect(msg.query).toBeUndefined()
  })

  test('appendMessage threads the query payload when provided', () => {
    const store = useQueryStore()
    const range = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 5 }
    store.appendMessage('q1', { level: 'error', text: 'boom', query: { text: 'find', range } })
    const msg = store.getQueryState('q1').messages[0] as LogMessage
    expect(msg.query).toEqual({ text: 'find', range })
  })

  test('clearMessages empties the array', () => {
    const store = useQueryStore()
    store.appendMessage('q1', { level: 'info', text: 'a' })
    store.appendMessage('q1', { level: 'warning', text: 'b' })
    store.clearMessages('q1')
    expect(store.getQueryState('q1').messages).toEqual([])
  })

  test('setMessageFilter toggles a level', () => {
    const store = useQueryStore()
    store.getQueryState('q1')
    store.setMessageFilter('q1', 'warning', false)
    expect(store.getQueryState('q1').messageFilter.warning).toBe(false)
    expect(store.getQueryState('q1').messageFilter.info).toBe(true)
  })
})

describe('queryStore result summary', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  async function summaryOf(data: Record<string, unknown>) {
    const tabStore = useTabStore()
    vi.spyOn(tabStore, 'currentTabId', 'get').mockReturnValue('srv-1')
    vi.spyOn(tabStore, 'currentTab', 'get').mockReturnValue({
      serverId: 'srv-1',
      activeInnerTabId: 'q1',
    } as never)
    const store = useQueryStore()
    store.initQueryState('q1', 'mydb')
    vi.mocked(shellProxy.ExecuteQuery).mockResolvedValue({ isSuccess: true, data } as never)
    await store.executeQuery('q1', {
      text: 'db.c.run()',
      range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 },
    })
    const messages = store.getQueryState('q1').messages
    return (messages[messages.length - 1] as LogMessage).text
  }

  const ack = [{ acknowledged: true }]

  test('a write reports how many documents it affected, not its one result document', async () => {
    const text = await summaryOf({ documents: ack, operationType: 'updateMany', affectedCount: 3 })
    expect(text).toMatch(/^3 document\(s\) updated in /)
  })

  // A zero count is left out of the JSON, so it arrives as undefined.
  test('a write that changed nothing reports zero', async () => {
    const text = await summaryOf({ documents: ack, operationType: 'updateMany' })
    expect(text).toMatch(/^0 document\(s\) updated in /)
  })

  test.each(['find', 'findOne', 'aggregate', 'listIndexes'])(
    '%s counts its documents',
    async (op) => {
      const text = await summaryOf({ documents: [{ a: 1 }, { a: 2 }], operationType: op })
      expect(text).toMatch(/^2 /)
    },
  )

  test('a result with no operation is summarised as documents returned', async () => {
    const text = await summaryOf({ documents: [{ a: 1 }, { a: 2 }] })
    expect(text).toMatch(/^2 document\(s\) returned in /)
  })
})
