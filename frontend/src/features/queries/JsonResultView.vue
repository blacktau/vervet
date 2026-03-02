<script lang="ts" setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import { useSettingsStore } from '@/features/settings/settingsStore'
import * as monaco from 'monaco-editor'

const props = defineProps<{
  content: string
}>()

const settingsStore = useSettingsStore()
const container = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneCodeEditor | null = null

onMounted(() => {
  if (!container.value) {
    return
  }

  editor = monaco.editor.create(container.value, {
    value: props.content,
    language: 'json',
    theme: settingsStore.isDark ? 'vervet-dark' : 'vervet-light',
    readOnly: true,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 13,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    padding: { top: 8 },
    folding: true,
    domReadOnly: true,
  })
})

watch(() => props.content, (newContent) => {
  if (editor) {
    editor.setValue(newContent)
  }
})

watch(() => settingsStore.isDark, (isDark) => {
  if (editor) {
    monaco.editor.setTheme(isDark ? 'vervet-dark' : 'vervet-light')
  }
})

onBeforeUnmount(() => {
  if (editor) {
    editor.dispose()
    editor = null
  }
})
</script>

<template>
  <div ref="container" class="json-view" />
</template>

<style lang="scss" scoped>
.json-view {
  height: 100%;
  width: 100%;
}
</style>
