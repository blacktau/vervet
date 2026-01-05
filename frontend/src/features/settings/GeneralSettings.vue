<script setup lang="ts">
import { QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import type { SelectOption } from 'naive-ui'

const props = defineProps<{
  loading: boolean
}>()

const settingsStore = useSettingsStore()

</script>

<template>
  <n-form
    :disabled="props.loading"
    :model="settingsStore.general"
    :show-require-mark="false"
    label-placement="top">
    <n-grid :x-gap="10">
      <n-form-item-gi :label="$t('settings.general.theme')" :span="24" required>
        <n-radio-group v-model:value="settingsStore.general.theme" name="theme" size="medium">
          <n-radio-button
            v-for="opt in settingsStore.themeOptions"
            :key="opt.value"
            :value="opt.value">
            {{ $t(opt.label) }}
          </n-radio-button>
        </n-radio-group>
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.general.language')" :span="24" required>
        <n-select
          v-model:value="settingsStore.general.language"
          :options="settingsStore.languageOptions"
          :render-label="({ label, value }: SelectOption) => (value === 'auto' ? $t(label as string) : label)"
          filterable />
      </n-form-item-gi>
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
          v-model:value="settingsStore.general.font.family"
          :options="settingsStore.fontOptions"
          :placeholder="$t('settings.common.fontTip')"
          :render-label="({ label, value }: SelectOption) => (value === '' ? $t(label as string) : label)"
          filterable
          multiple
          tag />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')" :span="24">
        <n-input-number v-model:value="settingsStore.general.font.size" :min="1" :max="65535" />
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style scoped lang="scss"></style>
