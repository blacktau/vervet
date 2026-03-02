<script lang="ts" setup>
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
}>()

const { t } = useI18n()

interface FlatRow {
  key: string
  field: string
  value: string
  type: string
  depth: number
  hasChildren: boolean
  expanded: boolean
  childKeys: string[]
  isDocRoot: boolean
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getTypeName(val: any): string {
  if (val === null) {
    return 'Null'
  }
  if (Array.isArray(val)) {
    return 'Array'
  }
  if (typeof val === 'object') {
    if ('$oid' in val) {
      return 'ObjectId'
    }
    if ('$date' in val) {
      return 'Date'
    }
    if ('$numberDecimal' in val) {
      return 'Decimal128'
    }
    if ('$numberLong' in val) {
      return 'Long'
    }
    if ('$numberInt' in val) {
      return 'Int32'
    }
    if ('$numberDouble' in val) {
      return 'Double'
    }
    if ('$binary' in val) {
      return 'Binary'
    }
    if ('$regex' in val) {
      return 'Regex'
    }
    if ('$timestamp' in val) {
      return 'Timestamp'
    }
    return 'Document'
  }
  if (typeof val === 'string') {
    return 'String'
  }
  if (typeof val === 'number') {
    return Number.isInteger(val) ? 'Int32' : 'Double'
  }
  if (typeof val === 'boolean') {
    return 'Boolean'
  }
  return typeof val
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getDisplayValue(val: any): string {
  if (val === null) {
    return 'null'
  }
  if (Array.isArray(val)) {
    return `[ ${val.length} elements ]`
  }
  if (typeof val === 'object') {
    if ('$oid' in val) {
      return val.$oid
    }
    if ('$date' in val) {
      return String(val.$date)
    }
    if ('$numberDecimal' in val) {
      return val.$numberDecimal
    }
    if ('$numberLong' in val) {
      return val.$numberLong
    }
    if ('$numberInt' in val) {
      return val.$numberInt
    }
    if ('$numberDouble' in val) {
      return val.$numberDouble
    }
    if ('$binary' in val) {
      return `Binary (${val.$binary.subType})`
    }
    if ('$regex' in val) {
      return `/${val.$regex}/${val.$options || ''}`
    }
    if ('$timestamp' in val) {
      return `Timestamp(${val.$timestamp.t}, ${val.$timestamp.i})`
    }
    const keys = Object.keys(val)
    return `{ ${keys.length} fields }`
  }
  if (typeof val === 'string') {
    return val
  }
  return String(val)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getDocLabel(doc: any, index: number): string {
  if (typeof doc !== 'object' || doc === null) {
    return `(${index + 1})`
  }
  const id = doc._id
  if (id !== undefined) {
    const idStr = typeof id === 'object' && id !== null && '$oid' in id
      ? id.$oid
      : String(id)
    return `(${index + 1}) {_id: ${idStr}}`
  }
  return `(${index + 1})`
}

// Build a flat map of all rows (keyed by path) for toggling
const rowMap = reactive<Map<string, FlatRow>>(new Map())

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function buildRowMap(obj: any, prefix: string, depth: number) {
  if (typeof obj !== 'object' || obj === null) {
    return
  }

  const entries = Array.isArray(obj)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    ? obj.map((v, i) => [String(i), v] as [string, any])
    : Object.entries(obj)

  for (const [key, val] of entries) {
    const nodeKey = prefix ? `${prefix}.${key}` : key
    const typeName = getTypeName(val)
    const hasChildren = typeName === 'Document' || typeName === 'Array'

    const childKeys: string[] = []
    if (hasChildren && typeof val === 'object' && val !== null) {
      const childEntries = Array.isArray(val)
        ? val.map((_v, i) => String(i))
        : Object.keys(val)
      for (const ck of childEntries) {
        childKeys.push(`${nodeKey}.${ck}`)
      }
    }

    rowMap.set(nodeKey, {
      key: nodeKey,
      field: key,
      value: getDisplayValue(val),
      type: typeName,
      depth,
      hasChildren,
      expanded: false,
      childKeys,
      isDocRoot: false,
    })

    if (hasChildren) {
      buildRowMap(val, nodeKey, depth + 1)
    }
  }
}

// Build top-level document root nodes + their children
const topLevelKeys = computed(() => {
  rowMap.clear()
  const docs = props.documents
  if (!docs || docs.length === 0) {
    return []
  }

  const rootKeys: string[] = []

  for (let i = 0; i < docs.length; i++) {
    const doc = docs[i]
    const rootKey = `__doc_${i}`
    const fieldCount = typeof doc === 'object' && doc !== null
      ? Object.keys(doc).length
      : 0

    // Build child keys for this document's fields
    const childKeys: string[] = []
    if (typeof doc === 'object' && doc !== null) {
      for (const key of Object.keys(doc)) {
        childKeys.push(`${rootKey}.${key}`)
      }
    }

    rowMap.set(rootKey, {
      key: rootKey,
      field: getDocLabel(doc, i),
      value: `{ ${fieldCount} fields }`,
      type: 'Document',
      depth: 0,
      hasChildren: true,
      expanded: false,
      childKeys,
      isDocRoot: true,
    })

    // Build the nested row map for this document
    if (typeof doc === 'object' && doc !== null) {
      buildRowMap(doc, rootKey, 1)
    }

    rootKeys.push(rootKey)
  }

  return rootKeys
})

// Compute visible rows by walking the tree in order
const visibleRows = computed(() => {
  const result: FlatRow[] = []

  function addVisible(keys: string[]) {
    for (const key of keys) {
      const row = rowMap.get(key)
      if (!row) {
        continue
      }
      result.push(row)
      if (row.expanded && row.childKeys.length > 0) {
        addVisible(row.childKeys)
      }
    }
  }

  addVisible(topLevelKeys.value)
  return result
})

function toggleExpand(row: FlatRow) {
  if (row.hasChildren) {
    row.expanded = !row.expanded
  }
}

function getTypeClass(type: string): string {
  switch (type) {
    case 'String': return 'type-string'
    case 'Int32':
    case 'Double':
    case 'Long':
    case 'Decimal128': return 'type-number'
    case 'Boolean': return 'type-boolean'
    case 'Null': return 'type-null'
    case 'ObjectId': return 'type-objectid'
    case 'Date': return 'type-date'
    case 'Document':
    case 'Array': return 'type-composite'
    default: return ''
  }
}

</script>

<template>
  <div class="tree-table">
    <div class="tree-header">
      <span class="col-key">{{ t('query.key') }}</span>
      <span class="col-value">{{ t('query.value') }}</span>
      <span class="col-type">{{ t('query.type') }}</span>
    </div>
    <div class="tree-body">
      <div
        v-for="row in visibleRows"
        :key="row.key"
        class="tree-row"
        :class="{ clickable: row.hasChildren, 'doc-root': row.isDocRoot }"
        @click="toggleExpand(row)"
      >
        <span class="col-key" :style="{ paddingLeft: row.depth * 16 + 4 + 'px' }">
          <span v-if="row.hasChildren" class="expand-icon">
            {{ row.expanded ? '▼' : '▶' }}
          </span>
          <span v-else class="expand-spacer" />
          <span class="field-name">{{ row.field }}</span>
        </span>
        <span class="col-value" :class="getTypeClass(row.type)">{{ row.value }}</span>
        <span class="col-type type-badge">{{ row.type }}</span>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.tree-table {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  font-size: 13px;
  font-family: monospace;
}

.tree-header {
  display: flex;
  border-bottom: 1px solid var(--n-border-color);
  font-weight: 600;
  font-size: 12px;
  color: var(--n-text-color-3);
  flex-shrink: 0;

  .col-key, .col-value, .col-type {
    padding: 4px 8px;
  }
}

.tree-body {
  overflow: auto;
  flex: 1;
}

.tree-row {
  display: flex;
  border-bottom: 1px solid var(--n-divider-color);

  &.clickable {
    cursor: pointer;
  }

  &:hover {
    background: var(--n-color-hover);
  }

  &.doc-root {
    background: var(--n-color-hover);

    &:hover {
      background: var(--n-color-pressed);
    }
  }
}

.col-key {
  flex: 2;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 4px;
  overflow: hidden;
}

.col-value {
  flex: 2;
  padding: 3px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-type {
  flex: 1;
  padding: 3px 8px;
  min-width: 80px;
}

.expand-icon {
  font-size: 10px;
  width: 14px;
  flex-shrink: 0;
  text-align: center;
}

.expand-spacer {
  display: inline-block;
  width: 14px;
  flex-shrink: 0;
}

.field-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-badge {
  font-size: 11px;
  color: var(--n-text-color-3);
}

.type-string { color: #ce9178; }
.type-number { color: #b5cea8; }
.type-boolean { color: #569cd6; }
.type-null { color: #808080; }
.type-objectid { color: #dcdcaa; }
.type-date { color: #c586c0; }
.type-composite { color: var(--n-text-color-2); }
</style>
