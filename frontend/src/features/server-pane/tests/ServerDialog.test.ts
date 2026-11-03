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

const disconnect = vi.fn()
vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: () => ({ isConnected: () => false, disconnect }),
}))

const notifyError = vi.fn()
const notifyWarning = vi.fn()
vi.mock('@/utils/dialog.ts', () => ({
  useNotifier: () => ({ error: notifyError, warning: notifyWarning }),
  useMessager: () => ({ error: vi.fn(), success: vi.fn() }),
}))

const metadata = { id: 's1', name: 'Server', colour: '#F75B52', parentID: 'g1' }
const loaded = {
  status: 'loaded',
  server: {
    ...metadata,
    uri: 'mongodb://host',
    authMethod: 'oidc',
    oidcConfig: { providerUrl: 'https://idp', clientId: 'vervet', scopes: [] },
  },
}
const failed = { status: 'failed', error: { errorCode: 'unknown_error', errorDetail: 'keyring locked' } }

type DialogVm = {
  generalForm: { name: string; connectionString: string; parentId: string; colour: string }
  authPicker: string
}

function deferred() {
  let resolve: (value: unknown) => void = () => {}
  const promise = new Promise((r) => {
    resolve = r
  })
  return { promise, resolve }
}

const modalStub = { template: '<div><slot /><slot name="action" /></div>' }
const buttonStub = { props: ['disabled'], template: '<button :disabled="disabled"><slot /></button>' }

function mountDialog() {
  const wrapper = shallowMount(ServerDialog, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })],
      mocks: { $t: (key: string) => key },
      // Naive UI's own component names, without the N prefix.
      stubs: { Modal: modalStub, Button: buttonStub },
    },
  })
  const buttons = () => wrapper.findAll('button')
  return {
    wrapper,
    vm: wrapper.vm as unknown as DialogVm,
    testButton: () => buttons()[0]!,
    cancelButton: () => buttons()[1]!,
    saveButton: () => buttons()[2]!,
  }
}

describe('ServerDialog loading an existing server', () => {
  let dialogStore: ReturnType<typeof useDialogStore>
  const isOpen = () => dialogStore.dialogs[DialogType.Server].visible

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    getServerDetails.mockReset()
    dialogStore = useDialogStore()
  })

  test.each(['edit', 'clone'])('shows the error and closes in %s mode when the load fails', async (mode) => {
    getServerDetails.mockResolvedValue(failed)
    mountDialog()

    if (mode === 'edit') {
      await dialogStore.showServerEditDialog('s1')
    } else {
      await dialogStore.showCloneServerDialog('s1')
    }
    await flushPromises()

    expect(notifyError).toHaveBeenCalledTimes(1)
    expect(notifyError.mock.calls[0]?.[1]).toMatchObject({ detail: 'keyring locked' })
    expect(isOpen()).toBe(false)
  })

  test('fills the form and stays open when the load succeeds', async () => {
    getServerDetails.mockResolvedValue(loaded)
    const { vm } = mountDialog()

    await dialogStore.showServerEditDialog('s1')
    await flushPromises()

    expect(isOpen()).toBe(true)
    expect(vm.generalForm.connectionString).toBe('mongodb://host')
    expect(vm.authPicker).toBe('oidc')
  })

  test('cannot save or test until the load settles, but can cancel', async () => {
    const load = deferred()
    getServerDetails.mockReturnValue(load.promise)
    const { wrapper, vm, testButton, cancelButton, saveButton } = mountDialog()

    await dialogStore.showServerEditDialog('s1')
    await flushPromises()
    vm.generalForm.connectionString = 'mongodb://pasted'
    await wrapper.vm.$nextTick()

    expect(vm.generalForm.name).toBe('pasted')
    expect(saveButton().attributes('disabled')).toBeDefined()
    expect(testButton().attributes('disabled')).toBeDefined()
    expect(cancelButton().attributes('disabled')).toBeUndefined()

    load.resolve(loaded)
    await flushPromises()

    expect(saveButton().attributes('disabled')).toBeUndefined()
    expect(testButton().attributes('disabled')).toBeUndefined()
  })

  test('keeps the metadata when the keyring entry is missing, so the server can be repaired', async () => {
    getServerDetails.mockResolvedValue({ status: 'configMissing', server: metadata })
    const { wrapper, vm, saveButton } = mountDialog()

    await dialogStore.showServerEditDialog('s1')
    await flushPromises()

    expect(isOpen()).toBe(true)
    expect(notifyWarning).toHaveBeenCalledTimes(1)
    expect(notifyError).not.toHaveBeenCalled()
    expect(vm.generalForm).toMatchObject({ name: 'Server', colour: '#F75B52', parentId: 'g1', connectionString: '' })
    expect(saveButton().attributes('disabled')).toBeDefined()

    vm.generalForm.connectionString = 'mongodb://new-host'
    await wrapper.vm.$nextTick()

    expect(vm.generalForm.name).toBe('Server')
    expect(saveButton().attributes('disabled')).toBeUndefined()
  })

  describe('a load from an earlier open', () => {
    async function reopenWhileLoading(secondLoad: unknown) {
      const first = deferred()
      getServerDetails.mockReturnValueOnce(first.promise)
      getServerDetails.mockReturnValueOnce(secondLoad)
      const dialog = mountDialog()

      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      dialogStore.hide(DialogType.Server)
      await flushPromises()
      await dialogStore.showServerEditDialog('s1')
      await flushPromises()

      return { ...dialog, finishFirstLoad: first.resolve }
    }

    test('does not close or report on the reopened dialog when it fails', async () => {
      const { finishFirstLoad } = await reopenWhileLoading(Promise.resolve(loaded))

      finishFirstLoad(failed)
      await flushPromises()

      expect(isOpen()).toBe(true)
      expect(notifyError).not.toHaveBeenCalled()
    })

    test('does not overwrite the reopened form when it succeeds', async () => {
      const { vm, finishFirstLoad } = await reopenWhileLoading(Promise.resolve(loaded))
      vm.generalForm.name = 'Typed by the user'

      finishFirstLoad(loaded)
      await flushPromises()

      expect(vm.generalForm.name).toBe('Typed by the user')
    })

    test('does not re-enable saving while the newer load is still running', async () => {
      const { vm, wrapper, saveButton, finishFirstLoad } = await reopenWhileLoading(deferred().promise)
      vm.generalForm.connectionString = 'mongodb://pasted'
      await wrapper.vm.$nextTick()

      finishFirstLoad(loaded)
      await flushPromises()

      expect(saveButton().attributes('disabled')).toBeDefined()
    })
  })

  test('says nothing when a cancelled load later fails', async () => {
    const load = deferred()
    getServerDetails.mockReturnValue(load.promise)
    mountDialog()

    await dialogStore.showServerEditDialog('s1')
    await flushPromises()
    dialogStore.hide(DialogType.Server)
    await flushPromises()

    load.resolve(failed)
    await flushPromises()

    expect(notifyError).not.toHaveBeenCalled()
  })

  describe('editing a connected server', () => {
    test('disconnects only once the load has succeeded', async () => {
      const load = deferred()
      getServerDetails.mockReturnValue(load.promise)
      mountDialog()

      await dialogStore.showServerEditDialog('s1', { disconnectOnLoad: true })
      await flushPromises()
      expect(disconnect).not.toHaveBeenCalled()

      load.resolve(loaded)
      await flushPromises()

      expect(disconnect).toHaveBeenCalledWith('s1')
    })

    test('stays connected when the load fails', async () => {
      getServerDetails.mockResolvedValue(failed)
      mountDialog()

      await dialogStore.showServerEditDialog('s1', { disconnectOnLoad: true })
      await flushPromises()

      expect(disconnect).not.toHaveBeenCalled()
    })

    test('stays connected when cancelled during the load', async () => {
      const load = deferred()
      getServerDetails.mockReturnValue(load.promise)
      mountDialog()

      await dialogStore.showServerEditDialog('s1', { disconnectOnLoad: true })
      await flushPromises()
      dialogStore.hide(DialogType.Server)
      await flushPromises()

      load.resolve(loaded)
      await flushPromises()

      expect(disconnect).not.toHaveBeenCalled()
    })
  })
})
