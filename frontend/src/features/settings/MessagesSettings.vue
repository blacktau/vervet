<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/features/settings/settingsStore.ts'

const props = defineProps<{ loading: boolean }>()

const { t } = useI18n()
const settingsStore = useSettingsStore()

const fontOptions = computed(() => [
  { label: t('settings.common.defaultFont'), value: '' },
  ...settingsStore.monoFontOptions.map((f) => ({ label: f.family, value: f.family })),
])
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
          :options="fontOptions"
          :placeholder="$t('settings.common.defaultFont')"
          filterable
          tag />
      </n-form-item-gi>
      <n-form-item-gi :label="$t('settings.common.fontSize')" :span="24">
        <n-input-number v-model:value="settingsStore.terminal.font.size" :max="65535" :min="1" />
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style lang="scss" scoped></style>
