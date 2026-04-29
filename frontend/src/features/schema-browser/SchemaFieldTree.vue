<script setup lang="ts">
import { ref } from 'vue'
import type { models } from 'wailsjs/go/models'
import SchemaFieldRow from './SchemaFieldRow.vue'

defineProps<{
  fields: models.FieldInfo[]
  sampledCount: number
  depth?: number
}>()

const emit = defineEmits<{
  'copy-path': [path: string]
  'create-index': [path: string]
}>()

const expanded = ref<Set<string>>(new Set())

function toggle(path: string) {
  if (expanded.value.has(path)) {
    expanded.value.delete(path)
  } else {
    expanded.value.add(path)
  }
}
</script>

<template>
  <div class="schema-tree">
    <template v-for="f in fields" :key="f.path">
      <SchemaFieldRow
        :field="f"
        :sampled-count="sampledCount"
        :depth="depth ?? 0"
        :expandable="(f.children?.length ?? 0) > 0"
        :expanded="expanded.has(f.path)"
        @copy-path="(p) => emit('copy-path', p)"
        @create-index="(p) => emit('create-index', p)"
        @toggle="toggle"
      />
      <SchemaFieldTree
        v-if="(f.children?.length ?? 0) > 0 && expanded.has(f.path)"
        :fields="f.children"
        :sampled-count="sampledCount"
        :depth="(depth ?? 0) + 1"
        @copy-path="(p) => emit('copy-path', p)"
        @create-index="(p) => emit('create-index', p)"
      />
    </template>
  </div>
</template>
