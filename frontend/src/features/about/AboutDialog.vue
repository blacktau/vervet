<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useDialogStore } from '@/stores/dialog.ts'
import { onMounted, ref } from 'vue'
import * as settingsProxy from 'wailsjs/go/api/SettingsProxy'
import * as runtime from 'wailsjs/runtime'
import iconUrl from '@/assets/logo.svg'

const themeVars = useThemeVars()
const dialogStore = useDialogStore()
const version = ref('')

onMounted(async () => {
  const result = await settingsProxy.GetAppVersion()
  if (result.isSuccess) {
    version.value = result.data
  }
})

const onOpenSource = () => {
  runtime.BrowserOpenURL('https://github.com/blacktau/vervet')
}

</script>

<template>
  <n-modal v-model:show="dialogStore.aboutDialogVisible" :show-icon="false" preset="dialog" transform-origin="center">
    <n-space :size="10" :wrap="false" align="center" vertical>
      <n-avatar :size="120" :src="iconUrl" color="#0000"></n-avatar>
      <div class="about-app-title">Vervet</div>
      <n-text>{{ version }}</n-text>
      <n-space :size="5" :wrap="false" :wrap-item="false" align="center">
        <n-text class="about-link" @click="onOpenSource">{{ $t('dialog.about.source') }}</n-text>
      </n-space>
      <div :style="{ color: themeVars.textColor3 }" class="about-copyright">
        Copyright © 2025 Sean Garrett All rights reserved
      </div>
    </n-space>
  </n-modal>
</template>

<style scoped lang="scss">
.about-app-title {
  font-weight: bold;
  font-size: 18px;
  margin: 5px;
}

.about-link {
  cursor: pointer;

  &:hover {
    text-decoration: underline;
  }
}

.about-copyright {
  font-size: 12px;
}
</style>
