<script setup lang="ts">

import { useSettingsStore } from '@/features/settings/settings.ts'

const props = defineProps<{ loading: boolean }>()

const settingsStore = useSettingsStore()

</script>

<template>
<n-form :disabled="props.loading" :model="settingsStore.editor" :show-require-mark="false" label-placement="top">
  <n-grid :x-gap="10">
    <n-form-item-gi :span="24" required>
      <template #label>
        {{ $t('settings.') }}
      </template>
    </n-form-item-gi>
  </n-grid>
</n-form>
</template>

<style scoped lang="scss"></style>
