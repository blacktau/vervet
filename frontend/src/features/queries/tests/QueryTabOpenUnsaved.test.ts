// @vitest-environment happy-dom
//
// Opening a file replaces the editor's content, so a dirty tab must only go on
// to open one if the user's edits were saved or they chose to discard them.
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive from 'naive-ui'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'

type EditorAction = { id: string; run: () => void }
type WarningOptions = { onPositiveClick: () => Promise<void>; onNegativeClick: () => void }

const actions: EditorAction[] = []
const dialogWarning = vi.fn<(options: WarningOptions) => void>()

vi.mock('naive-ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('naive-ui')>()),
  useDialog: () => ({ warning: dialogWarning }),
}))

vi.mock('@/features/queries/useMonacoEditor', () => ({
  useMonacoEditor: () => ({
    container: ref(document.createElement('div')),
    editor: shallowRef({
      getValue: () => 'unsaved edits',
      setValue: vi.fn(),
      addAction: (action: EditorAction) => actions.push(action),
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

const notifyError = vi.fn()
vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ error: notifyError, warning: vi.fn(), info: vi.fn(), success: vi.fn() }),
  useDialoger: () => ({}),
  useMessager: () => ({}),
}))

import * as filesProxy from 'wailsjs/go/api/FilesProxy'

const i18n = createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })

let queryId: string

/** Presses Ctrl+O in a dirty tab, and returns the unsaved-changes prompt it raises. */
async function pressOpenInDirtyTab(filePath?: string): Promise<WarningOptions> {
  const QueryTab = (await import('@/features/queries/QueryTab.vue')).default
  mount(QueryTab, { props: { queryId }, global: { plugins: [naive, i18n] } })
  await flushPromises()

  const queryStore = useQueryStore()
  if (filePath) {
    queryStore.setFilePath(queryId, filePath)
  }
  queryStore.setDirty(queryId, true)

  actions.find((action) => action.id === 'vervet.openFile')!.run()
  await flushPromises()
  return dialogWarning.mock.calls[0]![0]
}

beforeEach(() => {
  actions.length = 0
  vi.clearAllMocks()
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
  vi.mocked(filesProxy.SelectFile).mockResolvedValue({ isSuccess: true, data: '' } as never)
})

describe('opening a file from a dirty QueryTab', () => {
  it('does not open a file when the save fails', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: false, errorDetail: 'read-only' } as never)
    const prompt = await pressOpenInDirtyTab('/q.js')

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.SelectFile).not.toHaveBeenCalled()
    expect(notifyError).toHaveBeenCalledOnce()
    expect(useQueryStore().getQueryState(queryId).isDirty).toBe(true)
  })

  it('does not open a file when the save dialog is cancelled', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({ isSuccess: true, data: '' } as never)
    const prompt = await pressOpenInDirtyTab()

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.SelectFile).not.toHaveBeenCalled()
    expect(notifyError).not.toHaveBeenCalled()
  })

  it('opens a file once the save succeeds', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: true } as never)
    const prompt = await pressOpenInDirtyTab('/q.js')

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.WriteFile).toHaveBeenCalledWith('/q.js', 'unsaved edits')
    expect(filesProxy.SelectFile).toHaveBeenCalledOnce()
  })

  it('opens a file when the user discards their edits', async () => {
    const prompt = await pressOpenInDirtyTab('/q.js')

    prompt.onNegativeClick()
    await flushPromises()

    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
    expect(filesProxy.SelectFile).toHaveBeenCalledOnce()
  })
})
