// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

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

type Rendered = () => unknown
type DialogOptions = { content: Rendered; action: Rendered; onAfterLeave: () => void }

const dialogShow = vi.fn<(options: DialogOptions) => { destroy: () => void }>()
const confirmDisconnect = vi.fn()
const notifyError = vi.fn()
vi.mock('@/utils/dialog', () => ({
  useDialoger: () => ({ show: dialogShow, warning: confirmDisconnect }),
  useNotifier: () => ({ error: notifyError, warning: vi.fn(), info: vi.fn(), success: vi.fn() }),
  useMessager: () => ({}),
}))

import * as filesProxy from 'wailsjs/go/api/FilesProxy'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'

const SERVER = 'server-1'

function renderToWrapper(render: Rendered): VueWrapper {
  return mount({ render })
}

async function clickButton(label: string) {
  const options = dialogShow.mock.calls[0]![0]
  const button = renderToWrapper(options.action)
    .findAll('button')
    .find((candidate) => candidate.text() === label)
  await button!.trigger('click')
  await flushPromises()
}

function promptOptions() {
  return dialogShow.mock.calls[0]![0]
}

function buttonLabels() {
  return renderToWrapper(promptOptions().action)
    .findAll('button')
    .map((button) => button.text())
}

/** An unsaved query, in a tab with a file if a path is given. */
function addDirtyQuery(filePath?: string, serverId: string = SERVER): string {
  const id = useTabStore().openQuery(serverId, 'shop')!
  const queryStore = useQueryStore()
  queryStore.setSavedContent(id, 'original')
  if (filePath) {
    queryStore.setFilePath(id, filePath)
  }
  queryStore.setCurrentContent(id, `edits to ${id}`)
  queryStore.setDirty(id, true)
  return id
}

function addSecondServer() {
  useTabStore().tabItems.push({
    serverId: 'server-2',
    title: 'Server 2',
    blank: false,
    queries: [],
    innerTabOrder: [],
  } as never)
}

beforeEach(() => {
  vi.clearAllMocks()
  setActivePinia(createPinia())
  const tabStore = useTabStore()
  tabStore.tabItems = [
    {
      serverId: SERVER,
      title: 'Server 1',
      blank: false,
      queries: [],
      innerTabOrder: [],
    },
  ] as unknown as typeof tabStore.tabItems
  tabStore.activeTabIndex = 0
  dialogShow.mockImplementation(() => ({ destroy: vi.fn() }))
  vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: true } as never)
})

describe('closeTab with unsaved queries', () => {
  it('goes straight to the disconnect confirmation when nothing is unsaved', async () => {
    useTabStore().openQuery(SERVER, 'shop')

    await useTabStore().closeTab(SERVER)

    expect(dialogShow).not.toHaveBeenCalled()
    expect(confirmDisconnect).toHaveBeenCalledOnce()
  })

  it('asks once about all of them, naming each', async () => {
    addDirtyQuery('/queries/a.js')
    addDirtyQuery()
    addDirtyQuery()

    void useTabStore().closeTab(SERVER)
    await flushPromises()

    expect(dialogShow).toHaveBeenCalledOnce()
    const content = renderToWrapper(promptOptions().content).text()
    expect(content).toContain('3 tabs have unsaved changes')
    expect(renderToWrapper(promptOptions().content).findAll('li').map((li) => li.text())).toEqual([
      'a.js',
      'shop 2',
      'shop 3',
    ])
    expect(buttonLabels()).toEqual(['Cancel', 'Discard All', 'Save All'])
  })

  it('asks about a single query by name', async () => {
    addDirtyQuery('/queries/a.js')

    void useTabStore().closeTab(SERVER)
    await flushPromises()

    expect(renderToWrapper(promptOptions().content).text()).toBe('Save changes to a.js?')
    expect(buttonLabels()).toEqual(['Cancel', "Don't Save", 'Save'])
  })

  it('writes nothing on Discard All, then asks to confirm the disconnect', async () => {
    addDirtyQuery('/queries/a.js')
    addDirtyQuery('/queries/b.js')

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Discard All')

    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
    expect(confirmDisconnect).toHaveBeenCalledOnce()
  })

  it('keeps everything on Cancel', async () => {
    const id = addDirtyQuery('/queries/a.js')
    addDirtyQuery('/queries/b.js')

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Cancel')

    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
    expect(confirmDisconnect).not.toHaveBeenCalled()
    expect(useQueryStore().getQueryState(id).isDirty).toBe(true)
  })

  it('treats closing the prompt another way as a cancel', async () => {
    addDirtyQuery('/queries/a.js')

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    promptOptions().onAfterLeave()
    await flushPromises()

    expect(confirmDisconnect).not.toHaveBeenCalled()
  })

  it('saves every query on Save All, then asks to confirm the disconnect', async () => {
    const a = addDirtyQuery('/queries/a.js')
    const b = addDirtyQuery('/queries/b.js')

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Save All')

    expect(filesProxy.WriteFile).toHaveBeenCalledWith('/queries/a.js', `edits to ${a}`)
    expect(filesProxy.WriteFile).toHaveBeenCalledWith('/queries/b.js', `edits to ${b}`)
    expect(filesProxy.SaveFile).not.toHaveBeenCalled()
    expect(confirmDisconnect).toHaveBeenCalledOnce()
  })

  it('shows each untitled query while asking where to save it', async () => {
    const first = addDirtyQuery()
    const second = addDirtyQuery()
    const shownDuringDialog: (string | undefined)[] = []
    vi.mocked(filesProxy.SaveFile).mockImplementation(async () => {
      shownDuringDialog.push(useTabStore().currentTab?.activeInnerTabId)
      return { isSuccess: true, data: `/queries/${shownDuringDialog.length}.js` } as never
    })

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Save All')

    expect(shownDuringDialog).toEqual([first, second])
    expect(confirmDisconnect).toHaveBeenCalledOnce()
  })

  it('stops at a cancelled save dialog, keeping earlier saves and the connection', async () => {
    const saved = addDirtyQuery('/queries/a.js')
    const untitled = addDirtyQuery()
    const notReached = addDirtyQuery('/queries/c.js')
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({ isSuccess: true, data: '' } as never)

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Save All')

    const queryStore = useQueryStore()
    expect(queryStore.getQueryState(saved).isDirty).toBe(false)
    expect(queryStore.getQueryState(untitled).isDirty).toBe(true)
    expect(queryStore.getQueryState(notReached).isDirty).toBe(true)
    expect(useTabStore().currentTab?.activeInnerTabId).toBe(untitled)
    expect(filesProxy.WriteFile).toHaveBeenCalledOnce()
    expect(notifyError).not.toHaveBeenCalled()
    expect(confirmDisconnect).not.toHaveBeenCalled()
  })

  it('stops at a failed save and reports it', async () => {
    addDirtyQuery('/queries/a.js')
    const notReached = addDirtyQuery('/queries/b.js')
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({ isSuccess: false, errorDetail: 'read-only' } as never)

    void useTabStore().closeTab(SERVER)
    await flushPromises()
    await clickButton('Save All')

    expect(notifyError).toHaveBeenCalledOnce()
    expect(useQueryStore().getQueryState(notReached).isDirty).toBe(true)
    expect(confirmDisconnect).not.toHaveBeenCalled()
  })
})

describe('disconnecting with no tab open', () => {
  it('still confirms, using the name of the connection', async () => {
    const browserStore = useDataBrowserStore()
    browserStore.connections = [{ serverID: 'server-2', name: 'Other Server' }] as never

    await useTabStore().closeTab('server-2')

    expect(dialogShow).not.toHaveBeenCalled()
    expect(confirmDisconnect.mock.calls[0]![0]).toContain('Other Server')
  })

  it('tells the caller once the disconnect has happened', async () => {
    const onDisconnected = vi.fn()

    await useTabStore().closeTab(SERVER, onDisconnected)
    expect(onDisconnected).not.toHaveBeenCalled()

    await confirmDisconnect.mock.calls[0]![1]()
    expect(onDisconnected).toHaveBeenCalledOnce()
  })
})

describe('resolveUnsavedQueries', () => {
  it('is true straight away when nothing is unsaved', async () => {
    await expect(useTabStore().resolveUnsavedQueries(SERVER)).resolves.toBe(true)
    expect(dialogShow).not.toHaveBeenCalled()
  })

  it('does not ask for a confirmation to disconnect', async () => {
    addDirtyQuery('/queries/a.js')

    const resolved = useTabStore().resolveUnsavedQueries(SERVER)
    await flushPromises()
    await clickButton("Don't Save")

    await expect(resolved).resolves.toBe(true)
    expect(confirmDisconnect).not.toHaveBeenCalled()
  })

  it('is false when the user cancels', async () => {
    addDirtyQuery('/queries/a.js')

    const resolved = useTabStore().resolveUnsavedQueries(SERVER)
    await flushPromises()
    await clickButton('Cancel')

    await expect(resolved).resolves.toBe(false)
  })

  it('only covers the server it was asked about', async () => {
    addSecondServer()
    addDirtyQuery('/queries/a.js', 'server-2')

    await expect(useTabStore().resolveUnsavedQueries(SERVER)).resolves.toBe(true)
    expect(dialogShow).not.toHaveBeenCalled()
  })
})

describe('resolveAllUnsavedQueries', () => {
  it('asks once across every server, saying which server each tab is on', async () => {
    addSecondServer()
    addDirtyQuery('/queries/a.js')
    addDirtyQuery('/queries/b.js', 'server-2')

    const resolved = useTabStore().resolveAllUnsavedQueries()
    await flushPromises()

    expect(dialogShow).toHaveBeenCalledOnce()
    expect(renderToWrapper(promptOptions().content).findAll('li').map((li) => li.text())).toEqual([
      'Server 1: a.js',
      'Server 2: b.js',
    ])

    await clickButton('Save All')
    await expect(resolved).resolves.toBe(true)
    expect(filesProxy.WriteFile).toHaveBeenCalledTimes(2)
  })

  it('leaves out the server name when they are all on one', async () => {
    addSecondServer()
    addDirtyQuery('/queries/a.js')

    void useTabStore().resolveAllUnsavedQueries()
    await flushPromises()

    expect(renderToWrapper(promptOptions().content).text()).toBe('Save changes to a.js?')
  })
})

describe('hasUnsavedQueries', () => {
  it('follows the dirty state of open queries', () => {
    const tabStore = useTabStore()
    expect(tabStore.hasUnsavedQueries).toBe(false)

    const id = addDirtyQuery('/queries/a.js')
    expect(tabStore.hasUnsavedQueries).toBe(true)

    useQueryStore().setDirty(id, false)
    expect(tabStore.hasUnsavedQueries).toBe(false)
  })

  it('ignores queries left behind by a disconnected server', () => {
    const tabStore = useTabStore()
    addDirtyQuery('/queries/a.js')

    tabStore.removeTabById(SERVER)

    expect(tabStore.hasUnsavedQueries).toBe(false)
  })
})
