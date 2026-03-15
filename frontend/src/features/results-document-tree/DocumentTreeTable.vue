<script lang="ts" setup>
import { computed, h, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DataTableColumns, DataTableRowKey } from 'naive-ui'
import { buildTreeData } from './documentTreeUtils'
import type { DocumentRow } from './types'
import { typeColorMap } from '@/features/queries/typeColorMap.ts'

const PAGE_SIZES = [25, 50, 100, 200, 500]

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
  defaultExpandDepth?: number
}>()

const emit = defineEmits<{
  (e: 'update:checkedKeys', keys: DataTableRowKey[]): void
}>()

const { t } = useI18n()

const treeData = computed(() => buildTreeData(props.documents))

const expandedKeys = ref<DataTableRowKey[]>([])
const checkedKeys = ref<DataTableRowKey[]>([])

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
    const depth = props.defaultExpandDepth ?? 1
    expandedKeys.value = collectKeysToDepth(data, depth)
  }
}, { immediate: true })

// const paginationConfig = reactive({
//   page: 1,
//   pageSize: 25,
//   pageSizes: PAGE_SIZES,
//   showSizePicker: true,
//   size: 'small' as const,
//   'on-update:page': (page: number) => {
//     console.log('on-update:page', page)
//     paginationConfig.page = page
//   },
//   'on-update:page-size': (pageSize: number) => {
//     console.log('on-update:page-size', pageSize)
//     paginationConfig.pageSize = pageSize
//     paginationConfig.page = 1
//   },
// })

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

function rowProps(row: DocumentRow) {
  return {
    'on:click.ctrl': () => {
      console.log('ctrl', row)
    },
    onClick: (evt: PointerEvent) => {
      if (evt.ctrlKey) {
        if (checkedKeys.value.includes(row.key)) {
          checkedKeys.value = checkedKeys.value.filter((key) => key !== row.key)
        } else {
          checkedKeys.value = [...checkedKeys.value, row.key]
        }
        console.log(checkedKeys.value, row.key, checkedKeys.value.includes(row.key))
      }
    },
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
