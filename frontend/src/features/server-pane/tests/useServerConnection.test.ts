import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const connect = vi.fn()
const disconnect = vi.fn().mockResolvedValue(true)
const upsertTab = vi.fn()

vi.mock('wailsjs/go/api/OIDCProxy', () => ({
  CancelLogin: vi.fn().mockResolvedValue({ isSuccess: true }),
}))

vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: vi.fn(() => ({ connect, disconnect })),
}))

vi.mock('@/features/tabs/tabs.ts', () => ({
  useTabStore: vi.fn(() => ({ upsertTab })),
}))

vi.mock('@/utils/dialog.ts', () => ({
  useMessager: vi.fn(() => ({ error: vi.fn() })),
}))

import * as oidcProxy from 'wailsjs/go/api/OIDCProxy'
import { useServerConnection } from '@/features/server-pane/useServerConnection'

describe('useServerConnection', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test('cancelling cancels the in-flight OIDC login', async () => {
    const { connectingServer, connectToServer, onCancelConnecting } = useServerConnection()
    connect.mockReturnValue(new Promise(() => {}))
    void connectToServer('s1')
    await onCancelConnecting()
    expect(oidcProxy.CancelLogin).toHaveBeenCalledWith('s1')
    expect(connectingServer.value).toBe('')
  })

  test('a cancelled attempt resolving late does not swallow the retry tab', async () => {
    const { connectToServer, onCancelConnecting } = useServerConnection()

    let failFirst: (v: { success: boolean }) => void = () => {}
    connect.mockReturnValueOnce(new Promise((resolve) => (failFirst = resolve)))
    const first = connectToServer('s1')
    await onCancelConnecting()

    let succeedSecond: (v: { success: boolean; serverId: string; name: string }) => void = () => {}
    connect.mockReturnValueOnce(new Promise((resolve) => (succeedSecond = resolve)))
    const second = connectToServer('s1')

    // The cancelled attempt only errors out once the retry is already in flight.
    failFirst({ success: false })
    await first
    succeedSecond({ success: true, serverId: 's1', name: 'S1' })
    await second

    expect(upsertTab).toHaveBeenCalledWith(expect.objectContaining({ serverId: 's1', title: 'S1' }))
  })
})
