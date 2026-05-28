import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('wailsjs/go/api/ServersProxy', () => ({
  CreateGroup: vi.fn(),
  GetServers: vi.fn().mockResolvedValue({ isSuccess: true, data: [] }),
}))

vi.mock('wailsjs/go/api/FilesProxy', () => ({}))

vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: vi.fn(() => ({})),
}))

vi.mock('@/utils/dialog.ts', () => ({
  useNotifier: vi.fn(() => ({ error: vi.fn(), warning: vi.fn() })),
}))

import * as serversProxy from 'wailsjs/go/api/ServersProxy'
import { useServerStore } from '@/features/server-pane/serverStore'

describe('serverStore.createGroup', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('returns the new group id on success', async () => {
    vi.mocked(serversProxy.CreateGroup).mockResolvedValue({
      isSuccess: true,
      data: 'new-id',
    })

    const store = useServerStore()
    const result = await store.createGroup('Test', '')

    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.id).toBe('new-id')
    }
  })

  test('returns failure with msg on error', async () => {
    vi.mocked(serversProxy.CreateGroup).mockResolvedValue({
      isSuccess: false,
      data: '',
      errorCode: 'someError',
    })

    const store = useServerStore()
    const result = await store.createGroup('Test', '')

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.msg).toBeTruthy()
    }
  })
})
