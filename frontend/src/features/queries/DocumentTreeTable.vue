<script lang="ts" setup>
import { computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { buildTreeData } from './documentTreeUtils'
import type { DocumentRow } from './types'

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
}>()

const emit = defineEmits<{
  (e: 'update:checkedKeys', keys: DataTableRowKey[]): void
}>()

const { t } = useI18n()

const treeData = computed(() => buildTreeData(props.documents))

const expandedKeys = ref<DataTableRowKey[]>([])
const checkedKeys = ref<DataTableRowKey[]>([])

const typeColorMap: Record<string, string> = {
  string: '#ce9178',
  int32: '#b5cea8',
  double: '#b5cea8',
  long: '#b5cea8',
  decimal128: '#b5cea8',
  boolean: '#569cd6',
  null: '#808080',
  objectId: '#dcdcaa',
  date: '#c586c0',
  document: 'var(--n-td-text-color)',
  array: 'var(--n-td-text-color)',
}

const columns = computed<DataTableColumns<DocumentRow>>(() => [
  { type: 'selection', width: 40 },
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

function handleExpandedKeysUpdate(keys: DataTableRowKey[]) {
  expandedKeys.value = keys
}

function handleCheckedKeysUpdate(keys: DataTableRowKey[]) {
  checkedKeys.value = keys
  emit('update:checkedKeys', keys)
}

function rowClassName(row: DocumentRow): string {
  return row.isDocRoot ? 'doc-root-row' : ''
}
</script>

<template>
  <n-data-table
    :columns="columns"
    :data="treeData"
    :row-key="(row: DocumentRow) => row.key"
    :expanded-row-keys="expandedKeys"
    :checked-row-keys="checkedKeys"
    :row-class-name="rowClassName"
    :style="{ height: '100%' }"
    children-key="children"
    virtual-scroll
    flex-height
    size="small"
    @update:expanded-row-keys="handleExpandedKeysUpdate"
    @update:checked-row-keys="handleCheckedKeysUpdate" />
</template>

<style lang="scss" scoped>
:deep(.doc-root-row td) {
  background-color: var(--n-td-color-hover) !important;
}

:deep(.n-data-table) {
  font-family: monospace;
  font-size: 13px;
}
</style>
