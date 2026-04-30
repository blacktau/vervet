<script setup lang="ts">
import { onMounted, computed, h, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  type DataTableColumns,
  type DataTableRowKey,
  NButton,
  NIcon,
  NTag,
  NTooltip,
  useThemeVars,
} from 'naive-ui'
import { DocumentDuplicateIcon, InformationCircleIcon, PlusIcon } from '@heroicons/vue/24/outline'
import { useSchemaStore } from './schemaStore'
import { buildSchemaRows, type SchemaRow } from './schemaRows'
import { TYPE_PALETTE } from './typePalette'
import { DialogType, useDialogStore } from '@/stores/dialog'
import { useMessager } from '@/utils/dialog'

const props = defineProps<{
  tabId: string
  serverId: string
  dbName: string
  collectionName: string
}>()

const i18n = useI18n()
const { t, te } = useI18n()
const themeVars = useThemeVars()
const store = useSchemaStore()
const messager = useMessager()
const dialogStore = useDialogStore()

const sizeOptions = [
  { label: '100', value: 100 },
  { label: '500', value: 500 },
  { label: '1000', value: 1000 },
  { label: '5000', value: 5000 },
]

const state = computed(() => store.stateFor(props.tabId))

const rows = computed<SchemaRow[]>(() => {
  const result = state.value?.result
  if (!result) {
    return []
  }
  return buildSchemaRows(result.fields, result.sampledCount)
})

const expandedKeys = ref<DataTableRowKey[]>([])

async function run(size: number) {
  await store.sample(props.tabId, props.serverId, props.dbName, props.collectionName, size)
}

function onSizeChange(val: number) {
  void run(val)
}

function onCancel() {
  store.cancel(props.tabId)
}

function onCopyPath(path: string) {
  void navigator.clipboard.writeText(path)
  messager.success(i18n.t('schemaBrowser.copiedPath'))
}

function onCreateIndex(path: string) {
  dialogStore.showNewDialog(DialogType.CreateIndex, {
    serverID: props.serverId,
    dbName: props.dbName,
    collectionName: props.collectionName,
    presetField: path,
  })
}

function typeLabel(type: string): string {
  const key = `schemaBrowser.types.${type}`
  return te(key) ? t(key) : type
}

function typeColor(type: string): string {
  return TYPE_PALETTE[type] ?? themeVars.value.primaryColor
}

function renderTypeLozenges(row: SchemaRow) {
  return h(
    'div',
    { class: 'schema-types' },
    row.types.map((tStat) => {
      const pct = row.count > 0 ? ((tStat.count / row.count) * 100).toFixed(1) : '0'
      const color = typeColor(tStat.type)
      return h(
        NTag,
        {
          key: tStat.type,
          size: 'small',
          round: true,
          bordered: false,
          color: { color, textColor: '#fff' },
          title: `${typeLabel(tStat.type)}: ${tStat.count} (${pct}%)`,
        },
        { default: () => typeLabel(tStat.type) },
      )
    }),
  )
}

function renderActions(row: SchemaRow) {
  return h('div', { class: 'schema-actions' }, [
    h(
      NButton,
      {
        size: 'tiny',
        quaternary: true,
        title: t('schemaBrowser.copyPath'),
        onClick: (e: Event) => {
          e.stopPropagation()
          onCopyPath(row.path)
        },
      },
      {
        icon: () => h(NIcon, null, { default: () => h(DocumentDuplicateIcon) }),
      },
    ),
    h(
      NButton,
      {
        size: 'tiny',
        quaternary: true,
        title: t('schemaBrowser.createIndex'),
        onClick: (e: Event) => {
          e.stopPropagation()
          onCreateIndex(row.path)
        },
      },
      {
        icon: () => h(NIcon, null, { default: () => h(PlusIcon) }),
      },
    ),
  ])
}

const columns = computed<DataTableColumns<SchemaRow>>(() => [
  {
    title: t('schemaBrowser.columns.field'),
    key: 'name',
    resizable: true,
    width: 280,
    minWidth: 120,
    ellipsis: { tooltip: true },
    tree: true,
    render(row) {
      return h(
        'span',
        { class: 'schema-field-name' },
        row.name,
      )
    },
  },
  {
    title: () =>
      h(NTooltip, null, {
        trigger: () => h('span', null, t('schemaBrowser.columns.types')),
        default: () => t('schemaBrowser.typesTooltip'),
      }),
    key: 'types',
    resizable: true,
    minWidth: 200,
    render: renderTypeLozenges,
  },
  {
    title: () =>
      h(NTooltip, null, {
        trigger: () => h('span', null, t('schemaBrowser.columns.presence')),
        default: () => t('schemaBrowser.presenceTooltip'),
      }),
    key: 'presence',
    width: 110,
    align: 'right',
    sorter: (a, b) => a.presence - b.presence,
    render(row) {
      return h(
        'span',
        { class: 'schema-presence' },
        `${row.presence.toFixed(1)}%`,
      )
    },
  },
  {
    title: t('schemaBrowser.columns.actions'),
    key: 'actions',
    width: 80,
    align: 'center',
    render: renderActions,
  },
])

onMounted(() => {
  if (!state.value) {
    void run(1000)
  }
})
</script>

<template>
  <div class="schema-pane">
    <div class="schema-pane__header">
      <NSelect
        :value="state?.sampleSize ?? 1000"
        :options="sizeOptions"
        size="small"
        style="width: 110px"
        @update:value="onSizeChange"
      />
      <NButton size="small" :disabled="state?.loading" @click="run(state?.sampleSize ?? 1000)">
        {{ $t('schemaBrowser.resample') }}
      </NButton>
      <span v-if="state?.result" class="schema-pane__count">
        {{
          $t('schemaBrowser.sampledOf', {
            sampled: state.result.sampledCount,
            total: state.result.totalCount,
          })
        }}
      </span>
      <NButton v-if="state?.loading" size="small" type="warning" @click="onCancel">
        {{ $t('schemaBrowser.cancel') }}
      </NButton>
      <NTooltip placement="bottom-end" trigger="hover">
        <template #trigger>
          <NIcon size="16" class="schema-pane__help">
            <InformationCircleIcon />
          </NIcon>
        </template>
        {{ $t('schemaBrowser.legend.caption') }}
      </NTooltip>
    </div>

    <div class="schema-pane__body">
      <div v-if="state?.loading && !state?.result" class="schema-pane__centered">
        <NSpin />
      </div>
      <div v-else-if="state?.error" class="schema-pane__centered schema-pane__error">
        <span>{{ state.error }}</span>
        <NButton size="small" @click="run(state.sampleSize)">{{ $t('schemaBrowser.retry') }}</NButton>
      </div>
      <div
        v-else-if="state?.result && state.result.fields.length === 0"
        class="schema-pane__centered"
      >
        {{ $t('schemaBrowser.empty') }}
      </div>
      <NDataTable
        v-else-if="state?.result"
        :columns="columns"
        :data="rows"
        :row-key="(row: SchemaRow) => row.key"
        :expanded-row-keys="expandedKeys"
        :loading="state?.loading"
        size="small"
        flex-height
        striped
        class="schema-table"
        @update:expanded-row-keys="(keys: DataTableRowKey[]) => (expandedKeys = keys)"
      />
    </div>
  </div>
</template>

<style scoped>
.schema-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.schema-pane__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-bottom: 1px solid var(--n-border-color, rgba(127, 127, 127, 0.2));
}
.schema-pane__count {
  font-size: 12px;
  opacity: 0.75;
}
.schema-pane__help {
  margin-left: auto;
  cursor: help;
  opacity: 0.6;
}
.schema-pane__help:hover {
  opacity: 1;
}
.schema-pane__body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.schema-pane__centered {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  opacity: 0.75;
}
.schema-pane__error {
  flex-direction: column;
}
.schema-table {
  height: 100%;
}
.schema-table :deep(.schema-types) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.schema-table :deep(.schema-actions) {
  display: flex;
  gap: 2px;
  justify-content: center;
}
.schema-table :deep(.schema-field-name) {
  font-family: var(--font-mono, monospace);
  font-size: 13px;
}
.schema-table :deep(.schema-presence) {
  font-variant-numeric: tabular-nums;
  font-size: 12px;
  opacity: 0.85;
}
</style>
