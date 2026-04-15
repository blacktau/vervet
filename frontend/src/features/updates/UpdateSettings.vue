<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { useUpdateStore } from './updateStore'
import { GetAppVersion } from 'wailsjs/go/api/SettingsProxy'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const updates = useUpdateStore()

const currentVersion = ref('')

onMounted(async () => {
  const r = await GetAppVersion()
  if (r.isSuccess) {
    currentVersion.value = r.data as string
  }
})

const frequencyOptions = computed(() => [
  { label: t('settings.updates.frequencyOptions.never'), value: 'never' },
  { label: t('settings.updates.frequencyOptions.startup'), value: 'startup' },
  { label: t('settings.updates.frequencyOptions.daily'), value: 'daily' },
  { label: t('settings.updates.frequencyOptions.weekly'), value: 'weekly' },
])

const frequency = computed({
  get: () => {
    const u = (settingsStore as unknown as { updates?: { frequency?: string } }).updates
    return u?.frequency ?? 'daily'
  },
  set: (v: string) => {
    const s = settingsStore as unknown as { updates: { frequency: string; lastCheckedAt?: string; dismissedVersion?: string } }
    if (!s.updates) {
      s.updates = { frequency: v }
      return
    }
    s.updates.frequency = v
  },
})

const lastCheckedLabel = computed(() => {
  const u = (settingsStore as unknown as { updates?: { lastCheckedAt?: string } }).updates
  const ts = u?.lastCheckedAt
  if (!ts) {
    return t('settings.updates.lastCheckedNever')
  }
  const d = new Date(ts)
  if (isNaN(d.getTime())) {
    return t('settings.updates.lastCheckedNever')
  }
  return d.toLocaleString()
})

const hasCheckedBefore = computed(() => {
  const u = (settingsStore as unknown as { updates?: { lastCheckedAt?: string } }).updates
  return Boolean(u?.lastCheckedAt)
})
</script>

<template>
  <n-form label-placement="left" label-width="auto">
    <n-form-item :label="$t('settings.updates.frequency')">
      <n-select
        v-model:value="frequency"
        :options="frequencyOptions"
        style="max-width: 240px" />
    </n-form-item>
    <n-form-item :show-label="false">
      <n-space vertical :size="12">
        <span>{{ $t('settings.updates.currentVersion', { version: currentVersion }) }}</span>
        <span>{{ $t('settings.updates.lastChecked', { time: lastCheckedLabel }) }}</span>
        <n-space>
          <n-button :loading="updates.checking" @click="updates.checkNow">
            {{ updates.checking ? $t('settings.updates.checking') : $t('settings.updates.checkNow') }}
          </n-button>
          <n-button v-if="updates.available" type="primary" @click="updates.openReleasePage">
            {{ $t('settings.updates.viewRelease') }}
          </n-button>
          <n-button v-if="updates.available" @click="updates.dismiss">
            {{ $t('settings.updates.dismiss') }}
          </n-button>
        </n-space>
        <n-alert v-if="updates.available" type="info" :show-icon="false">
          {{ $t('settings.updates.available', { version: updates.version }) }}
        </n-alert>
        <n-alert v-else-if="hasCheckedBefore" type="success" :show-icon="false">
          {{ $t('settings.updates.upToDate') }}
        </n-alert>
        <n-alert v-if="updates.lastError" type="error" :show-icon="false">
          {{ $t('settings.updates.checkFailed', { error: updates.lastError }) }}
        </n-alert>
      </n-space>
    </n-form-item>
  </n-form>
</template>
