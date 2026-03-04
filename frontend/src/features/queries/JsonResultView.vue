<script lang="ts" setup>
import { watch } from 'vue'
import { useMonacoEditor } from './useMonacoEditor'

const props = defineProps<{
  content: string
}>()

const { container, editor } = useMonacoEditor({
  language: 'json',
  value: props.content,
  readOnly: true,
  fontSize: 13,
  extraOptions: {
    folding: true,
    domReadOnly: true,
  },
})

watch(
  () => props.content,
  (newContent) => {
    if (editor.value) {
      editor.value.setValue(newContent)
    }
  },
)
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
