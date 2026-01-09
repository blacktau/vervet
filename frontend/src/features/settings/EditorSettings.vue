<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'

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
          :options="settingsStore.monoFontOptions"
          :placeholder="$t('settings.common.fontTip')"
          filterable
          label-field="family"
          tag
          value-field="family" />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')" :span="24">
        <n-input-number v-model:value="settingsStore.editor.font.size" :max="65535" :min="1" />
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
        <n-checkbox v-model:checked="settingsStore.editor.lineNumbers">
          {{ $t('settings.editor.showLineNumbers') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
        <n-checkbox v-model:checked="settingsStore.editor.showFolding">
          {{ $t('settings.editor.showFolding') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
        <n-checkbox v-model:checked="settingsStore.editor.dropText">
          {{ $t('settings.editor.dropText') }}
        </n-checkbox>
      </n-form-item-gi>
      <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
        <n-checkbox v-model:checked="settingsStore.editor.links">
          {{ $t('settings.editor.links') }}
        </n-checkbox>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style lang="scss" scoped></style>
