import { computed, ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DropdownOption } from 'naive-ui'
import type { DocumentRow } from './types'
import { hasAmbiguousEjsonField, hasReorderedKeys, hasUnwritableDate } from './bsonValue'

export interface CollectionContext {
  serverId: string
  dbName: string
  collectionName: string
  /** The query's own collection is unknown, so collectionName is the tab's */
  collectionFromTab: boolean
  /** False when the results hide fields: replacing would delete them */
  canEdit: boolean
}

export function formatNamespace(dbName: string, collectionName: string): string {
  return `${dbName}.${collectionName}`
}

/** Edit and delete target a document by _id; without one they'd match nothing, or the wrong one. */
export function hasDocumentId(doc: unknown): boolean {
  if (typeof doc !== 'object' || doc === null) {
    return false
  }
  const id = (doc as Record<string, unknown>)._id
  return (
    id !== undefined &&
    id !== null &&
    !hasUnwritableDate(id) &&
    !hasAmbiguousEjsonField(id) &&
    !hasReorderedKeys(id)
  )
}

/** Saving replaces the whole document, so every value in it must survive the trip. */
export function isDocumentEditable(doc: unknown): boolean {
  return hasDocumentId(doc) && !hasUnwritableDate(doc) && !hasAmbiguousEjsonField(doc)
}

/**
 * Whether Edit is off because of a field's name. The document looks ordinary, so
 * unlike the other cases Edit stays visible and says why it can't be used.
 */
export function hasEditBlockingFieldName(doc: unknown): boolean {
  return hasDocumentId(doc) && hasAmbiguousEjsonField(doc)
}

export function useDocumentContextMenu(
  collectionContext: Ref<CollectionContext | undefined>,
  rowHasId: (row: DocumentRow) => boolean = () => true,
  rowIsEditable: (row: DocumentRow) => boolean = rowHasId,
  rowHasBlockingFieldName: (row: DocumentRow) => boolean = () => false,
) {
  const { t } = useI18n()
  const showMenu = ref(false)
  const menuX = ref(0)
  const menuY = ref(0)
  const targetRow = ref<DocumentRow | null>(null)

  function openMenu(row: DocumentRow, event: MouseEvent) {
    event.preventDefault()
    targetRow.value = row
    showMenu.value = true
    menuX.value = event.clientX
    menuY.value = event.clientY
  }

  function closeMenu() {
    showMenu.value = false
    targetRow.value = null
  }

  const menuOptions = computed<DropdownOption[]>(() => {
    if (!targetRow.value) {
      return []
    }

    if (targetRow.value.kind === 'document') {
      const targetable = rowHasId(targetRow.value)
      const options: DropdownOption[] = [
        { label: t('query.contextMenu.viewDocument'), key: 'viewDocument' },
      ]

      if (collectionContext.value?.canEdit && rowIsEditable(targetRow.value)) {
        options.push({ label: t('query.contextMenu.editDocument'), key: 'editDocument' })
      } else if (collectionContext.value?.canEdit && rowHasBlockingFieldName(targetRow.value)) {
        options.push({
          label: t('query.contextMenu.editDocument'),
          key: 'editDocument',
          disabled: true,
          props: { title: t('query.dialogs.editBlockedByFieldName') },
        })
      }
      if (collectionContext.value) {
        options.push({ label: t('query.contextMenu.insertDocument'), key: 'insertDocument' })
      }

      options.push(
        { type: 'divider', key: 'd1' },
        { label: t('query.contextMenu.copyDocument'), key: 'copyDocument' },
        { label: t('export.contextMenu'), key: 'exportResults' },
      )

      if (collectionContext.value && targetable) {
        options.push(
          { type: 'divider', key: 'd2' },
          { label: t('query.contextMenu.deleteDocument'), key: 'deleteDocument' },
        )
      }

      return options
    }

    if (targetRow.value.kind === 'value') {
      const view: DropdownOption[] =
        targetRow.value.type === 'array'
          ? [{ label: t('query.contextMenu.viewValue'), key: 'viewDocument' }]
          : []
      return [
        ...view,
        { label: t('query.contextMenu.copyValue'), key: 'copyValue' },
        { type: 'divider', key: 'd1' },
        { label: t('export.contextMenu'), key: 'exportResults' },
      ]
    }

    return [
      { label: t('query.contextMenu.copyValue'), key: 'copyValue' },
      { label: t('query.contextMenu.copyField'), key: 'copyField' },
    ]
  })

  return {
    showMenu,
    menuX,
    menuY,
    targetRow,
    menuOptions,
    openMenu,
    closeMenu,
  }
}
