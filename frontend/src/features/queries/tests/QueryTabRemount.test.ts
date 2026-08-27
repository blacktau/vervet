// @vitest-environment happy-dom
//
// Regression cover for #296: switching between connected server tabs unmounts
// every QueryTab, so the editor must be re-seeded from the query store rather
// than from the tab item's initialText, which is consumed on first mount.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { h, ref, shallowRef, nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive, { NDialogProvider } from 'naive-ui'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'

// Records the `value` every editor is constructed with — the thing under test.
const seededValues: string[] = []
let editorContent = ''

function fakeEditor() {
  return {
    getValue: () => editorContent,
    setValue: (v: string) => {
      editorContent = v
    },
    addAction: vi.fn(),
    onDidChangeModelContent: vi.fn(),
    focus: vi.fn(),
    getModel: () => null,
    dispose: vi.fn(),
  }
}

vi.mock('@/features/queries/useMonacoEditor', () => ({
  useMonacoEditor: (options: { value?: string }) => {
    seededValues.push(options.value ?? '')
    editorContent = options.value ?? ''
    return { container: ref(document.createElement('div')), editor: shallowRef(fakeEditor()) }
  },
}))

// Pulled in for KeyMod/KeyCode only; the real barrel loads language
// contributions asynchronously and races Vitest's teardown.
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

async function mountQueryTab(queryId: string) {
  const QueryTab = (await import('@/features/queries/QueryTab.vue')).default
  // QueryTab calls useDialog(), which needs a provider above it.
  const wrapper = mount(
    { render: () => h(NDialogProvider, null, { default: () => h(QueryTab, { queryId }) }) },
    { global: { plugins: [naive, i18n] } },
  )
  // onMounted awaits CheckMongosh and the database list before it writes the
  // editor's value back to the store; let that tail finish before assertions.
  await flushPromises()
  return wrapper
}

let queryId: string

beforeEach(() => {
  seededValues.length = 0
  editorContent = ''
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
  queryId = tabStore.openQuery('server-1', 'shop')!
})

describe('QueryTab editor seeding', () => {
  it('seeds a new tab with the default template', async () => {
    const wrapper = await mountQueryTab(queryId)
    expect(seededValues[0]).toContain('// MongoDB Query')
    wrapper.unmount()
  })

  it('seeds from initialText when the tab was opened with one', async () => {
    const tabStore = useTabStore()
    const withText = tabStore.openQuery('server-1', 'shop', 'db.users.find({})')!
    const wrapper = await mountQueryTab(withText)
    expect(seededValues[0]).toBe('db.users.find({})')
    wrapper.unmount()
  })

  it('restores the stored content when the tab is remounted', async () => {
    const queryStore = useQueryStore()
    const first = await mountQueryTab(queryId)
    // Stand in for the user typing: the editor's change handler is what writes
    // this in the real app.
    queryStore.setCurrentContent(queryId, 'db.orders.find({ total: { $gt: 10 } })')

    // Switching server tabs tears the pane down and builds it again.
    first.unmount()
    const second = await mountQueryTab(queryId)

    expect(seededValues[1]).toBe('db.orders.find({ total: { $gt: 10 } })')
    second.unmount()
  })

  it('does not fall back to the default template on remount', async () => {
    const queryStore = useQueryStore()
    const first = await mountQueryTab(queryId)
    queryStore.setCurrentContent(queryId, 'db.users.countDocuments()')
    first.unmount()

    const second = await mountQueryTab(queryId)
    expect(seededValues[1]).not.toContain('// MongoDB Query')
    second.unmount()
  })

  it('keeps unsaved edits instead of resetting to the file on disk', async () => {
    const queryStore = useQueryStore()
    const first = await mountQueryTab(queryId)
    queryStore.setFilePath(queryId, '/tmp/query.js')
    queryStore.setSavedContent(queryId, 'db.users.find({})')
    queryStore.setCurrentContent(queryId, 'db.users.find({ name: "alice" })')
    queryStore.setDirty(queryId, true)
    first.unmount()

    const second = await mountQueryTab(queryId)
    await nextTick()
    // The savedContent watcher fires when the editor appears; it must not
    // overwrite the dirty buffer with the file's contents.
    expect(editorContent).toBe('db.users.find({ name: "alice" })')
    second.unmount()
  })

  it('still loads file content into a clean editor', async () => {
    const queryStore = useQueryStore()
    const wrapper = await mountQueryTab(queryId)
    queryStore.setSavedContent(queryId, 'db.orders.find({})')
    await nextTick()
    expect(editorContent).toBe('db.orders.find({})')
    wrapper.unmount()
  })
})
