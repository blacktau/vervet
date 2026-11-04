// @vitest-environment happy-dom
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createI18n } from 'vue-i18n'

vi.mock('wailsjs/go/api/ServersProxy', () => ({}))

const resetSession = vi.fn(async () => ({ isSuccess: true }))
vi.mock('wailsjs/go/api/OIDCProxy', () => ({
  ResetSession: () => resetSession(),
}))

const closeTab = vi.fn(async () => undefined)
const resolveUnsavedQueries = vi.fn(async () => true)
vi.mock('@/features/tabs/tabs.ts', () => ({
  useTabStore: () => ({ closeTab, resolveUnsavedQueries }),
}))

const disconnect = vi.fn(async () => true)
const isConnected = vi.fn(() => true)
vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: () => ({ disconnect, isConnected }),
}))

const deleteServer = vi.fn(async () => ({ success: true }))
vi.mock('@/features/server-pane/serverStore.ts', () => ({
  useServerStore: () => ({
    deleteServer,
    findServerById: () => ({ id: 's1', name: 'Server' }),
  }),
}))

vi.mock('@/stores/dialog.ts', () => ({
  DialogType: {},
  useDialogStore: () => ({}),
}))

const messagerSuccess = vi.fn()
// The confirmation everything here starts with: says yes straight away.
const confirm = vi.fn((_content: string, onConfirm?: () => unknown) => onConfirm?.())
vi.mock('@/utils/dialog.ts', () => ({
  useDialoger: () => ({ warning: confirm }),
  useMessager: () => ({ success: messagerSuccess, error: vi.fn() }),
}))

import { MenuKeys, useServerTreeContextMenu } from '@/features/server-pane/useServerTreeContextMenu'

function selectMenuItem(key: string) {
  let handle: (key: string) => Promise<void> = async () => {}
  mount(
    defineComponent({
      setup() {
        handle = useServerTreeContextMenu(ref(['s1']), vi.fn()).handleSelectContextMenu
        return () => null
      },
    }),
    { global: { plugins: [createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })] } },
  )
  return handle(key).then(flushPromises)
}

describe('useServerTreeContextMenu disconnecting', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    isConnected.mockReturnValue(true)
    resolveUnsavedQueries.mockResolvedValue(true)
  })

  test('Disconnect goes through the normal flow, and reports success afterwards', async () => {
    await selectMenuItem(MenuKeys.ServerDisconnect)

    expect(closeTab).toHaveBeenCalledWith('s1', expect.any(Function))
    expect(disconnect).not.toHaveBeenCalled()
    expect(messagerSuccess).not.toHaveBeenCalled()

    const onDisconnected = (closeTab.mock.calls[0] as unknown as [string, () => void])[1]
    onDisconnected()
    expect(messagerSuccess).toHaveBeenCalledOnce()
  })

  test('Delete asks about unsaved queries before deleting', async () => {
    await selectMenuItem(MenuKeys.ServerDelete)

    expect(resolveUnsavedQueries).toHaveBeenCalledWith('s1')
    expect(deleteServer).toHaveBeenCalledWith('s1')
    expect(resolveUnsavedQueries.mock.invocationCallOrder[0]).toBeLessThan(deleteServer.mock.invocationCallOrder[0]!)
  })

  test('cancelling the prompt keeps the server', async () => {
    resolveUnsavedQueries.mockResolvedValue(false)

    await selectMenuItem(MenuKeys.ServerDelete)

    expect(deleteServer).not.toHaveBeenCalled()
  })

  test('Reset OIDC session asks about unsaved queries before disconnecting', async () => {
    await selectMenuItem(MenuKeys.ServerResetOIDC)

    expect(resolveUnsavedQueries).toHaveBeenCalledWith('s1')
    expect(disconnect).toHaveBeenCalledWith('s1')
    expect(resetSession).toHaveBeenCalledOnce()
  })

  test('cancelling the prompt stops the OIDC reset', async () => {
    resolveUnsavedQueries.mockResolvedValue(false)

    await selectMenuItem(MenuKeys.ServerResetOIDC)

    expect(disconnect).not.toHaveBeenCalled()
    expect(resetSession).not.toHaveBeenCalled()
  })

  test('Reset OIDC session has nothing to ask about when not connected', async () => {
    isConnected.mockReturnValue(false)

    await selectMenuItem(MenuKeys.ServerResetOIDC)

    expect(resolveUnsavedQueries).not.toHaveBeenCalled()
    expect(resetSession).toHaveBeenCalledOnce()
  })
})
