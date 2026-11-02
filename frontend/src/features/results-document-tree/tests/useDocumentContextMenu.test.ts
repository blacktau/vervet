import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

import {
  hasDocumentId,
  useDocumentContextMenu,
  type CollectionContext,
} from '../useDocumentContextMenu'
import type { DocumentRow } from '../types'

function rootMenuKeys(context: CollectionContext | undefined, hasId = true): string[] {
  const menu = useDocumentContextMenu(ref(context), () => hasId)
  menu.openMenu(
    { isDocRoot: true } as DocumentRow,
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

  it('offers copy and export, and no writes, for a top-level value', () => {
    const menu = useDocumentContextMenu(ref(context), () => false)
    menu.openMenu(
      { isDocRoot: false, isRootValue: true } as DocumentRow,
      { preventDefault: () => {}, clientX: 0, clientY: 0 } as MouseEvent,
    )

    const keys = menu.menuOptions.value.filter((o) => o.type !== 'divider').map((o) => o.key)
    expect(keys).toEqual(['copyValue', 'exportResults'])
  })

  it('offers copy value and copy field for a field inside a document', () => {
    const menu = useDocumentContextMenu(ref(context))
    menu.openMenu(
      { isDocRoot: false, isRootValue: false } as DocumentRow,
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
