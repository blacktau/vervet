// @vitest-environment happy-dom
import { nextTick } from 'vue'
import { describe, expect, test, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive from 'naive-ui'
import OnboardingPanel from '@/features/onboarding/OnboardingPanel.vue'
import * as connectionsProxy from 'wailsjs/go/api/ConnectionsProxy'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({
  TestConnection: vi.fn(),
}))

interface MockNode {
  id: string
  name: string
  isGroup: boolean
  parentID?: string
  colour: string
  isCluster: boolean
  isSrv: boolean
  children?: MockNode[]
}
const storeState: { tree: MockNode[] } = { tree: [] }
const saveServerWithConfig = vi.fn()
const refreshServers = vi.fn()
vi.mock('@/features/server-pane/serverStore.ts', () => ({
  useServerStore: () => ({
    saveServerWithConfig,
    refreshServers,
    get serverTree() {
      return storeState.tree
    },
    findServerById: () => undefined,
  }),
}))

const connectToServer = vi.fn()
vi.mock('@/features/server-pane/useServerConnection.ts', () => ({
  useServerConnection: () => ({ connectToServer }),
}))

function makeWrapper() {
  const i18n = createI18n({
    legacy: false,
    locale: 'en-GB',
    messages: {
      'en-GB': {
        onboarding: {
          welcomeTitle: 'Welcome to Vervet',
          welcomeSubtitle: 'sub',
          uriLabel: 'Connection string',
          uriPlaceholder: 'mongodb://...',
          nameLabel: 'Name',
          namePlaceholder: 'My Server',
          connect: 'Connect',
          advanced: 'Advanced',
          errorTitle: 'Could not connect',
        },
        errors: { CONN_FAIL: 'Connection failed', saveFailed: 'Could not save server' },
        uriParser: { invalidScheme: 'bad scheme', emptyUri: 'empty' },
      },
    },
  })
  return mount(OnboardingPanel, { global: { plugins: [i18n, naive] } })
}

describe('OnboardingPanel', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('renders welcome heading', () => {
    const w = makeWrapper()
    expect(w.text()).toContain('Welcome to Vervet')
  })

  test('Connect button is disabled when URI is empty', () => {
    const w = makeWrapper()
    const button = w.find('[data-test="connect-btn"]')
    expect(button.attributes('disabled')).toBeDefined()
  })

  test('Connect button is disabled when URI is invalid', async () => {
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('not-a-uri')
    const button = w.find('[data-test="connect-btn"]')
    expect(button.attributes('disabled')).toBeDefined()
  })

  test('Connect button is enabled with valid URI', async () => {
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('mongodb://localhost:27017')
    const button = w.find('[data-test="connect-btn"]')
    expect(button.attributes('disabled')).toBeUndefined()
  })
})

describe('OnboardingPanel auto-name', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('name auto-fills from URI host', async () => {
    const w = makeWrapper()
    const uriField = w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea')
    await uriField.setValue('mongodb://example.com:27017')
    await nextTick()
    const nameField = w.find('[data-test="name-input"] input')
    expect((nameField.element as HTMLInputElement).value).toBe('example.com:27017')
  })

  test('auto-fill stops after user edits name', async () => {
    const w = makeWrapper()
    const uriField = w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea')
    const nameField = w.find('[data-test="name-input"] input')
    await uriField.setValue('mongodb://first.example.com:27017')
    await nextTick()
    await nameField.setValue('My Custom Name')
    await uriField.setValue('mongodb://second.example.com:27017')
    await nextTick()
    expect((nameField.element as HTMLInputElement).value).toBe('My Custom Name')
  })
})

describe('OnboardingPanel connect flow', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    storeState.tree = []
    saveServerWithConfig.mockImplementation(async (name: string) => {
      storeState.tree = [
        {
          id: 'new-server-id',
          name,
          isGroup: false,
          colour: '',
          isCluster: false,
          isSrv: false,
        },
      ]
      return { success: true }
    })
    refreshServers.mockResolvedValue(undefined)
    vi.mocked(connectionsProxy.TestConnection).mockResolvedValue({ isSuccess: true })
  })

  test('happy path: tests, saves, then connects', async () => {
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('mongodb://localhost:27017')
    await w.find('[data-test="connect-btn"]').trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    expect(connectionsProxy.TestConnection).toHaveBeenCalledWith('mongodb://localhost:27017')
    expect(saveServerWithConfig).toHaveBeenCalledWith(
      'localhost:27017',
      '',
      '',
      { uri: 'mongodb://localhost:27017', authMethod: 'password', oidcConfig: undefined },
    )
    expect(connectToServer).toHaveBeenCalledWith('new-server-id')
  })

  test('save succeeds but new ID not in tree shows saveFailed error', async () => {
    saveServerWithConfig.mockResolvedValue({ success: true })
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('mongodb://localhost:27017')
    await w.find('[data-test="connect-btn"]').trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    expect(saveServerWithConfig).toHaveBeenCalled()
    expect(connectToServer).not.toHaveBeenCalled()
    expect(w.find('[data-test="error-alert"]').exists()).toBe(true)
  })

  test('OIDC URI skips TestConnection', async () => {
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('mongodb://host/?authMechanism=MONGODB-OIDC')
    await w.find('[data-test="connect-btn"]').trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    expect(connectionsProxy.TestConnection).not.toHaveBeenCalled()
    expect(saveServerWithConfig).toHaveBeenCalledWith(
      'host',
      '',
      '',
      expect.objectContaining({ authMethod: 'oidc' }),
    )
    expect(connectToServer).toHaveBeenCalled()
  })

  test('failed test prevents save', async () => {
    vi.mocked(connectionsProxy.TestConnection).mockResolvedValue({
      isSuccess: false,
      errorCode: 'CONN_FAIL',
      errorDetail: 'refused',
    })
    const w = makeWrapper()
    await w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea').setValue('mongodb://localhost:27017')
    await w.find('[data-test="connect-btn"]').trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    expect(saveServerWithConfig).not.toHaveBeenCalled()
    expect(connectToServer).not.toHaveBeenCalled()
    expect(w.find('[data-test="error-alert"]').exists()).toBe(true)
  })

  test('editing URI after failure clears error', async () => {
    vi.mocked(connectionsProxy.TestConnection).mockResolvedValue({
      isSuccess: false,
      errorCode: 'CONN_FAIL',
      errorDetail: 'refused',
    })
    const w = makeWrapper()
    const uriField = w.find('[data-test="uri-input"] input, [data-test="uri-input"] textarea')
    await uriField.setValue('mongodb://localhost:27017')
    await w.find('[data-test="connect-btn"]').trigger('click')
    await new Promise((r) => setTimeout(r, 0))
    expect(w.find('[data-test="error-alert"]').exists()).toBe(true)
    await uriField.setValue('mongodb://localhost:27018')
    expect(w.find('[data-test="error-alert"]').exists()).toBe(false)
  })
})
