// @vitest-environment happy-dom
//
// The edit dialog must keep the namespace it opened with. Starting another
// query replaces the table's live collection context, and a save made after
// that still has to go where the document came from.
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import naive, { NDialogProvider } from 'naive-ui'
import type { CollectionContext } from '../useDocumentContextMenu'

vi.mock('wailsjs/go/api/ShellProxy', () => ({ ExecuteQuery: vi.fn() }))
vi.mock('@/utils/dialog', () => ({ useNotifier: () => ({ error: vi.fn(), success: vi.fn() }) }))

const EditDialogStub = defineComponent({
  name: 'DocumentEditDialog',
  props: ['show', 'document', 'mode', 'serverId', 'dbName', 'collectionName', 'collectionFromTab'],
  render: () => h('div'),
})
const ContextMenuStub = defineComponent({
  name: 'DocumentContextMenu',
  props: ['show', 'x', 'y', 'options'],
  emits: ['select', 'close'],
  render: () => h('div'),
})

// The real table virtualises its rows, which needs layout happy-dom lacks.
// This renders each row with the table's own rowProps, so the context menu
// handlers under test still run.
const DataTableStub = defineComponent({
  name: 'NDataTable',
  props: ['data', 'rowProps'],
  setup(props) {
    return () =>
      h(
        'div',
        (props.data as unknown[]).map((row) => h('div', { class: 'row', ...props.rowProps(row) })),
      )
  },
})

const i18n = createI18n({ legacy: false, locale: 'en-GB', messages: { 'en-GB': {} } })

const shopOrders: CollectionContext = {
  serverId: 'server-1',
  dbName: 'shop',
  collectionName: 'orders',
  collectionFromTab: false,
  canEdit: true,
}

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('DocumentTreeTable edit context', () => {
  it('keeps the namespace the edit dialog opened with', async () => {
    const DocumentTreeTable = (await import('../DocumentTreeTable.vue')).default
    const context = ref<CollectionContext | undefined>(shopOrders)
    const wrapper = mount(
      {
        render: () =>
          h(NDialogProvider, null, {
            default: () =>
              h(DocumentTreeTable, {
                documents: [{ _id: 1, name: 'a' }],
                enableContextMenu: true,
                collectionContext: context.value,
              }),
          }),
      },
      {
        global: {
          plugins: [naive, i18n],
          stubs: {
            DocumentEditDialog: EditDialogStub,
            DocumentContextMenu: ContextMenuStub,
            DocumentViewDialog: true,
            DataTable: DataTableStub,
          },
        },
      },
    )
    await flushPromises()

    const row = wrapper.find('.row')
    expect(row.exists()).toBe(true)
    await row.trigger('contextmenu')
    wrapper.findComponent(ContextMenuStub).vm.$emit('select', 'editDocument')
    await flushPromises()
    expect(wrapper.findComponent(EditDialogStub).props('dbName')).toBe('shop')

    // A new query starts while the dialog is open
    context.value = { ...shopOrders, dbName: 'archive', collectionFromTab: true }
    await flushPromises()

    const dialog = wrapper.findComponent(EditDialogStub)
    expect(dialog.exists()).toBe(true)
    expect(dialog.props('dbName')).toBe('shop')
    expect(dialog.props('collectionFromTab')).toBe(false)
    wrapper.unmount()
  })
})
