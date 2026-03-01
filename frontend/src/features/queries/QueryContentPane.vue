<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { useI18n } from 'vue-i18n'
import QueryTab from './QueryTab.vue'
import { computed } from 'vue'

const tabStore = useTabStore()
const queryStore = useQueryStore()
const { t } = useI18n()

const activeQueryId = computed({
  get: () => tabStore.currentTab?.activeQueryId ?? '',
  set: (val: string) => tabStore.setActiveQuery(val),
})

function handleClose(queryId: string) {
  const serverId = tabStore.currentTabId
  if (!serverId) {
    return
  }
  queryStore.removeQueryState(queryId)
  tabStore.closeQuery(serverId, queryId)
}
</script>

<template>
  <div class="content-container flex-box-v" style="margin-right: 5px">
    <n-tabs
      v-if="tabStore.currentTab?.queryOpen && tabStore.currentQueries.length > 0"
      v-model:value="activeQueryId"
      type="card"
      closable
      @close="handleClose">
      <n-tab-pane
        v-for="query in tabStore.currentQueries"
        :key="query.id"
        :name="query.id"
        :tab="tabStore.queryTabLabel(tabStore.currentTab!, query)"
        display-directive="show:lazy">
        <QueryTab :query-id="query.id" />
      </n-tab-pane>
    </n-tabs>
    <div v-else class="empty-state">
      <n-empty :description="t('query.emptyState')" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}
</style>
