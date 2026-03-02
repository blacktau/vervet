<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
  selectedIndex: number
}>()

const emit = defineEmits<{
  select: [index: number]
}>()

const { t } = useI18n()

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getDocPreview(doc: any): string {
  if (typeof doc !== 'object' || doc === null) {
    return String(doc)
  }

  const parts: string[] = []
  for (const [key, val] of Object.entries(doc)) {
    if (key === '_id') {
      continue
    }
    if (parts.length >= 3) {
      break
    }
    if (val === null || typeof val === 'string' || typeof val === 'number' || typeof val === 'boolean') {
      parts.push(`${key}: ${JSON.stringify(val)}`)
    }
  }
  return parts.join(', ')
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function getDocId(doc: any): string {
  if (typeof doc !== 'object' || doc === null) {
    return ''
  }
  const id = doc._id
  if (id === undefined) {
    return ''
  }
  if (typeof id === 'object' && id !== null && '$oid' in id) {
    return id.$oid
  }
  return String(id)
}

const documentCount = computed(() => {
  return t('query.documentCount', { count: props.documents.length })
})
</script>

<template>
  <div class="document-list">
    <div class="document-list-header">{{ documentCount }}</div>
    <div class="document-list-items">
      <div
        v-for="(doc, index) in documents"
        :key="index"
        class="document-item"
        :class="{ selected: index === selectedIndex }"
        @click="emit('select', index)"
      >
        <span class="doc-index">{{ index + 1 }}</span>
        <span class="doc-content">
          <span v-if="getDocId(doc)" class="doc-id">{{ getDocId(doc) }}</span>
          <span class="doc-preview">{{ getDocPreview(doc) }}</span>
        </span>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.document-list {
  display: flex;
  flex-direction: column;
  min-width: 200px;
  max-width: 300px;
  border-right: 1px solid var(--n-border-color);
  overflow: hidden;
}

.document-list-header {
  padding: 6px 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--n-text-color-3);
  border-bottom: 1px solid var(--n-border-color);
  flex-shrink: 0;
}

.document-list-items {
  overflow-y: auto;
  flex: 1;
}

.document-item {
  display: flex;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  font-size: 12px;
  border-bottom: 1px solid var(--n-border-color);

  &:hover {
    background: var(--n-color-hover);
  }

  &.selected {
    background: var(--n-color-target);
  }
}

.doc-index {
  color: var(--n-text-color-3);
  flex-shrink: 0;
  min-width: 20px;
}

.doc-content {
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.doc-id {
  font-family: monospace;
  font-size: 11px;
  color: var(--n-text-color-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.doc-preview {
  font-size: 11px;
  color: var(--n-text-color-3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
