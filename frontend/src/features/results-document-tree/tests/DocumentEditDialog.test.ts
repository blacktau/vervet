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
import * as shellProxy from 'wailsjs/go/api/ShellProxy'
import * as monaco from 'monaco-editor'

vi.mock('monaco-editor', () => ({
  editor: {
    create: vi.fn(() => ({
      addCommand: vi.fn(),
      setValue: vi.fn(),
      getValue: () => '{}',
      dispose: vi.fn(),
    })),
    setTheme: vi.fn(),
  },
  KeyMod: { CtrlCmd: 2048 },
  KeyCode: { KeyS: 49 },
}))

vi.mock('wailsjs/go/api/ShellProxy', () => ({
  ExecuteQuery: vi.fn(async () => ({ isSuccess: true })),
}))

vi.mock('@/utils/dialog', () => ({
  useNotifier: () => ({ error: vi.fn() }),
}))

const i18n = createI18n({ legacy: false, locale: 'en-GB', messages: { 'en-GB': enGB } })

let wrapper: VueWrapper | undefined

async function openDialog(props: {
  mode: 'edit' | 'insert'
  collectionFromTab?: boolean
  collectionName?: string
}) {
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
    expect(text).not.toContain("don't say which collection")
  })

  it('names the namespace an insert goes into', async () => {
    const text = await openDialog({ mode: 'insert', collectionFromTab: false })
    expect(text).toContain('Insert Document into shop.orders')
  })

  it("warns when the collection is the tab's fallback", async () => {
    const text = await openDialog({ mode: 'edit', collectionFromTab: true })
    expect(text).toContain('changes go to shop.orders')
  })
})

describe('DocumentEditDialog save', () => {
  it.each(['edit', 'insert'] as const)(
    'quotes the collection name safely in the %s script',
    async (mode) => {
      await openDialog({ mode, collectionName: "o'brien" })
      const save = [...document.body.querySelectorAll('button')].find(
        (b) => b.textContent?.trim() === 'Save',
      )
      save!.click()
      await flushPromises()

      const script = vi.mocked(shellProxy.ExecuteQuery).mock.lastCall![3]
      expect(script).toContain(`db.getCollection("o'brien").`)
    },
  )
})

describe('DocumentEditDialog keeps BSON types', () => {
  async function mountEdit(doc: unknown) {
    vi.mocked(monaco.editor.create).mockClear()
    const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
    wrapper = mount(DocumentEditDialog, {
      props: {
        show: true,
        document: doc,
        mode: 'edit',
        serverId: 'server-1',
        dbName: 'shop',
        collectionName: 'orders',
      },
      global: { plugins: [naive, i18n] },
      attachTo: document.body,
    })
    await flushPromises()
  }

  it('shows types plain JSON cannot hold as Extended JSON, without the _id', async () => {
    await mountEdit({
      _id: { $oid: '507f1f77bcf86cd799439011' },
      long: { $numberLong: '7' },
      whole: { $numberDouble: '5.0' },
      int: { $numberInt: '5' },
    })

    const shown = vi.mocked(monaco.editor.create).mock.lastCall![1]!.value!
    expect(JSON.parse(shown)).toEqual({
      long: { $numberLong: '7' },
      whole: { $numberDouble: '5.0' },
      int: 5,
    })
  })

  it('saves through a strict EJSON.parse of the editor text', async () => {
    await mountEdit({ _id: { $oid: '507f1f77bcf86cd799439011' }, name: 'a' })

    const save = [...document.body.querySelectorAll('button')].find(
      (b) => b.textContent?.trim() === 'Save',
    )
    save!.click()
    await flushPromises()

    expect(vi.mocked(shellProxy.ExecuteQuery).mock.lastCall![3]).toBe(
      'db.getCollection("orders").replaceOne(' +
        'EJSON.parse("{\\"_id\\":{\\"$oid\\":\\"507f1f77bcf86cd799439011\\"}}", { relaxed: false }), ' +
        'EJSON.parse("{}", { relaxed: false }))',
    )
  })
})

describe('DocumentEditDialog save checks', () => {
  async function saveWithText(text: string, reply?: unknown) {
    vi.mocked(monaco.editor.create).mockReturnValueOnce({
      addCommand: vi.fn(),
      setValue: vi.fn(),
      getValue: () => text,
      dispose: vi.fn(),
    } as never)
    vi.mocked(shellProxy.ExecuteQuery).mockClear()
    if (reply) {
      vi.mocked(shellProxy.ExecuteQuery).mockResolvedValueOnce(reply as never)
    }
    const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
    wrapper = mount(DocumentEditDialog, {
      props: {
        show: true,
        document: { _id: 1, name: 'a' },
        mode: 'edit',
        serverId: 'server-1',
        dbName: 'shop',
        collectionName: 'orders',
      },
      global: { plugins: [naive, i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    ;[...document.body.querySelectorAll('button')]
      .find((b) => b.textContent?.trim() === 'Save')!
      .click()
    await flushPromises()
  }

  it.each(['null', '[1]', '5', '"a"'])('does not save %s, which is not a document', async (text) => {
    await saveWithText(text)

    expect(shellProxy.ExecuteQuery).not.toHaveBeenCalled()
    expect(document.body.textContent).toContain('A document must be a JSON object.')
    expect(wrapper!.emitted('saved')).toBeUndefined()
  })

  it('says so and stays open when the save matched nothing', async () => {
    await saveWithText('{"a":1}', { isSuccess: true, data: { documents: [{ matchedCount: 0 }] } })

    expect(document.body.textContent).toContain('this document was not found in shop.orders')
    expect(wrapper!.emitted('saved')).toBeUndefined()
    expect(wrapper!.emitted('update:show')).toBeUndefined()
  })

  it('sends one insert when Ctrl+S is pressed again during a save', async () => {
    let finish: (reply: unknown) => void = () => {}
    vi.mocked(shellProxy.ExecuteQuery).mockClear()
    vi.mocked(shellProxy.ExecuteQuery).mockReturnValueOnce(
      new Promise((resolve) => {
        finish = resolve
      }) as never,
    )
    const addCommand = vi.fn()
    vi.mocked(monaco.editor.create).mockReturnValueOnce({
      addCommand,
      setValue: vi.fn(),
      getValue: () => '{"a":1}',
      dispose: vi.fn(),
    } as never)
    const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
    wrapper = mount(DocumentEditDialog, {
      props: {
        show: true,
        document: {},
        mode: 'insert',
        serverId: 'server-1',
        dbName: 'shop',
        collectionName: 'orders',
      },
      global: { plugins: [naive, i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    const pressCtrlS = addCommand.mock.lastCall![1]

    pressCtrlS()
    pressCtrlS()
    await flushPromises()
    expect(shellProxy.ExecuteQuery).toHaveBeenCalledTimes(1)

    finish({ isSuccess: true })
    await flushPromises()
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('refuses to save an edit of a document with no _id', async () => {
    vi.mocked(shellProxy.ExecuteQuery).mockClear()
    const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
    wrapper = mount(DocumentEditDialog, {
      props: {
        show: true,
        document: { name: 'a' },
        mode: 'edit',
        serverId: 'server-1',
        dbName: 'shop',
        collectionName: 'orders',
      },
      global: { plugins: [naive, i18n] },
      attachTo: document.body,
    })
    await flushPromises()
    ;[...document.body.querySelectorAll('button')]
      .find((b) => b.textContent?.trim() === 'Save')!
      .click()
    await flushPromises()

    expect(shellProxy.ExecuteQuery).not.toHaveBeenCalled()
    expect(wrapper.emitted('saved')).toBeUndefined()
  })

  it('closes when the save matched', async () => {
    await saveWithText('{"a":1}', { isSuccess: true, data: { documents: [{ matchedCount: 1 }] } })

    expect(wrapper!.emitted('saved')).toHaveLength(1)
  })
})

describe('DocumentEditDialog mounted open', () => {
  // The table mounts the dialog the first time it opens, so it starts shown
  it('builds the editor and shows the _id', async () => {
    vi.mocked(monaco.editor.create).mockClear()
    const DocumentEditDialog = (await import('../DocumentEditDialog.vue')).default
    wrapper = mount(DocumentEditDialog, {
      props: {
        show: true,
        document: { _id: 7, name: 'a' },
        mode: 'edit',
        serverId: 'server-1',
        dbName: 'shop',
        collectionName: 'orders',
      },
      global: { plugins: [naive, i18n] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(monaco.editor.create).toHaveBeenCalledTimes(1)
    expect(document.body.textContent).toContain('_id: 7')
  })
})
