<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { type IndexInfo, useIndexStore } from '@/features/indexes/indexStore.ts'
import { useDialogStore } from '@/stores/dialog.ts'
import { useDialoger } from '@/utils/dialog.ts'
import IconPlus from '~icons/tabler/plus'
import IconEdit from '~icons/tabler/edit'
import IconTrash from '~icons/tabler/trash'

const props = defineProps<{
  serverId: string
  dbName: string
  collectionName: string
}>()

const { t } = useI18n()
const indexStore = useIndexStore()
const dialogStore = useDialogStore()

const selectedIndexName = ref<string | undefined>(undefined)

const indexes = computed(() => indexStore.getIndexList(props.serverId, props.dbName, props.collectionName))
const loading = computed(() => indexStore.isLoading(props.serverId, props.dbName, props.collectionName))

const selectedIndex = computed(() => indexes.value.find((i) => i.name === selectedIndexName.value))

const isIdIndex = computed(() => selectedIndexName.value === '_id_')
const canEditOrDrop = computed(() => selectedIndexName.value != null && !isIdIndex.value)

const columns = computed(() => [
  { title: t('indexes.columns.name'), key: 'name' },
  {
    title: t('indexes.columns.keys'),
    key: 'keys',
    render: (row: IndexInfo) =>
      row.keys.map((k) => `${k.field}: ${k.direction}`).join(', '),
  },
  {
    title: t('indexes.columns.unique'),
    key: 'unique',
    render: (row: IndexInfo) => (row.unique ? 'Yes' : ''),
    width: 80,
  },
  {
    title: t('indexes.columns.sparse'),
    key: 'sparse',
    render: (row: IndexInfo) => (row.sparse ? 'Yes' : ''),
    width: 80,
  },
  {
    title: t('indexes.columns.ttl'),
    key: 'ttl',
    render: (row: IndexInfo) => (row.ttl != null ? String(row.ttl) : ''),
    width: 120,
  },
])

function rowProps(row: IndexInfo) {
  return {
    style: 'cursor: pointer',
    onClick: () => {
      selectedIndexName.value = row.name === selectedIndexName.value ? undefined : row.name
    },
  }
}

function rowClassName(row: IndexInfo) {
  return row.name === selectedIndexName.value ? 'selected-row' : ''
}

function handleAdd() {
  dialogStore.openCreateIndexDialog(props.serverId, props.dbName, props.collectionName)
}

function handleEdit() {
  if (!selectedIndex.value) {
    return
  }
  dialogStore.openEditIndexDialog(
    props.serverId,
    props.dbName,
    props.collectionName,
    selectedIndex.value,
  )
}

function handleDrop() {
  if (!selectedIndexName.value || isIdIndex.value) {
    return
  }
  const name = selectedIndexName.value
  const dialoger = useDialoger()
  dialoger.warning(t('indexes.dialogs.drop.message', { name }), async () => {
    const success = await indexStore.dropIndex(props.serverId, props.dbName, props.collectionName, name)
    if (success) {
      selectedIndexName.value = undefined
    }
  })
}

onMounted(() => {
  indexStore.getIndexes(props.serverId, props.dbName, props.collectionName)
})
</script>

<template>
  <div class="index-tab">
    <div class="index-toolbar">
      <n-button-group size="small">
        <n-button @click="handleAdd">
          <template #icon>
            <n-icon :component="IconPlus" />
          </template>
          {{ t('indexes.toolbar.addIndex') }}
        </n-button>
        <n-button :disabled="!canEditOrDrop" @click="handleEdit">
          <template #icon>
            <n-icon :component="IconEdit" />
          </template>
          {{ t('indexes.toolbar.editIndex') }}
        </n-button>
        <n-button :disabled="!canEditOrDrop" @click="handleDrop">
          <template #icon>
            <n-icon :component="IconTrash" />
          </template>
          {{ t('indexes.toolbar.dropIndex') }}
        </n-button>
      </n-button-group>
    </div>
    <div class="index-table-wrapper">
      <n-data-table
        :columns="columns"
        :data="indexes"
        :loading="loading"
        :row-class-name="rowClassName"
        :row-props="rowProps"
        :bordered="false"
        flex-height
        size="small" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.index-tab {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.index-toolbar {
  padding: 8px 12px;
  flex-shrink: 0;
}

.index-table-wrapper {
  flex: 1;
  min-height: 0;
}

:deep(.selected-row td) {
  background-color: var(--n-td-color-hover) !important;
}
</style>
