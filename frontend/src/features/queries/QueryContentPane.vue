<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { type QueryState } from '@/features/queries/queryStore'
import { useI18n } from 'vue-i18n'
import QueryTab from './QueryTab.vue'
import { computed } from 'vue'

const tabStore = useTabStore()
const queryStore = useQueryStore()
const { t } = useI18n()
const dialog = useDialog()

const activeQueryId = computed({
  get: () => tabStore.currentTab?.activeQueryId ?? '',
  set: (val: string) => tabStore.setActiveQuery(val),
})

async function handleClose(queryId: string) {
  const state = queryStore.getQueryState(queryId)
  if (state.isDirty) {
    const shouldClose = await promptSaveBeforeClose(queryId, state)
    if (!shouldClose) {
      return
    }
  }

  const serverId = tabStore.currentTab?.serverId
  if (!serverId) {
    return
  }
  queryStore.removeQueryState(queryId)
  tabStore.closeQuery(serverId, queryId)
}

async function promptSaveBeforeClose(queryId: string, state: QueryState): Promise<boolean> {
  const filename = state.filePath?.split('/').pop() ?? 'Untitled'
  return new Promise((resolve) => {
    dialog.warning({
      title: t('query.unsavedChangesTitle'),
      content: t('query.unsavedChangesMessage', { filename }),
      positiveText: t('query.unsavedChangesSave'),
      negativeText: t('query.unsavedChangesDontSave'),
      onPositiveClick: async () => {
        const saved = await queryStore.saveFile(queryId, state.currentContent)
        if (!saved) {
          resolve(false)
          return
        }
        resolve(true)
      },
      onNegativeClick: () => {
        resolve(true)
      },
      onClose: () => {
        resolve(false)
      },
    })
  })
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
.content-container {
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
