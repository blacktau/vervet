import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('bindings/vervet/internal/api/shellproxy', () => ({
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(),
  CheckMongosh: vi.fn(),
}))

vi.mock('bindings/vervet/internal/api/filesproxy', () => ({
  SelectFile: vi.fn(),
  ReadFile: vi.fn(),
  WriteFile: vi.fn(),
  SaveFile: vi.fn(),
}))

const notifyError = vi.fn()
vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ info: vi.fn(), success: vi.fn(), error: notifyError, warning: vi.fn() }),
  useDialoger: () => ({}),
  useMessager: () => ({}),
}))

import * as filesProxy from 'bindings/vervet/internal/api/filesproxy'
import { useQueryStore } from '@/features/queries/queryStore'

const QUERY_ID = 'q-1'
const dialogFailure = { isSuccess: false, data: '', errorCode: 'unknown_error', errorDetail: 'no display' }
const cancelled = { isSuccess: true, data: '' }

function storeWithQuery() {
  const store = useQueryStore()
  store.initQueryState(QUERY_ID, 'mydb')
  return store
}

describe('queryStore file reads and writes', () => {
  const ioFailure = { isSuccess: false, errorCode: 'unknown_error', errorDetail: 'permission denied' }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('openFile tells the user when the file cannot be read and keeps the current query', async () => {
    vi.mocked(filesProxy.SelectFile).mockResolvedValue({ isSuccess: true, data: '/q.js' } as never)
    vi.mocked(filesProxy.ReadFile).mockResolvedValue({ ...ioFailure, data: '' } as never)
    const store = storeWithQuery()

    expect(await store.openFile(QUERY_ID)).toBeNull()

    expect(notifyError.mock.calls[0]![0]).toBe('permission denied')
    expect(store.getQueryState(QUERY_ID).filePath).not.toBe('/q.js')
  })

  test('leads with the real error when Go classifies it as a database failure', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue({
      isSuccess: false,
      errorCode: 'query_timeout',
      errorDetail: 'failed to write file: open /mnt/q.js: connection timed out',
    } as never)
    const store = storeWithQuery()
    store.setFilePath(QUERY_ID, '/mnt/q.js')

    await store.saveFile(QUERY_ID, 'db.x.find()')

    expect(notifyError.mock.calls[0]![0]).toBe('failed to write file: open /mnt/q.js: connection timed out')
  })

  test('saveFile tells the user when the write fails and keeps the query dirty', async () => {
    vi.mocked(filesProxy.WriteFile).mockResolvedValue(ioFailure as never)
    const store = storeWithQuery()
    store.setFilePath(QUERY_ID, '/q.js')
    store.setDirty(QUERY_ID, true)

    expect(await store.saveFile(QUERY_ID, 'db.x.find()')).toBe(false)

    expect(notifyError.mock.calls[0]![0]).toBe('permission denied')
    expect(store.getQueryState(QUERY_ID).isDirty).toBe(true)
  })

  test('saveFileAs tells the user when the write fails and keeps the old path', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({ isSuccess: true, data: '/new.js' } as never)
    vi.mocked(filesProxy.WriteFile).mockResolvedValue(ioFailure as never)
    const store = storeWithQuery()

    expect(await store.saveFileAs(QUERY_ID, 'db.x.find()')).toBe(false)

    expect(notifyError.mock.calls[0]![0]).toBe('permission denied')
    expect(store.getQueryState(QUERY_ID).filePath).not.toBe('/new.js')
  })
})

// #363: text typed while a slow write runs must stay unsaved.
describe('queryStore save while the user keeps typing', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  function slowWrite() {
    let finish!: () => void
    vi.mocked(filesProxy.WriteFile).mockReturnValueOnce(
      new Promise((resolve) => {
        finish = () => resolve({ isSuccess: true } as never)
      }) as never,
    )
    return () => finish()
  }

  test('saveFile keeps the query dirty when it changed during the write', async () => {
    const finish = slowWrite()
    const store = storeWithQuery()
    store.setFilePath(QUERY_ID, '/mnt/q.js')
    store.setCurrentContent(QUERY_ID, 'db.x.find()')

    const saving = store.saveFile(QUERY_ID, 'db.x.find()')
    store.setCurrentContent(QUERY_ID, 'db.x.find({ a: 1 })')
    finish()

    expect(await saving).toBe(true)
    const state = store.getQueryState(QUERY_ID)
    expect(state.savedContent).toBe('db.x.find()')
    expect(state.isDirty).toBe(true)
  })

  test('saveFileAs keeps the query dirty when it changed during the write', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue({
      isSuccess: true,
      data: '/mnt/new.js',
    } as never)
    const finish = slowWrite()
    const store = storeWithQuery()
    store.setCurrentContent(QUERY_ID, 'db.x.find()')

    const saving = store.saveFileAs(QUERY_ID, 'db.x.find()')
    await vi.waitFor(() => expect(filesProxy.WriteFile).toHaveBeenCalled())
    store.setCurrentContent(QUERY_ID, 'db.x.find({ a: 1 })')
    finish()

    expect(await saving).toBe(true)
    const state = store.getQueryState(QUERY_ID)
    expect(state.filePath).toBe('/mnt/new.js')
    expect(state.savedContent).toBe('db.x.find()')
    expect(state.isDirty).toBe(true)
  })

  test('does not bring back the state of a tab closed during the write', async () => {
    const finish = slowWrite()
    const store = storeWithQuery()
    store.setFilePath(QUERY_ID, '/mnt/q.js')

    const saving = store.saveFile(QUERY_ID, 'db.x.find()')
    store.removeQueryState(QUERY_ID)
    finish()

    expect(await saving).toBe(true)
    expect(store.queries[QUERY_ID]).toBeUndefined()
  })

  test('saveFile marks the query clean when nothing changed during the write', async () => {
    const finish = slowWrite()
    const store = storeWithQuery()
    store.setFilePath(QUERY_ID, '/mnt/q.js')
    store.setCurrentContent(QUERY_ID, 'db.x.find()')
    store.setDirty(QUERY_ID, true)

    const saving = store.saveFile(QUERY_ID, 'db.x.find()')
    finish()

    expect(await saving).toBe(true)
    expect(store.getQueryState(QUERY_ID).isDirty).toBe(false)
  })
})

describe('queryStore file dialogs', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('openFile tells the user when the dialog fails', async () => {
    vi.mocked(filesProxy.SelectFile).mockResolvedValue(dialogFailure as never)

    expect(await storeWithQuery().openFile(QUERY_ID)).toBeNull()

    expect(notifyError).toHaveBeenCalledOnce()
    expect(notifyError.mock.calls[0]![0]).toBe('no display')
    expect(filesProxy.ReadFile).not.toHaveBeenCalled()
  })

  test('openFile stays silent when the dialog is cancelled', async () => {
    vi.mocked(filesProxy.SelectFile).mockResolvedValue(cancelled as never)

    expect(await storeWithQuery().openFile(QUERY_ID)).toBeNull()

    expect(notifyError).not.toHaveBeenCalled()
    expect(filesProxy.ReadFile).not.toHaveBeenCalled()
  })

  test('saveFileAs tells the user when the dialog fails', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue(dialogFailure as never)

    expect(await storeWithQuery().saveFileAs(QUERY_ID, 'db.x.find()')).toBe(false)

    expect(notifyError).toHaveBeenCalledOnce()
    expect(notifyError.mock.calls[0]![0]).toBe('no display')
    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
  })

  test('saveFileAs stays silent when the dialog is cancelled', async () => {
    vi.mocked(filesProxy.SaveFile).mockResolvedValue(cancelled as never)

    expect(await storeWithQuery().saveFileAs(QUERY_ID, 'db.x.find()')).toBe(false)

    expect(notifyError).not.toHaveBeenCalled()
    expect(filesProxy.WriteFile).not.toHaveBeenCalled()
  })
})

describe('queryStore.loadFileByPath', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('leads with the real error when the file cannot be read', async () => {
    vi.mocked(filesProxy.ReadFile).mockResolvedValue({
      isSuccess: false,
      data: '',
      errorCode: 'unknown_error',
      errorDetail: 'permission denied',
    } as never)

    expect(await storeWithQuery().loadFileByPath(QUERY_ID, '/q.js')).toBe(false)

    expect(notifyError.mock.calls[0]![0]).toBe('permission denied')
  })
})
