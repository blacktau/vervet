import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

import { useDocumentContextMenu, type CollectionContext } from '../useDocumentContextMenu'
import type { DocumentRow } from '../types'

function rootMenuKeys(context: CollectionContext | undefined): string[] {
  const menu = useDocumentContextMenu(ref(context))
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
})
