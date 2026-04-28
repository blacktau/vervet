<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore.ts'
import { QuestionMarkCircleIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{ loading: boolean }>()

const settingsStore = useSettingsStore()

const pageSizeOptions = [
  { label: '25', value: 25 },
  { label: '50', value: 50 },
  { label: '100', value: 100 },
  { label: '200', value: 200 },
  { label: '500', value: 500 },
]
</script>

<template>
  <n-form
    :disabled="props.loading"
    :model="settingsStore.query"
    :show-require-mark="false"
    label-placement="top">
    <n-grid :x-gap="10">
      <n-form-item-gi :span="24">
        <template #label>
          {{ $t('settings.query.defaultLimit') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="QuestionMarkCircleIcon" />
            </template>
            <div class="text-block">
              {{ $t('settings.query.defaultLimitHelp') }}
            </div>
          </n-tooltip>
        </template>
        <n-input-number
          v-model:value="settingsStore.query.defaultLimit"
          :max="10000"
          :min="1" />
      </n-form-item-gi>
      <n-form-item-gi :span="24">
        <template #label>
          {{ $t('settings.query.defaultPageSize') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="QuestionMarkCircleIcon" />
            </template>
            <div class="text-block">
              {{ $t('settings.query.defaultPageSizeHelp') }}
            </div>
          </n-tooltip>
        </template>
        <n-select
          v-model:value="settingsStore.query.defaultPageSize"
          :options="pageSizeOptions" />
      </n-form-item-gi>
      <n-form-item-gi :span="24">
        <template #label>
          {{ $t('settings.query.queryEngine') }}
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-icon :component="QuestionMarkCircleIcon" />
            </template>
            <div class="text-block">
              {{ $t('settings.query.queryEngineHelp') }}
            </div>
          </n-tooltip>
        </template>
        <n-radio-group v-model:value="settingsStore.query.queryEngine" name="queryEngine" size="medium">
          <n-radio-button value="builtin">
            {{ $t('settings.query.queryEngineBuiltin') }}
          </n-radio-button>
          <n-radio-button value="mongosh">
            {{ $t('settings.query.queryEngineMongosh') }}
          </n-radio-button>
        </n-radio-group>
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

<style lang="scss" scoped></style>
