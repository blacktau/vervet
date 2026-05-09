// @vitest-environment happy-dom
import { describe, expect, test, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive from 'naive-ui'
import OnboardingPanel from '@/features/onboarding/OnboardingPanel.vue'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({
  TestConnection: vi.fn(),
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
        errors: { CONN_FAIL: 'Connection failed' },
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
