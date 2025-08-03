<script setup lang="ts">
import * as systemProxy from 'wailsjs/go/api/SystemProxy'
import { isEmpty } from 'lodash'
const props = defineProps<{
  value?: string
  placeHolder?: string
  disabled?: boolean
  ext?: string
}>()

const emit = defineEmits<{
  (e: 'update:value', value: string | [string, string]): void
}>()

const onInput = (val: string | [string, string]) => {
  emit('update:value', val)
}

const onClear = () => {
  emit('update:value', '')
}

const handleSelectFile = async () => {
  const result = await systemProxy.SelectFile('', isEmpty(props.ext) ? undefined : [props.ext])
  if (result.isSuccess) {
    const path = result.data ?? ''
    emit('update:value', path)
  }
}
</script>

<template>
  <n-input-group>
   <n-input
     :disabled="props.disabled"
     :placeholder="props.placeHolder"
     :title="props.value"
     :value="props.value"
     clearable
     @clear="onClear"
     @input="onInput" />
    <n-button
      :disabled="props.disabled"
      :focusable="false"
      @click="handleSelectFile">...</n-button>
  </n-input-group>
</template>

<style scoped lang="scss"></style>
