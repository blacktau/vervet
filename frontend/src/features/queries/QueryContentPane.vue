<script lang="ts" setup>
import { BrowserSubTabType } from '@/consts/BrowserSubTabType'
import { useTabStore } from '@/features/tabs/tabs'
import { useI18n } from 'vue-i18n'
import QueryTab from './QueryTab.vue'
import { CodeBracketSquareIcon } from '@heroicons/vue/24/outline'

const tabStore = useTabStore()
const { t } = useI18n()
</script>

<template>
  <div class="content-container flex-box-v" style="margin-right: 5px">
    <n-tabs
      :value="tabStore.currentSubTab"
      type="line"
      animated
      @update:value="tabStore.switchSubTab">
      <n-tab-pane :name="BrowserSubTabType.Query" display-directive="show:lazy">
        <template #tab>
          <n-icon :component="CodeBracketSquareIcon" style="margin-right: 4px" />
          {{ t('dataBrowser.subTab.query') }}
        </template>
        <QueryTab />
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<style lang="scss" scoped></style>
