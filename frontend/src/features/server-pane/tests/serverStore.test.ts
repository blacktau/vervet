import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('wailsjs/go/api/ServersProxy', () => ({
  CreateGroup: vi.fn(),
  GetServers: vi.fn().mockResolvedValue({ isSuccess: true, data: [] }),
  GetServer: vi.fn(),
  GetConnectionConfig: vi.fn(),
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

describe('serverStore.getServerDetails', () => {
  const server = { id: 's1', name: 'Server', colour: '', isGroup: false, isCluster: false, isSrv: false }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(serversProxy.GetServer).mockResolvedValue({ isSuccess: true, data: server } as never)
  })

  test('returns the stored auth method and OIDC settings', async () => {
    const oidcConfig = { providerUrl: 'https://idp', clientId: 'vervet', scopes: [] }
    vi.mocked(serversProxy.GetConnectionConfig).mockResolvedValue({
      isSuccess: true,
      data: { uri: 'mongodb://host', authMethod: 'oidc', oidcConfig },
    } as never)

    const result = await useServerStore().getServerDetails('s1')

    expect(result).toMatchObject({
      status: 'loaded',
      server: { uri: 'mongodb://host', authMethod: 'oidc', oidcConfig },
    })
  })

  test('reports a failed config load rather than assuming password auth', async () => {
    const failure = { isSuccess: false, errorCode: 'unknown_error', errorDetail: 'keyring locked' }
    vi.mocked(serversProxy.GetConnectionConfig).mockResolvedValue(failure as never)

    const result = await useServerStore().getServerDetails('s1')

    expect(result).toEqual({ status: 'failed', error: failure })
  })

  test('tells a missing keyring entry apart, keeping the server metadata', async () => {
    vi.mocked(serversProxy.GetConnectionConfig).mockResolvedValue({
      isSuccess: false,
      errorCode: 'connection_config_missing',
    } as never)

    const result = await useServerStore().getServerDetails('s1')

    expect(result).toEqual({ status: 'configMissing', server })
  })

  test('reports a rejected bridge call as a failure', async () => {
    vi.mocked(serversProxy.GetConnectionConfig).mockRejectedValue(new Error('bridge down'))

    const result = await useServerStore().getServerDetails('s1')

    expect(result).toEqual({ status: 'failed', error: { errorDetail: 'bridge down' } })
  })
})
