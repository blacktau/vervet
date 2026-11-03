// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ServerDialog from '@/features/server-pane/ServerDialog.vue'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({ TestConnection: vi.fn() }))

const getServerDetails = vi.fn()
vi.mock('@/features/server-pane/serverStore.ts', () => ({
  useServerStore: () => ({ getServerDetails, serverTree: [] }),
}))

vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: () => ({ isConnected: () => false }),
}))

const serverDetails = {
  id: 's1',
  name: 'Server',
  colour: '',
  parentID: '',
  uri: 'mongodb://host',
  authMethod: 'oidc',
  oidcConfig: { providerUrl: 'https://idp', clientId: 'vervet', scopes: [] },
}

async function openDialog(mode: 'edit' | 'clone') {
  const dialogStore = useDialogStore()
  shallowMount(ServerDialog, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })],
      mocks: { $t: (key: string) => key },
    },
  })
  if (mode === 'edit') {
    await dialogStore.showServerEditDialog('s1')
    return dialogStore
  }
  await dialogStore.showCloneServerDialog('s1')
  return dialogStore
}

describe('ServerDialog loading an existing server', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  test.each(['edit', 'clone'] as const)('closes in %s mode when the details fail to load', async (mode) => {
    getServerDetails.mockResolvedValue(undefined)

    const dialogStore = await openDialog(mode)
    await flushPromises()

    expect(getServerDetails).toHaveBeenCalledWith('s1')
    expect(dialogStore.dialogs[DialogType.Server].visible).toBe(false)
  })

  test('stays open when the details load', async () => {
    getServerDetails.mockResolvedValue(serverDetails)

    const dialogStore = await openDialog('edit')
    await flushPromises()

    expect(dialogStore.dialogs[DialogType.Server].visible).toBe(true)
  })

  test('a late failure does not close a dialog since opened for another server', async () => {
    let finishLoad: (details: undefined) => void = () => {}
    getServerDetails.mockReturnValueOnce(
      new Promise((resolve) => {
        finishLoad = resolve
      }),
    )
    getServerDetails.mockResolvedValueOnce({ ...serverDetails, id: 's2' })

    const dialogStore = await openDialog('edit')
    await flushPromises()

    dialogStore.hide(DialogType.Server)
    await flushPromises()
    await dialogStore.showServerEditDialog('s2')
    await flushPromises()

    finishLoad(undefined)
    await flushPromises()

    expect(dialogStore.dialogs[DialogType.Server].visible).toBe(true)
  })
})
