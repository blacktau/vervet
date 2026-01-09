<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import type { SelectOption } from 'naive-ui'

const props = defineProps<{ loading: boolean }>()

const settingsStore = useSettingsStore()
</script>

<template>
  <n-form
    :disabled="props.loading"
    :model="settingsStore.terminal"
    :show-require-mark="false"
    label-placement="top">
    <n-grid :x-gap="10">
      <n-form-item-gi :span="24" required>
        <template #label>
          {{ $t('settings.common.font') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon component="QuestionMarkCircleIcon" />
            </template>
            <div class="text-block">
              {{ $t('settings.common.fontTip') }}
            </div>
          </n-tooltip>
        </template>
        <n-select
          v-model:value="settingsStore.terminal.font.family"
          :options="settingsStore.monoFontOptions"
          :placeholder="$t('settings.common.fontTip')"
          :render-label="({ label, value }: SelectOption) => value || $t(label as string)"
          filterable
          multiple
          tag />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')" :span="24">
        <n-input-number v-model:value="settingsStore.terminal.font.size" :max="65535" :min="1" />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.terminal.cursorStyle')" :span="24">
        <n-radio-group
          v-model:value="settingsStore.terminal.cursorStyle"
          name="theme"
          size="medium">
          <n-radio-button
            v-for="opt in settingsStore.terminalCursorOptions"
            :key="opt.value"
            :value="opt.value">
            {{ $t(opt.label) }}
          </n-radio-button>
        </n-radio-group>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style lang="scss" scoped></style>
