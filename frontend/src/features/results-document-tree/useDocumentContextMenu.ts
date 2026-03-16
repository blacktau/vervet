import { computed, ref, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DropdownOption } from 'naive-ui'
import type { DocumentRow } from './types'

export interface CollectionContext {
  serverId: string
  dbName: string
  collectionName: string
}

export function useDocumentContextMenu(collectionContext: Ref<CollectionContext | undefined>) {
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

    if (targetRow.value.isDocRoot) {
      const options: DropdownOption[] = [
        { label: t('query.contextMenu.viewDocument'), key: 'viewDocument' },
      ]

      if (collectionContext.value) {
        options.push(
          { label: t('query.contextMenu.editDocument'), key: 'editDocument' },
          { label: t('query.contextMenu.insertDocument'), key: 'insertDocument' },
        )
      }

      options.push(
        { type: 'divider', key: 'd1' },
        { label: t('query.contextMenu.copyDocument'), key: 'copyDocument' },
      )

      if (collectionContext.value) {
        options.push(
          { type: 'divider', key: 'd2' },
          { label: t('query.contextMenu.deleteDocument'), key: 'deleteDocument' },
        )
      }

      return options
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
