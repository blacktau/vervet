<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useSettingsStore } from './settingsStore'

const { t } = useI18n()
const settingsStore = useSettingsStore()

const defaultExtensions = ['.js', '.mongodb']

function resetDefaults() {
  settingsStore.workspaces.fileExtensions = [...defaultExtensions]
}
</script>

<template>
  <div class="settings-section">
    <div class="setting-item">
      <label>{{ t('settings.workspaces.fileExtensions') }}</label>
      <p class="tip">{{ t('settings.workspaces.fileExtensionsTip') }}</p>
      <n-dynamic-tags v-model:value="settingsStore.workspaces.fileExtensions" />
      <n-button text type="primary" size="small" @click="resetDefaults">
        {{ t('settings.workspaces.resetDefaults') }}
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.settings-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.setting-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.setting-item label {
  font-weight: 500;
}

.tip {
  font-size: 12px;
  color: var(--n-text-color-3);
  margin: 0;
}
</style>
