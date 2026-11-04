// @vitest-environment happy-dom
//
// Opening a file replaces the editor's content, so a tab with edits must only go
// on to open one if they were saved or the user chose to discard them. Edits go
// through the editor's change handler, so the dirty flag is set as in the real app.
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ref, shallowRef } from 'vue'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive from 'naive-ui'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'

type EditorAction = { id: string; run: () => void }
type WarningOptions = { onPositiveClick: () => Promise<void>; onNegativeClick: () => void }

const editorState = vi.hoisted(() => ({
  value: '',
  listeners: [] as (() => void)[],
  actions: [] as { id: string; run: () => void }[],
}))
const dialogWarning = vi.fn<(options: WarningOptions) => void>()

function setEditorValue(value: string) {
  editorState.value = value
  editorState.listeners.forEach((listener) => listener())
}

vi.mock('naive-ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('naive-ui')>()),
  useDialog: () => ({ warning: dialogWarning }),
}))

vi.mock('@/features/queries/useMonacoEditor', () => ({
  useMonacoEditor: (options: { value?: string }) => {
    editorState.value = options.value ?? ''
    editorState.listeners = []
    editorState.actions = []
    return {
      container: ref(document.createElement('div')),
      editor: shallowRef({
        getValue: () => editorState.value,
        setValue: (value: string) => {
          editorState.value = value
          editorState.listeners.forEach((listener) => listener())
        },
        addAction: (action: EditorAction) => editorState.actions.push(action),
        onDidChangeModelContent: (listener: () => void) => editorState.listeners.push(listener),
        focus: vi.fn(),
        getModel: () => null,
        dispose: vi.fn(),
      }),
    }
  },
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

const FILE_PATH = '/q.js'
const ON_DISK = 'content on disk'
const EDITED = 'unsaved edits'

const i18n = createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })

let queryId: string
let wrapper: VueWrapper | undefined

const isDirty = () => useQueryStore().getQueryState(queryId).isDirty

async function mountTab() {
  const QueryTab = (await import('@/features/queries/QueryTab.vue')).default
  wrapper = mount(QueryTab, { props: { queryId }, global: { plugins: [naive, i18n] } })
  await flushPromises()
}

async function loadFile() {
  vi.mocked(filesProxy.ReadFile).mockResolvedValue({ isSuccess: true, data: ON_DISK } as never)
  await useQueryStore().loadFileByPath(queryId, FILE_PATH)
  await flushPromises()
}

async function pressOpen() {
  editorState.actions.find((action) => action.id === 'vervet.openFile')!.run()
  await flushPromises()
}

/** Presses Ctrl+O where edits are expected, and returns the unsaved-changes prompt it raises. */
async function pressOpenAndGetPrompt(): Promise<WarningOptions> {
  await pressOpen()
  return dialogWarning.mock.calls[0]![0]
}

beforeEach(() => {
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

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('dirty state of an untitled QueryTab', () => {
  it('starts clean with the default template', async () => {
    await mountTab()

    expect(editorState.value).toContain('// MongoDB Query')
    expect(isDirty()).toBe(false)
  })

  it('starts clean with the text it was opened with', async () => {
    queryId = useTabStore().openQuery('server-1', 'shop', 'db.users.find({})')!
    await mountTab()

    expect(isDirty()).toBe(false)
  })

  it('becomes dirty when edited, and clean again when the edit is undone', async () => {
    await mountTab()
    const original = editorState.value

    setEditorValue(EDITED)
    expect(isDirty()).toBe(true)

    setEditorValue(original)
    expect(isDirty()).toBe(false)
  })

  it('stays dirty after a remount', async () => {
    await mountTab()
    setEditorValue(EDITED)
    wrapper!.unmount()

    await mountTab()

    expect(editorState.value).toBe(EDITED)
    expect(isDirty()).toBe(true)
  })

  it('becomes clean once saved', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({ isSuccess: true, data: FILE_PATH } as never)
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: true } as never)
    await mountTab()
    setEditorValue(EDITED)

    await useQueryStore().saveFile(queryId, EDITED)

    expect(filesProxy.WriteFile).toHaveBeenCalledWith(FILE_PATH, EDITED)
    expect(isDirty()).toBe(false)
  })
})

describe('opening a file from an untitled QueryTab', () => {
  it('goes straight to the file picker when there are no edits', async () => {
    await mountTab()

    await pressOpen()

    expect(dialogWarning).not.toHaveBeenCalled()
    expect(filesProxy.SelectFile).toHaveBeenCalledOnce()
  })

  it('prompts when there are edits, and keeps them if the save dialog is cancelled', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({ isSuccess: true, data: '' } as never)
    await mountTab()
    setEditorValue(EDITED)
    const prompt = await pressOpenAndGetPrompt()

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.SelectFile).not.toHaveBeenCalled()
    expect(notifyError).not.toHaveBeenCalled()
    expect(editorState.value).toBe(EDITED)
    expect(isDirty()).toBe(true)
  })

  it('replaces the edits with the chosen file when the user discards them', async () => {
    vi.mocked(filesProxy.SelectFile).mockResolvedValue({ isSuccess: true, data: FILE_PATH } as never)
    vi.mocked(filesProxy.ReadFile).mockResolvedValue({ isSuccess: true, data: ON_DISK } as never)
    await mountTab()
    setEditorValue(EDITED)
    const prompt = await pressOpenAndGetPrompt()

    prompt.onNegativeClick()
    await flushPromises()

    expect(editorState.value).toBe(ON_DISK)
    expect(isDirty()).toBe(false)
  })
})

describe('opening a file from a QueryTab with a saved file', () => {
  beforeEach(async () => {
    await mountTab()
    await loadFile()
    setEditorValue(EDITED)
  })

  it('does not open a file when the save fails', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: false, errorDetail: 'read-only' } as never)
    const prompt = await pressOpenAndGetPrompt()

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.SelectFile).not.toHaveBeenCalled()
    expect(notifyError).toHaveBeenCalledOnce()
    expect(editorState.value).toBe(EDITED)
    expect(isDirty()).toBe(true)
  })

  it('opens a file once the save succeeds', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: true } as never)
    const prompt = await pressOpenAndGetPrompt()

    await prompt.onPositiveClick()
    await flushPromises()

    expect(filesProxy.WriteFile).toHaveBeenCalledWith(FILE_PATH, EDITED)
    expect(filesProxy.SelectFile).toHaveBeenCalledOnce()
  })

  it('opens a file when the user discards their edits', async () => {
    const prompt = await pressOpenAndGetPrompt()

    prompt.onNegativeClick()
    await flushPromises()

    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
    expect(filesProxy.SelectFile).toHaveBeenCalledOnce()
  })

  it('shows the file again when the user discards their edits and reopens it', async () => {
    vi.mocked(filesProxy.SelectFile).mockResolvedValue({ isSuccess: true, data: FILE_PATH } as never)
    const prompt = await pressOpenAndGetPrompt()

    prompt.onNegativeClick()
    await flushPromises()

    expect(editorState.value).toBe(ON_DISK)
    expect(useQueryStore().getQueryState(queryId).currentContent).toBe(ON_DISK)
    expect(isDirty()).toBe(false)
  })

  it('keeps the edits when the user discards them but cancels the file picker', async () => {
    const prompt = await pressOpenAndGetPrompt()

    prompt.onNegativeClick()
    await flushPromises()

    expect(editorState.value).toBe(EDITED)
    expect(isDirty()).toBe(true)
  })
})
