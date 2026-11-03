import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

import {
  hasDocumentId,
  hasEditBlockingFieldName,
  isDocumentEditable,
  useDocumentContextMenu,
  type CollectionContext,
} from '../useDocumentContextMenu'
import type { DocumentRow } from '../types'

function rootMenuKeys(context: CollectionContext | undefined, hasId = true): string[] {
  const menu = useDocumentContextMenu(ref(context), () => hasId)
  menu.openMenu(
    { kind: 'document' } as DocumentRow,
    {
      preventDefault: () => {},
      clientX: 0,
      clientY: 0,
    } as MouseEvent,
  )
  return menu.menuOptions.value.map((o) => String(o.key))
}

const context: CollectionContext = {
  serverId: 's',
  dbName: 'shop',
  collectionName: 'orders',
  collectionFromTab: false,
  canEdit: true,
}

describe('useDocumentContextMenu', () => {
  it('offers edit, insert and delete when the document can be replaced', () => {
    expect(rootMenuKeys(context)).toEqual(
      expect.arrayContaining(['editDocument', 'insertDocument', 'deleteDocument']),
    )
  })

  it('hides edit, but keeps insert and delete, when fields are hidden', () => {
    const keys = rootMenuKeys({ ...context, canEdit: false })
    expect(keys).not.toContain('editDocument')
    expect(keys).toEqual(expect.arrayContaining(['insertDocument', 'deleteDocument']))
  })

  it('offers no writes without a collection context', () => {
    const keys = rootMenuKeys(undefined)
    expect(keys).not.toContain('insertDocument')
    expect(keys).not.toContain('deleteDocument')
  })

  it('hides edit and delete, but keeps insert, for a document without an _id', () => {
    const keys = rootMenuKeys(context, false)
    expect(keys).not.toContain('editDocument')
    expect(keys).not.toContain('deleteDocument')
    expect(keys).toContain('insertDocument')
  })

  it('offers view as well for a top-level array', () => {
    const menu = useDocumentContextMenu(ref(context), () => false)
    menu.openMenu(
      { kind: 'value', type: 'array' } as DocumentRow,
      { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent,
    )

    const options = menu.menuOptions.value.filter((o) => o.type !== 'divider')
    expect(options.map((o) => o.key)).toEqual(['viewDocument', 'copyValue', 'exportResults'])
    expect(options[0]?.label).toBe('query.contextMenu.viewValue')
  })

  it('offers copy and export, and no writes, for a top-level value', () => {
    const menu = useDocumentContextMenu(ref(context), () => false)
    menu.openMenu(
      { kind: 'value', type: 'string' } as DocumentRow,
      { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent,
    )

    const keys = menu.menuOptions.value.filter((o) => o.type !== 'divider').map((o) => o.key)
    expect(keys).toEqual(['copyValue', 'exportResults'])
  })

  it('offers copy value and copy field for a field inside a document', () => {
    const menu = useDocumentContextMenu(ref(context))
    menu.openMenu(
      { kind: 'field' } as DocumentRow,
      { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent,
    )

    expect(menu.menuOptions.value.map((o) => o.key)).toEqual(['copyValue', 'copyField'])
  })
})

describe('hasDocumentId', () => {
  it.each([
    [{ _id: 1 }, true],
    [{ _id: 0 }, true],
    [{ _id: '' }, true],
    [{ _id: null }, false],
    [{ name: 'a' }, false],
    [null, false],
    ['text', false],
  ])('%j → %s', (doc, expected) => {
    expect(hasDocumentId(doc)).toBe(expected)
  })
})

describe('documents holding dates a script cannot write', () => {
  const farDate = { $date: { $numberLong: '99999999999999999' } }

  it('cannot be edited, since saving would change the date', () => {
    expect(isDocumentEditable({ _id: 1, when: farDate })).toBe(false)
    expect(isDocumentEditable({ _id: 1, nested: [{ when: farDate }] })).toBe(false)
  })

  it('can still be deleted when the date is not the _id', () => {
    expect(hasDocumentId({ _id: 1, when: farDate })).toBe(true)
  })

  it('cannot be edited or deleted when the date is the _id', () => {
    expect(hasDocumentId({ _id: farDate })).toBe(false)
    expect(isDocumentEditable({ _id: farDate })).toBe(false)
  })

  it('stay editable with ordinary dates', () => {
    expect(isDocumentEditable({ _id: 1, when: { $date: { $numberLong: '0' } } })).toBe(true)
  })

  it('hides edit but keeps delete in the menu', () => {
    const menu = useDocumentContextMenu(
      ref(context),
      () => true,
      () => false,
    )
    menu.openMenu(
      { kind: 'document' } as DocumentRow,
      { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent,
    )

    const keys = menu.menuOptions.value.map((o) => String(o.key))
    expect(keys).not.toContain('editDocument')
    expect(keys).toContain('deleteDocument')
  })
})

describe('Edit on a document with a field named like a BSON type', () => {
  const context = {
    serverId: 's',
    dbName: 'd',
    collectionName: 'c',
    collectionFromTab: false,
    canEdit: true,
  }
  const row = { key: '0', kind: 'document', field: '', value: '', type: 'document' } as DocumentRow
  const event = { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent

  function editOption(canEdit: boolean, blockedByName: boolean) {
    const menu = useDocumentContextMenu(
      ref({ ...context, canEdit }),
      () => true,
      () => false,
      () => blockedByName,
    )
    menu.openMenu(row, event)
    return menu.menuOptions.value.find((option) => option.key === 'editDocument')
  }

  it('shows Edit disabled, saying why', () => {
    const option = editOption(true, true)

    expect(option?.disabled).toBe(true)
    expect(option?.props?.title).toBeTruthy()
  })

  it('keeps Edit hidden for other reasons, and where results can never be edited', () => {
    expect(editOption(true, false)).toBeUndefined()
    expect(editOption(false, true)).toBeUndefined()
  })

  it('flags that document and no other', () => {
    expect(hasEditBlockingFieldName({ _id: 1, filter: { name: { $regex: '^a' } } })).toBe(true)
    expect(hasEditBlockingFieldName({ _id: 1, name: 'a' })).toBe(false)
    expect(hasEditBlockingFieldName({ filter: { $regex: '^a' } })).toBe(false)
  })
})
