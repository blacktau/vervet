// @vitest-environment happy-dom
//
// The dialog writes to whatever namespace it is given, so it has to say which
// one, and flag when the collection is only the tab's guess.
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive from 'naive-ui'
import enGB from '@/i18n/en-GB'

vi.mock('monaco-editor', () => ({
  editor: {
    create: () => ({
      addCommand: vi.fn(),
      setValue: vi.fn(),
      getValue: () => '{}',
      dispose: vi.fn(),
    }),
    setTheme: vi.fn(),
  },
  KeyMod: { CtrlCmd: 2048 },
  KeyCode: { KeyS: 49 },
}))

vi.mock('wailsjs/go/api/ShellProxy', () => ({ ExecuteQuery: vi.fn() }))

vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ error: vi.fn() }),
}))

const i18n = createI18n({ legacy: false, locale: 'en-GB', messages: { 'en-GB': enGB } })

let wrapper: VueWrapper | undefined

async function openDialog(props: { mode: 'edit' | 'insert'; collectionFromTab?: boolean }) {
  const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
  wrapper = mount(DocumentEditDialog, {
    props: {
      show: false,
      document: { _id: 1, name: 'a' },
      serverId: 'server-1',
      dbName: 'shop',
      collectionName: 'orders',
      ...props,
    },
    global: { plugins: [naive, i18n] },
    attachTo: document.body,
  })
  await wrapper.setProps({ show: true })
  await flushPromises()
  // NModal teleports its content to the body
  return document.body.textContent ?? ''
}

beforeEach(() => {
  setActivePinia(createPinia())
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

describe('DocumentEditDialog target', () => {
  it('names the namespace an edit is saved to', async () => {
    const text = await openDialog({ mode: 'edit', collectionFromTab: false })
    expect(text).toContain('Edit Document in shop.orders')
    expect(text).not.toContain("tab's collection")
  })

  it('names the namespace an insert goes into', async () => {
    const text = await openDialog({ mode: 'insert', collectionFromTab: false })
    expect(text).toContain('Insert Document into shop.orders')
  })

  it("warns when the collection is the tab's fallback", async () => {
    const text = await openDialog({ mode: 'edit', collectionFromTab: true })
    expect(text).toContain("changes go to the tab's collection, orders")
  })
})
