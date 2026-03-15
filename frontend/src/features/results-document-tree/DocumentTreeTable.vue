<script lang="ts" setup>
import { computed, h, reactive, ref, toRef, watch, type Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { buildTreeData } from './documentTreeUtils'
import type { DocumentRow } from './types'
import { typeColorMap } from '@/features/queries/typeColorMap'
import { useDocumentContextMenu, type CollectionContext } from './useDocumentContextMenu'
import { useNotifier } from '@/utils/dialog'
import { resolveRawValue } from './resolveRawValue'
import DocumentContextMenu from './DocumentContextMenu.vue'
import DocumentViewDialog from './DocumentViewDialog.vue'
import DocumentEditDialog from './DocumentEditDialog.vue'
import * as shellProxy from 'wailsjs/go/api/ShellProxy'

const PAGE_SIZES = [25, 50, 100, 200, 500]

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
  defaultExpandDepth?: number
  enableContextMenu?: boolean
  collectionContext?: CollectionContext
}>()

const emit = defineEmits<{
  (e: 'update:checkedKeys', keys: DataTableRowKey[]): void
  (e: 'document-changed'): void
}>()

const { t } = useI18n()

const treeData = computed(() => buildTreeData(props.documents))

const expandedKeys = ref<DataTableRowKey[]>([])
const checkedKeys = ref<DataTableRowKey[]>([])

// Context menu
const collectionContextRef = toRef(props, 'collectionContext')
const contextMenu = useDocumentContextMenu(collectionContextRef as Ref<CollectionContext | undefined>)
const notifier = useNotifier()
const dialog = useDialog()

// Dialog state
const showViewDialog = ref(false)
const showEditDialog = ref(false)
const viewDocument = ref<unknown>(null)
const editDocument = ref<unknown>(null)
const editMode = ref<'edit' | 'insert'>('edit')

function collectKeysToDepth(rows: DocumentRow[], depth: number, currentDepth: number = 0): string[] {
  if (currentDepth >= depth) {
    return []
  }
  const keys: string[] = []
  for (const row of rows) {
    if (row.children && row.children.length > 0) {
      keys.push(row.key)
      keys.push(...collectKeysToDepth(row.children, depth, currentDepth + 1))
    }
  }
  return keys
}

watch(treeData, (data) => {
  if (data.length > 0) {
    const depth = props.defaultExpandDepth ?? 0
    expandedKeys.value = collectKeysToDepth(data, depth)
  }
}, { immediate: true })

const pagination = reactive({
  page: 1,
  pageSize: 25,
  pageSizes: PAGE_SIZES,
  showSizePicker: true,
  size: 'small' as const,
  onChange: (page: number) => {
    pagination.page = page
  },
  onUpdatePageSize: (pageSize: number) => {
    pagination.pageSize = pageSize
  },
})

const columns = computed<DataTableColumns<DocumentRow>>(() => [
  {
    title: t('query.key'),
    key: 'field',
    resizable: true,
    width: 250,
    minWidth: 80,
    ellipsis: { tooltip: true },
    tree: true,
    render(row: DocumentRow) {
      return h(
        'span',
        {
          style: {
            fontWeight: row.isDocRoot ? '600' : undefined,
          },
        },
        row.field,
      )
    },
  },
  {
    title: t('query.value'),
    key: 'value',
    resizable: true,
    width: 350,
    minWidth: 80,
    ellipsis: { tooltip: true },
    render(row: DocumentRow) {
      return h(
        'span',
        {
          style: {
            color: typeColorMap[row.type] ?? 'var(--',
            userSelect: 'text',
            cursor: 'text',
          },
        },
        row.value,
      )
    },
  },
  {
    title: t('query.type'),
    key: 'typeLabel',
    width: 100,
    minWidth: 60,
    render(row: DocumentRow) {
      return h(
        'span',
        {
          style: {
            fontSize: '11px',
            color: 'var(--n-td-text-color)',
            opacity: '0.6',
          },
        },
        row.typeLabel,
      )
    },
  },
])

function handleContextMenuSelect(key: string) {
  const row = contextMenu.targetRow.value
  if (!row) {
    return
  }

  if (key === 'viewDocument') {
    viewDocument.value = resolveRawValue(props.documents, row.key)
    showViewDialog.value = true
  }

  if (key === 'editDocument') {
    editDocument.value = resolveRawValue(props.documents, row.key)
    editMode.value = 'edit'
    showEditDialog.value = true
  }

  if (key === 'insertDocument') {
    editDocument.value = {}
    editMode.value = 'insert'
    showEditDialog.value = true
  }

  if (key === 'deleteDocument') {
    const doc = resolveRawValue(props.documents, row.key) as Record<string, unknown>
    const idDisplay = doc?._id ? JSON.stringify(doc._id) : 'unknown'
    dialog.warning({
      title: t('query.dialogs.deleteConfirmTitle'),
      content: `${t('query.dialogs.deleteConfirmContent')}\n\n_id: ${idDisplay}`,
      positiveText: t('common.confirm'),
      negativeText: t('common.cancel'),
      onPositiveClick: async () => {
        if (!props.collectionContext) {
          return
        }
        const { serverId, dbName, collectionName } = props.collectionContext
        const filter = JSON.stringify({ _id: doc._id })
        const query = `db.getCollection('${collectionName}').deleteOne(${filter})`
        const result = await shellProxy.ExecuteQuery(serverId, dbName, query)
        if (result.isSuccess) {
          emit('document-changed')
        } else {
          notifier.error(result.error)
        }
      },
    })
  }

  if (key === 'copyDocument') {
    const doc = resolveRawValue(props.documents, row.key)
    copyToClipboard(JSON.stringify(doc, null, 2))
  }

  if (key === 'copyValue') {
    const val = resolveRawValue(props.documents, row.key)
    copyToClipboard(JSON.stringify(val))
  }

  if (key === 'copyField') {
    const val = resolveRawValue(props.documents, row.key)
    copyToClipboard(`"${row.field}": ${JSON.stringify(val)}`)
  }
}

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    notifier.success(t('query.contextMenu.copied'))
  } catch {
    notifier.error('Failed to copy to clipboard')
  }
}

function handleDocumentSaved() {
  emit('document-changed')
}

function rowProps(row: DocumentRow) {
  return {
    onClick: (evt: PointerEvent) => {
      if (evt.ctrlKey) {
        if (checkedKeys.value.includes(row.key)) {
          checkedKeys.value = checkedKeys.value.filter((key) => key !== row.key)
        } else {
          checkedKeys.value = [...checkedKeys.value, row.key]
        }
      }
    },
    onContextmenu: props.enableContextMenu
      ? (evt: MouseEvent) => {
          contextMenu.openMenu(row, evt)
        }
      : undefined,
  }
}

function handleExpandedKeysUpdate(keys: DataTableRowKey[]) {
  expandedKeys.value = keys
}

function handleCheckedKeysUpdate(keys: DataTableRowKey[]) {
  checkedKeys.value = keys
  emit('update:checkedKeys', keys)
}

function rowClassName(row: DocumentRow): string {
  if (row.isDocRoot) {
    return 'doc-root-row'
  }
  if (checkedKeys.value.includes(row.key)) {
    return 'selected-row'
  }

  return ''
}
</script>

<template>
  <n-data-table
    :columns="columns"
    :data="treeData"
    :pagination="pagination"
    :row-key="(row: DocumentRow) => row.key"
    :expanded-row-keys="expandedKeys"
    :checked-row-keys="checkedKeys"
    :row-class-name="rowClassName"
    :row-props="rowProps"
    :style="{ height: '100%' }"
    children-key="children"
    striped
    virtual-scroll
    flex-height
    size="small"
    @update:expanded-row-keys="handleExpandedKeysUpdate"
    @update:checked-row-keys="handleCheckedKeysUpdate" />
  <template v-if="enableContextMenu">
    <DocumentContextMenu
      :show="contextMenu.showMenu.value"
      :x="contextMenu.menuX.value"
      :y="contextMenu.menuY.value"
      :options="contextMenu.menuOptions.value"
      @select="handleContextMenuSelect"
      @close="contextMenu.closeMenu" />
    <DocumentViewDialog
      v-model:show="showViewDialog"
      :document="viewDocument" />
    <DocumentEditDialog
      v-if="collectionContext"
      v-model:show="showEditDialog"
      :document="editDocument"
      :mode="editMode"
      :server-id="collectionContext.serverId"
      :db-name="collectionContext.dbName"
      :collection-name="collectionContext.collectionName"
      @saved="handleDocumentSaved" />
  </template>
</template>

<style lang="scss" scoped>
:deep(.doc-root-row td) {
  background-color: var(--n-td-color-hover) !important;
}
:deep(.selected-row td) {
  background-color: greenyellow !important;
}

:deep(.n-data-table) {
  font-family: monospace;
  font-size: 13px;
}
</style>
