<script setup lang="ts">
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import type { SelectOption } from 'naive-ui'

const props = defineProps<{ loading: boolean }>()

const settingsStore = useSettingsStore()
</script>

<template>
  <n-form :disabled="props.loading" :model="settingsStore.terminal" label-placement="top" :show-require-mark="false">
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
        <n-select v-model:value="settingsStore.terminal.font.family"
                  :options="settingsStore.fontOptions"
                  :placeholder="$t('settings.common.fontTip')"
                  :render-label="({ label, value }: SelectOption) => value || $t(label as string)"
                  filterable multiple tag />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')"
                      :span="24">
        <n-input-number v-model:value="settingsStore.terminal.font.size" :min="1" :max="65535" />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.terminal.cursorStyle')" :span="24">
        <n-radio-button v-for="opt in settingsStore.terminalCursorOptions"
                        :key="opt.value"
                        :value="opt.value">
          {{ $t(opt.label) }}
        </n-radio-button>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style scoped lang="scss"></style>
