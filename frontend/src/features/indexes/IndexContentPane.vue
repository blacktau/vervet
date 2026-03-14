<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs'
import { useI18n } from 'vue-i18n'
import IndexTab from './IndexTab.vue'
import { computed } from 'vue'

const tabStore = useTabStore()
const { t } = useI18n()

const activeIndexTabId = computed({
  get: () => tabStore.currentTab?.activeIndexTabId ?? '',
  set: (val: string) => tabStore.setActiveIndexTab(val),
})

function handleClose(indexTabId: string) {
  const serverId = tabStore.currentTabId
  if (!serverId) {
    return
  }
  tabStore.closeIndexTab(serverId, indexTabId)
}
</script>

<template>
  <div class="index-content-container flex-box-v">
    <n-tabs
      v-if="tabStore.currentTab && (tabStore.currentTab.indexTabs?.length ?? 0) > 0"
      v-model:value="activeIndexTabId"
      type="card"
      closable
      @close="handleClose">
      <n-tab-pane
        v-for="indexTab in tabStore.currentTab.indexTabs ?? []"
        :key="indexTab.id"
        :name="indexTab.id"
        :tab="tabStore.indexTabLabel(tabStore.currentTab!, indexTab)"
        display-directive="show:lazy">
        <IndexTab
          :server-id="indexTab.serverId"
          :db-name="indexTab.dbName"
          :collection-name="indexTab.collectionName" />
      </n-tab-pane>
    </n-tabs>
    <div v-else class="empty-state">
      <n-empty :description="t('indexes.emptyState')" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.index-content-container {
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
