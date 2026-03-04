<script lang="ts" setup>
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getDocLabel, getTypeClass, getTypeName, buildRowMap } from './documentTreeUtils'
import type { FlatRow } from './types'

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
}>()

const { t } = useI18n()

// Build a flat map of all rows (keyed by path) for toggling
const rowMap = reactive<Map<string, FlatRow>>(new Map())


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
      type: 'document',
      depth: 0,
      hasChildren: true,
      expanded: false,
      childKeys,
      isDocRoot: true,
    })

    // Build the nested row map for this document
    if (typeof doc === 'object' && doc !== null) {
      buildRowMap(doc, rootKey, 1, rowMap)
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

const colKeyWidth = ref(0)
const colValueWidth = ref(0)
const tableRef = ref<HTMLElement | null>(null)

function getColumnStyles() {
  if (colKeyWidth.value > 0 && colValueWidth.value > 0) {
    return {
      key: { width: colKeyWidth.value + 'px', flex: 'none' },
      value: { width: colValueWidth.value + 'px', flex: 'none' },
      type: { flex: '1', minWidth: '80px' },
    }
  }
  return {
    key: { flex: '2' },
    value: { flex: '2' },
    type: { flex: '1', minWidth: '80px' },
  }
}

const colStyles = computed(() => getColumnStyles())

function startResize(colIndex: number, event: MouseEvent) {
  event.preventDefault()
  const startX = event.clientX
  const table = tableRef.value
  if (!table) {
    return
  }

  const headerCells = table.querySelectorAll('.tree-header > .col-key, .tree-header > .col-value, .tree-header > .col-type')
  const startKeyWidth = (headerCells[0] as HTMLElement).offsetWidth
  const startValueWidth = (headerCells[1] as HTMLElement).offsetWidth

  const onMouseMove = (e: MouseEvent) => {
    const delta = e.clientX - startX
    if (colIndex === 0) {
      const newKey = Math.max(60, startKeyWidth + delta)
      colKeyWidth.value = newKey
      if (colValueWidth.value === 0) {
        colValueWidth.value = startValueWidth
      }
    } else {
      const newValue = Math.max(60, startValueWidth + delta)
      colValueWidth.value = newValue
      if (colKeyWidth.value === 0) {
        colKeyWidth.value = startKeyWidth
      }
    }
  }

  const onMouseUp = () => {
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }

  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

</script>

<template>
  <div ref="tableRef" class="tree-table">
    <div class="tree-header">
      <span class="col-key" :style="colStyles.key">{{ t('query.key') }}</span>
      <span class="resize-handle" @mousedown="startResize(0, $event)" />
      <span class="col-value" :style="colStyles.value">{{ t('query.value') }}</span>
      <span class="resize-handle" @mousedown="startResize(1, $event)" />
      <span class="col-type" :style="colStyles.type">{{ t('query.type') }}</span>
    </div>
    <div class="tree-body">
      <div
        v-for="row in visibleRows"
        :key="row.key"
        class="tree-row"
        :class="{ clickable: row.hasChildren, 'doc-root': row.isDocRoot }"
        @click="toggleExpand(row)"
      >
        <span class="col-key" :style="{ ...colStyles.key, paddingLeft: row.depth * 16 + 4 + 'px' }">
          <span v-if="row.hasChildren" class="expand-icon">
            {{ row.expanded ? '▼' : '▶' }}
          </span>
          <span v-else class="expand-spacer" />
          <span class="field-name">{{ row.field }}</span>
        </span>
        <span class="col-value" :style="colStyles.value" :class="getTypeClass(row.type)">{{ row.value }}</span>
        <span class="col-type type-badge" :style="colStyles.type">{{ getTypeName(row.type) }}</span>
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

  > .col-key, > .col-value, > .col-type {
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
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 3px 4px;
  overflow: hidden;
  box-sizing: border-box;
}

.col-value {
  padding: 3px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  box-sizing: border-box;
}

.col-type {
  padding: 3px 8px;
  box-sizing: border-box;
}

.resize-handle {
  width: 5px;
  flex-shrink: 0;
  align-self: stretch;
  cursor: col-resize;
  background: linear-gradient(to right, transparent 2px, var(--n-text-color-3) 2px, var(--n-text-color-3) 3px, transparent 3px);

  &:hover {
    background: linear-gradient(to right, transparent 1px, var(--n-primary-color, #18a058) 1px, var(--n-primary-color, #18a058) 4px, transparent 4px);
  }
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
