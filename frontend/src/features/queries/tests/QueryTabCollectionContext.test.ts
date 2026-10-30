// @vitest-environment happy-dom
//
// The edit and delete dialogs write through the collection context QueryTab
// passes to the results table. It must point at the database and collection
// the results came from, not the ones the query tab was opened on.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { defineComponent, h, ref, shallowRef } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive, { NDialogProvider } from 'naive-ui'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'

vi.mock('@/features/queries/useMonacoEditor', () => ({
  useMonacoEditor: () => ({
    container: ref(document.createElement('div')),
    editor: shallowRef({
      getValue: () => '',
      setValue: vi.fn(),
      addAction: vi.fn(),
      onDidChangeModelContent: vi.fn(),
      focus: vi.fn(),
      getModel: () => null,
      dispose: vi.fn(),
    }),
  }),
}))

vi.mock('monaco-editor', () => ({
  KeyMod: { CtrlCmd: 2048, Shift: 1024 },
  KeyCode: { KeyO: 45, KeyS: 49, Enter: 3, F5: 68 },
}))

vi.mock('@/features/data-browser/browserStore', () => ({
  useDataBrowserStore: () => ({ connections: [], getDatabaseList: vi.fn() }),
}))

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  CheckMongosh: vi.fn().mockResolvedValue({ isSuccess: true, data: true }),
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(),
}))

vi.mock('wailsjs/go/api/FilesProxy', () => ({
  SelectFile: vi.fn(),
  ReadFile: vi.fn(),
  WriteFile: vi.fn(),
  SaveFile: vi.fn(),
}))

const i18n = createI18n({ legacy: false, locale: 'en-GB', messages: { 'en-GB': {} } })

const TableStub = defineComponent({
  name: 'DocumentTreeTable',
  props: ['collectionContext'],
  render: () => h('div'),
})

async function mountQueryTab(queryId: string) {
  const QueryTab = (await import('@/features/queries/QueryTab.vue')).default
  const wrapper = mount(
    { render: () => h(NDialogProvider, null, { default: () => h(QueryTab, { queryId }) }) },
    { global: { plugins: [naive, i18n], stubs: { DocumentTreeTable: TableStub } } },
  )
  await flushPromises()
  return wrapper
}

let queryId: string

beforeEach(() => {
  setActivePinia(createPinia())
  const tabStore = useTabStore()
  tabStore.tabItems = [
    {
      serverId: 'server-1',
      name: 'Server 1',
      queries: [],
      innerTabOrder: [],
      indexTabs: [],
      statisticsTabs: [],
      schemaTabs: [],
    },
  ] as unknown as typeof tabStore.tabItems
  tabStore.activeTabIndex = 0
  queryId = tabStore.openQuery('server-1', 'shop', 'db.orders.find()', 'orders')!
})

function showResults(database: string, collection?: string) {
  const state = useQueryStore().getQueryState(queryId)
  state.documents = [{ _id: 1 }]
  state.resultsDatabase = database
  state.pageContext = collection ? { collection } : null
}

describe('QueryTab collection context', () => {
  it('targets the database the results were read from', async () => {
    const wrapper = await mountQueryTab(queryId)
    useQueryStore().setDatabase(queryId, 'archive')
    showResults('archive')
    await flushPromises()

    expect(wrapper.findComponent(TableStub).props('collectionContext')).toEqual({
      serverId: 'server-1',
      dbName: 'archive',
      collectionName: 'orders',
      collectionFromTab: true,
    })
    wrapper.unmount()
  })

  it('keeps targeting that database after the dropdown changes', async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('shop')
    useQueryStore().setDatabase(queryId, 'archive')
    await flushPromises()

    expect(wrapper.findComponent(TableStub).props('collectionContext').dbName).toBe('shop')
    wrapper.unmount()
  })

  it('targets the collection a paged find read from', async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('shop', 'customers')
    await flushPromises()

    const context = wrapper.findComponent(TableStub).props('collectionContext')
    expect(context.collectionName).toBe('customers')
    expect(context.collectionFromTab).toBe(false)
    wrapper.unmount()
  })
})
