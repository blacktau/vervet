import { describe, expect, it, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('wailsjs/go/api/SettingsProxy', () => ({
  GetSettings: vi.fn(),
  SetSettings: vi.fn(),
  ResetSettings: vi.fn(),
  GetAvailableFonts: vi.fn(),
}))

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  CheckMongosh: vi.fn(),
  ExecuteQuery: vi.fn(),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(),
  CancelQuery: vi.fn(),
}))

vi.mock('wailsjs/go/api/FilesProxy', () => ({
  SelectFile: vi.fn(),
  ReadFile: vi.fn(),
  SaveFile: vi.fn(),
  WriteFile: vi.fn(),
}))

vi.mock('naive-ui', () => ({
  useOsTheme: () => ({ value: 'light' }),
}))

vi.mock('@/utils/dialog.ts', () => ({
  useDialoger: () => ({ confirm: vi.fn(), warning: vi.fn() }),
  useNotifier: () => ({ error: vi.fn(), success: vi.fn(), info: vi.fn() }),
}))

import { useTabStore } from './tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import type { ServerTabItem } from '@/types/ServerTabItem.ts'

function seedServerTab(serverId: string): ServerTabItem {
  return {
    title: serverId,
    blank: false,
    serverId,
    queries: [],
  }
}

describe('tabs.duplicateQuery', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('inserts a new query tab immediately after the source', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const firstId = tabs.openQuery('s1', 'mydb', 'first')!
    const secondId = tabs.openQuery('s1', 'otherdb', 'second')!

    const dupId = tabs.duplicateQuery('s1', firstId)
    expect(dupId).toBeTruthy()

    const ids = tabs.tabItems[0]!.queries.map((q) => q.id)
    expect(ids).toEqual([firstId, dupId, secondId])
  })

  it('copies database and collectionName, leaves filePath unset', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const srcId = tabs.openQuery('s1', 'mydb', 'first', 'mycoll')!
    tabs.tabItems[0]!.queries[0]!.filePath = '/tmp/q.js'

    const dupId = tabs.duplicateQuery('s1', srcId)!
    const dup = tabs.tabItems[0]!.queries.find((q) => q.id === dupId)!
    expect(dup.database).toBe('mydb')
    expect(dup.collectionName).toBe('mycoll')
    expect(dup.filePath).toBeUndefined()
  })

  it('seeds initialText with the source query store live content', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const srcId = tabs.openQuery('s1', 'mydb')!
    const queries = useQueryStore()
    queries.initQueryState(srcId, 'mydb')
    queries.setCurrentContent(srcId, 'db.foo.find({live: true})')

    const dupId = tabs.duplicateQuery('s1', srcId)!
    const dup = tabs.tabItems[0]!.queries.find((q) => q.id === dupId)!
    expect(dup.initialText).toBe('db.foo.find({live: true})')
  })

  it('activates the new tab and marks pendingFocusQueryId', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const srcId = tabs.openQuery('s1', 'mydb')!

    const dupId = tabs.duplicateQuery('s1', srcId)!
    expect(tabs.tabItems[0]!.activeInnerTabId).toBe(dupId)
    expect(tabs.pendingFocusQueryId).toBe(dupId)
  })

  it('switches nav to Browser via _setActivatedIndex', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const srcId = tabs.openQuery('s1', 'mydb')!
    // NavType is a const enum (inlined by tsc, not accessible at vitest runtime)
    tabs.nav = 'servers' as never

    tabs.duplicateQuery('s1', srcId)
    expect(tabs.nav).toBe('browser')
  })

  it('returns undefined for unknown serverId or queryId', () => {
    const tabs = useTabStore()
    tabs.tabItems = [seedServerTab('s1')]
    const srcId = tabs.openQuery('s1', 'mydb')!

    expect(tabs.duplicateQuery('missing', srcId)).toBeUndefined()
    expect(tabs.duplicateQuery('s1', 'query-999999')).toBeUndefined()
  })
})
