<script setup lang="ts">
import * as systemProxy from 'wailsjs/go/api/SystemProxy'

const props = defineProps<{
  value?: string
  placeholder?: string
  disabled?: boolean
  defaultPath?: string
}>()

const emit = defineEmits<{
  (e: 'update:value', val: string | undefined ): void
}>()

const onInput = (val: string | undefined) => {
  emit('update:value', val)
}

const onClear = () => {
  emit('update:value', '')
}

const handleSaveFile = async () => {
  const result = await systemProxy.SaveFile(undefined, props.defaultPath, ['csv'])
}
</script>

<template>
  <n-input-group>
    <n-input
      :disabled="props.disabled"
      :placeholder="props.placeholder"
      :value="props.value"
      clearable
      @clear="onClear"
      @input="onInput"/>
    <n-button :disabled="props.disabled" :focusable="false" @click="handleSaveFile">...</n-button>

  </n-input-group>
</template>

<style scoped lang="scss"></style>
