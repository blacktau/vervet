<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs'
import { useI18n } from 'vue-i18n'
import CollectionStatisticsTab from './CollectionStatisticsTab.vue'
import { computed } from 'vue'

const tabStore = useTabStore()
const { t } = useI18n()

const activeStatisticsTabId = computed({
  get: () => tabStore.currentTab?.activeStatisticsTabId ?? '',
  set: (val: string) => tabStore.setActiveStatisticsTab(val),
})

function handleClose(statisticsTabId: string) {
  const serverId = tabStore.currentTabId
  if (!serverId) {
    return
  }
  tabStore.closeStatisticsTab(serverId, statisticsTabId)
}
</script>

<template>
  <div class="statistics-content-container flex-box-v">
    <n-tabs
      v-if="tabStore.currentTab && (tabStore.currentTab.statisticsTabs?.length ?? 0) > 0"
      v-model:value="activeStatisticsTabId"
      type="card"
      closable
      @close="handleClose">
      <n-tab-pane
        v-for="statsTab in tabStore.currentTab.statisticsTabs ?? []"
        :key="statsTab.id"
        :name="statsTab.id"
        :tab="tabStore.statisticsTabLabel(tabStore.currentTab!, statsTab)"
        display-directive="show:lazy">
        <CollectionStatisticsTab
          v-if="statsTab.level === 'collection'"
          :server-id="statsTab.serverId"
          :db-name="statsTab.dbName"
          :collection-name="statsTab.collectionName" />
      </n-tab-pane>
    </n-tabs>
    <div v-else class="empty-state">
      <n-empty :description="t('statistics.emptyState')" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.statistics-content-container {
  :deep(.n-tabs) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  :deep(.n-tabs .n-tab-pane) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  :deep(.n-tabs .n-tabs-pane-wrapper) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
</style>
