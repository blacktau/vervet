import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  ExecuteQuery: vi.fn(),
  CancelQuery: vi.fn(),
  FetchPage: vi.fn(),
  CountForPage: vi.fn(),
  CheckMongosh: vi.fn(),
}))

vi.mock('wailsjs/go/api/FilesProxy', () => ({
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

import * as filesProxy from 'wailsjs/go/api/FilesProxy'
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
