<script setup lang="ts">
const props = defineProps<{
  isEdit?: boolean,
  value?: string | number,
}>()

const emit = defineEmits<{
  (e: 'update:value', value: string | [string, string]): void
}>()

const handleUpdateValue = (val: string | [string, string]) => {
  emit('update:value', val)
}
</script>

<template>
  <div style="min-height: 22px">
    <template v-if="props.isEdit">
      <n-input :value="props.value" @update:value="handleUpdateValue" />
    </template>
    <template v-else>
      {{ props.value }}
    </template>
  </div>
</template>

<style scoped lang="scss"></style>
