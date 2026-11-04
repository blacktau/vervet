// @vitest-environment happy-dom
import { flushPromises, mount, shallowMount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ServerDialog from '@/features/server-pane/ServerDialog.vue'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({ TestConnection: vi.fn() }))

const getServerDetails = vi.fn()
const updateServerWithConfig = vi.fn()
const saveServerWithConfig = vi.fn()
vi.mock('@/features/server-pane/serverStore.ts', () => ({
  useServerStore: () => ({ getServerDetails, updateServerWithConfig, saveServerWithConfig, serverTree: [] }),
}))

const disconnect = vi.fn()
const isConnected = vi.fn()
vi.mock('@/features/data-browser/browserStore.ts', () => ({
  useDataBrowserStore: () => ({ isConnected, disconnect }),
}))

const notifyError = vi.fn()
const notifyWarning = vi.fn()
const confirmWarning = vi.fn()
vi.mock('@/utils/dialog.ts', () => ({
  useNotifier: () => ({ error: notifyError, warning: notifyWarning }),
  useMessager: () => ({ error: vi.fn(), success: vi.fn() }),
  useDialoger: () => ({ warning: confirmWarning }),
}))

const resolveUnsavedQueries = vi.fn(async () => true)
vi.mock('@/features/tabs/tabs.ts', () => ({
  useTabStore: () => ({ resolveUnsavedQueries }),
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
const spinStub = { template: '<div><slot /></div>' }
const formStub = { template: '<div><slot /></div>', methods: { validate: () => Promise.resolve() } }
const globalOptions = () => ({
  plugins: [createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })],
  mocks: { $t: (key: string) => key },
  // Naive UI's own component names, without the N prefix.
  stubs: { Modal: modalStub, Button: buttonStub, Spin: spinStub, Form: formStub, Alert: spinStub },
})

function mountDialog() {
  const wrapper = shallowMount(ServerDialog, {
    global: globalOptions(),
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
    disconnect.mockReset().mockResolvedValue(true)
    isConnected.mockReset().mockReturnValue(false)
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
    expect(wrapper.text()).toContain('serverPane.dialogs.server.configMissing')
    expect(notifyError).not.toHaveBeenCalled()
    expect(vm.generalForm).toMatchObject({ name: 'Server', colour: '#F75B52', parentId: 'g1', connectionString: '' })
    expect(saveButton().attributes('disabled')).toBeDefined()

    vm.generalForm.connectionString = 'mongodb://new-host'
    await wrapper.vm.$nextTick()

    expect(vm.generalForm.name).toBe('Server')
    expect(saveButton().attributes('disabled')).toBeUndefined()
  })

  describe('saving', () => {
    const savedUri = () => updateServerWithConfig.mock.calls[0]?.[4]?.uri

    beforeEach(() => {
      updateServerWithConfig.mockReset().mockResolvedValue({ success: true })
    })

    test('a normally loaded server saves without asking', async () => {
      getServerDetails.mockResolvedValue(loaded)
      const { saveButton } = mountDialog()
      await dialogStore.showServerEditDialog('s1')
      await flushPromises()

      await saveButton().trigger('click')
      await flushPromises()

      expect(confirmWarning).not.toHaveBeenCalled()
      expect(savedUri()).toBe('mongodb://host')
    })

    test('a server with a missing keyring entry only saves once confirmed', async () => {
      getServerDetails.mockResolvedValue({ status: 'configMissing', server: metadata })
      const { wrapper, vm, saveButton } = mountDialog()
      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      vm.generalForm.connectionString = 'mongodb://new-host'
      await wrapper.vm.$nextTick()

      await saveButton().trigger('click')
      await flushPromises()

      expect(confirmWarning).toHaveBeenCalledTimes(1)
      expect(updateServerWithConfig).not.toHaveBeenCalled()
      expect(isOpen()).toBe(true)

      await confirmWarning.mock.calls[0]?.[1]()
      await flushPromises()

      expect(savedUri()).toBe('mongodb://new-host')
    })
  })

  test('a clone of a server with a missing keyring entry closes, as there is nothing to copy', async () => {
    getServerDetails.mockResolvedValue({ status: 'configMissing', server: metadata })
    mountDialog()

    await dialogStore.showCloneServerDialog('s1')
    await flushPromises()

    expect(isOpen()).toBe(false)
    expect(notifyError).toHaveBeenCalledTimes(1)
  })

  test('blocks the keyboard as well as the mouse until the load settles', async () => {
    const load = deferred()
    getServerDetails.mockReturnValue(load.promise)
    const { wrapper } = mountDialog()

    await dialogStore.showServerEditDialog('s1')
    await flushPromises()
    expect(wrapper.find('[inert]').exists()).toBe(true)

    load.resolve(loaded)
    await flushPromises()
    expect(wrapper.find('[inert]').exists()).toBe(false)
  })

  describe('mounted with v-if, as App.vue does', () => {
    function mountLikeApp() {
      const App = defineComponent({
        components: { ServerDialog },
        setup: () => ({ dialogStore, DialogType: 'server' }),
        template: '<ServerDialog v-if="dialogStore.isVisible(DialogType)" />',
      })
      return mount(App, { global: globalOptions() })
    }

    test('a cancelled load leaves the server connected', async () => {
      isConnected.mockReturnValue(true)
      const load = deferred()
      getServerDetails.mockReturnValue(load.promise)
      mountLikeApp()

      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      dialogStore.hide(DialogType.Server)
      await flushPromises()

      load.resolve(loaded)
      await flushPromises()

      expect(disconnect).not.toHaveBeenCalled()
    })

    test('a cancelled load that fails shows nothing and leaves the next dialog open', async () => {
      const load = deferred()
      getServerDetails.mockReturnValueOnce(load.promise)
      mountLikeApp()

      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      dialogStore.hide(DialogType.Server)
      await flushPromises()
      dialogStore.showNewServerDialog()
      await flushPromises()

      load.resolve(failed)
      await flushPromises()

      expect(notifyError).not.toHaveBeenCalled()
      expect(isOpen()).toBe(true)
    })
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
    beforeEach(() => {
      isConnected.mockReturnValue(true)
      updateServerWithConfig.mockReset().mockResolvedValue({ success: true })
    })

    async function openLoaded() {
      getServerDetails.mockResolvedValue(loaded)
      const dialog = mountDialog()
      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      return dialog
    }

    test('stays connected when cancelled, during the load or after it', async () => {
      const load = deferred()
      getServerDetails.mockReturnValue(load.promise)
      const { cancelButton } = mountDialog()

      await dialogStore.showServerEditDialog('s1')
      await flushPromises()
      expect(cancelButton().attributes('disabled')).toBeUndefined()

      load.resolve(loaded)
      await flushPromises()
      await cancelButton().trigger('click')
      await flushPromises()

      expect(isOpen()).toBe(false)
      expect(disconnect).not.toHaveBeenCalled()
    })

    test.each([
      ['fails', failed],
      ['finds no saved details', { status: 'configMissing', server: metadata }],
    ])('stays connected when the load %s', async (_name, result) => {
      getServerDetails.mockResolvedValue(result)
      mountDialog()

      await dialogStore.showServerEditDialog('s1')
      await flushPromises()

      expect(disconnect).not.toHaveBeenCalled()
    })

    test('warns before saving, then saves and disconnects in that order', async () => {
      const { saveButton } = await openLoaded()

      expect(saveButton().attributes('disabled')).toBeUndefined()
      await saveButton().trigger('click')
      await flushPromises()

      expect(confirmWarning.mock.calls[0]?.[0]).toContain('saveClosesConnection')
      expect(updateServerWithConfig).not.toHaveBeenCalled()
      expect(disconnect).not.toHaveBeenCalled()

      await confirmWarning.mock.calls[0]?.[1]()
      await flushPromises()

      expect(disconnect).toHaveBeenCalledWith('s1')
      expect(updateServerWithConfig.mock.invocationCallOrder[0]).toBeLessThan(disconnect.mock.invocationCallOrder[0]!)
      expect(isOpen()).toBe(false)
    })

    test('stays connected and open when the save fails', async () => {
      updateServerWithConfig.mockResolvedValue({ success: false, msg: 'keyring locked' })
      const { saveButton } = await openLoaded()

      await saveButton().trigger('click')
      await flushPromises()
      await confirmWarning.mock.calls[0]?.[1]()
      await flushPromises()

      expect(disconnect).not.toHaveBeenCalled()
      expect(isOpen()).toBe(true)
    })

    test('cancelling the unsaved queries prompt leaves the details and the connection alone', async () => {
      resolveUnsavedQueries.mockResolvedValueOnce(false)
      const { saveButton } = await openLoaded()

      await saveButton().trigger('click')
      await flushPromises()
      await confirmWarning.mock.calls[0]?.[1]()
      await flushPromises()

      expect(resolveUnsavedQueries).toHaveBeenCalledWith('s1')
      expect(updateServerWithConfig).not.toHaveBeenCalled()
      expect(disconnect).not.toHaveBeenCalled()
      expect(isOpen()).toBe(true)
    })

    test('reports a rejected disconnect after saving, without getting stuck', async () => {
      disconnect.mockRejectedValue(new Error('bridge down'))
      const { saveButton } = await openLoaded()

      await saveButton().trigger('click')
      await flushPromises()
      await confirmWarning.mock.calls[0]?.[1]()
      await flushPromises()

      expect(isOpen()).toBe(false)
      expect(notifyWarning.mock.calls[0]?.[1]).toMatchObject({ detail: 'bridge down' })
    })

    test('a clone does not disconnect the original', async () => {
      getServerDetails.mockResolvedValue(loaded)
      saveServerWithConfig.mockResolvedValue({ success: true })
      const { saveButton } = mountDialog()
      await dialogStore.showCloneServerDialog('s1')
      await flushPromises()

      await saveButton().trigger('click')
      await flushPromises()

      expect(confirmWarning).not.toHaveBeenCalled()
      expect(saveServerWithConfig).toHaveBeenCalledTimes(1)
      expect(disconnect).not.toHaveBeenCalled()
    })
  })
})
