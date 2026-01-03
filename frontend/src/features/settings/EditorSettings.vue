<script setup lang="ts">
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'
import type { SelectOption } from 'naive-ui'

const props = defineProps<{ loading: boolean }>()

const settingsStore = useSettingsStore()
</script>

<template>
  <n-form
    :disabled="props.loading"
    :model="settingsStore.editor"
    :show-require-mark="false"
    label-placement="top">
    <n-grid :x-gap="10">
      <n-form-item-gi :span="24" required>
        <template #label>
          {{ $t('settings.common.font') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="QuestionMarkCircleIcon" />
            </template>
            <div class="text-block">
              {{ $t('settings.common.fontTip') }}
            </div>
          </n-tooltip>
        </template>
        <n-select
          v-model:value="settingsStore.editor.font.family"
          :options="settingsStore.fontOptions"
          :placeholder="$t('settings.common.fontTip')"
          :render-label="({ label, value }: SelectOption) => value || $t(label as string)"
          filterable
          multiple
          tag />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')" :span="24">
        <n-input-number v-model:value="settingsStore.editor.font.size" :min="1" :max="65535" />
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :span="24" :show-label="false">
        <n-checkbox v-model:checked="settingsStore.editor.lineNumbers">
          {{ $t('settings.editor.showLineNumbers') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :span="24" :show-label="false">
        <n-checkbox v-model:checked="settingsStore.editor.showFolding">
          {{ $t('settings.editor.showFolding') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :span="24" :show-label="false">
        <n-checkbox v-model:checked="settingsStore.editor.dropText">
          {{ $t('settings.editor.dropText') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :span="24" :show-label="false">
        <n-checkbox v-model:checked="settingsStore.editor.links">
          {{ $t('settings.editor.links') }}
        </n-checkbox>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style scoped lang="scss"></style>
