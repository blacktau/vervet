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
import * as shellProxy from 'wailsjs/go/api/ShellProxy'

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

function showResults(database: string, collection?: string, wholeDocuments = true) {
  const state = useQueryStore().getQueryState(queryId)
  state.documents = [{ _id: 1 }]
  state.resultsNamespace = collection ? { database, collection, wholeDocuments } : null
  state.pageContext = collection ? { database, collection } : null
}

describe('QueryTab collection context', () => {
  it('turns off edit, but not insert or delete, when documents may be partial', async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('shop', 'orders', false)
    await flushPromises()

    const context = wrapper.findComponent(TableStub).props('collectionContext')
    expect(context.canEdit).toBe(false)
    expect(context.dbName).toBe('shop')
    wrapper.unmount()
  })

  it('allows edits on a tab opened without a collection when the result names one', async () => {
    const tabStore = useTabStore()
    queryId = tabStore.openQuery('server-1', 'shop', 'db.orders.find()')!
    const wrapper = await mountQueryTab(queryId)
    showResults('shop', 'orders')
    await flushPromises()

    expect(wrapper.findComponent(TableStub).props('collectionContext')).toEqual({
      serverId: 'server-1',
      dbName: 'shop',
      collectionName: 'orders',
      canEdit: true,
    })
    wrapper.unmount()
  })

  it('targets the namespace the results came from, not the tab', async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('archive', 'customers')
    await flushPromises()

    const context = wrapper.findComponent(TableStub).props('collectionContext')
    expect(context.dbName).toBe('archive')
    expect(context.collectionName).toBe('customers')
    wrapper.unmount()
  })

  it('keeps targeting that database after the dropdown changes', async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('shop', 'orders')
    useQueryStore().setDatabase(queryId, 'archive')
    await flushPromises()

    expect(wrapper.findComponent(TableStub).props('collectionContext').dbName).toBe('shop')
    wrapper.unmount()
  })

  it("does not fall back to the tab's collection when results have no namespace", async () => {
    const wrapper = await mountQueryTab(queryId)
    showResults('shop')
    await flushPromises()

    expect(wrapper.findComponent(TableStub).props('collectionContext')).toBeUndefined()
    wrapper.unmount()
  })
})

describe('QueryTab pager', () => {
  it('keeps a pending page-size change when a page is clicked', async () => {
    vi.mocked(shellProxy.FetchPage).mockResolvedValue({
      isSuccess: true,
      data: { documents: [{ _id: 1 }] },
    } as never)
    const wrapper = await mountQueryTab(queryId)
    showResults('shop', 'orders')
    const state = useQueryStore().getQueryState(queryId)
    state.pageSize = 25
    state.requestedPageSize = 50 // a page-size change whose response hasn't arrived
    await flushPromises()

    wrapper.findComponent(TableStub).vm.$emit('update:page', 3)
    await flushPromises()

    expect(vi.mocked(shellProxy.FetchPage).mock.lastCall!.slice(2)).toEqual([3, 50])
    wrapper.unmount()
  })
})
