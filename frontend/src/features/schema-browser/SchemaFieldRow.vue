<script setup lang="ts">
import { computed } from 'vue'
import type { models } from 'wailsjs/go/models'
import { DocumentDuplicateIcon, PlusIcon } from '@heroicons/vue/24/outline'
import TypeBar from './TypeBar.vue'

const props = defineProps<{
  field: models.FieldInfo
  sampledCount: number
  depth: number
  expandable: boolean
  expanded: boolean
}>()

const emit = defineEmits<{
  'copy-path': [path: string]
  'create-index': [path: string]
  toggle: [path: string]
}>()

const percent = computed(() =>
  props.sampledCount > 0
    ? ((props.field.count / props.sampledCount) * 100).toFixed(1)
    : '0',
)

function onCopy(e: Event) {
  e.stopPropagation()
  emit('copy-path', props.field.path)
}

function onCreateIndex(e: Event) {
  e.stopPropagation()
  emit('create-index', props.field.path)
}
</script>

<template>
  <div class="schema-row" :style="{ paddingLeft: depth * 16 + 'px' }" @click="expandable && emit('toggle', field.path)">
    <span class="schema-row__caret">
      <template v-if="expandable">{{ expanded ? '▾' : '▸' }}</template>
    </span>
    <span class="schema-row__name">{{ field.name }}</span>
    <span class="schema-row__bar">
      <TypeBar :types="field.types" :total="field.count" />
    </span>
    <span class="schema-row__pct">{{ percent }}%</span>
    <span class="schema-row__actions">
      <NButton
        data-test="copy-path"
        size="tiny"
        quaternary
        :title="$t('schemaBrowser.copyPath')"
        @click="onCopy"
      >
        <template #icon>
          <DocumentDuplicateIcon class="action-icon" />
        </template>
      </NButton>
      <NButton
        data-test="create-index"
        size="tiny"
        quaternary
        :title="$t('schemaBrowser.createIndex')"
        @click="onCreateIndex"
      >
        <template #icon>
          <PlusIcon class="action-icon" />
        </template>
      </NButton>
    </span>
  </div>
</template>

<style scoped>
.schema-row {
  display: grid;
  grid-template-columns: 16px 1fr 200px 60px auto;
  gap: 8px;
  align-items: center;
  padding: 4px 8px;
  cursor: pointer;
}
.schema-row:hover {
  background: rgba(127, 127, 127, 0.08);
}
.schema-row__caret {
  width: 16px;
  text-align: center;
  user-select: none;
  font-size: 10px;
  opacity: 0.6;
}
.schema-row__name {
  font-family: var(--font-mono, monospace);
  font-size: 13px;
}
.schema-row__pct {
  font-variant-numeric: tabular-nums;
  text-align: right;
  font-size: 12px;
  opacity: 0.75;
}
.schema-row__actions {
  display: flex;
  gap: 2px;
}
.action-icon {
  width: 14px;
  height: 14px;
}
</style>
