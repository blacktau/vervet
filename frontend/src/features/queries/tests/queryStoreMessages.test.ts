import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test } from 'vitest'
import { useQueryStore } from '../queryStore'
import type { LogMessage } from '../queryStore'

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
