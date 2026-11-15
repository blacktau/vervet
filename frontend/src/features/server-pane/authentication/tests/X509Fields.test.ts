// @vitest-environment happy-dom
import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createI18n } from 'vue-i18n'
import X509Fields from '@/features/server-pane/authentication/fields/X509Fields.vue'

const selectFile = vi.fn()
vi.mock('bindings/vervet/internal/api/filesproxy', () => ({ SelectFile: (...args: unknown[]) => selectFile(...args) }))

const notifyError = vi.fn()
vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ error: notifyError }),
}))

function mountAndBrowse() {
  const wrapper = shallowMount(X509Fields, {
    props: { uri: 'mongodb://host/' },
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en-GB', missingWarn: false, fallbackWarn: false })],
      mocks: { $t: (key: string) => key },
      // Naive UI's own component names, without the N prefix.
      stubs: {
        Form: { template: '<div><slot /></div>' },
        FormItem: { template: '<div><slot /></div>' },
        InputGroup: { template: '<div><slot /></div>' },
        Button: { template: '<button @click="$emit(\'click\')"><slot /></button>', emits: ['click'] },
      },
    },
  })
  return wrapper.find('button').trigger('click').then(async () => {
    await flushPromises()
    return wrapper
  })
}

describe('X509Fields browse', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('tells the user when the dialog fails and leaves the URI alone', async () => {
    selectFile.mockResolvedValue({ isSuccess: false, data: '', errorCode: 'unknown_error', errorDetail: 'no display' })

    const wrapper = await mountAndBrowse()

    expect(notifyError).toHaveBeenCalledOnce()
    expect(notifyError.mock.calls[0]![0]).toBe('no display')
    expect(wrapper.emitted('update:uri')).toBeUndefined()
  })

  test('stays silent when the dialog is cancelled', async () => {
    selectFile.mockResolvedValue({ isSuccess: true, data: '' })

    const wrapper = await mountAndBrowse()

    expect(notifyError).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:uri')).toBeUndefined()
  })

  test('uses the chosen file', async () => {
    selectFile.mockResolvedValue({ isSuccess: true, data: '/certs/client.pem' })

    const wrapper = await mountAndBrowse()

    expect(notifyError).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:uri')).toHaveLength(1)
  })
})
