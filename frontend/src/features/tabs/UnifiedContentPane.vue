<script lang="ts" setup>
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { type QueryState } from '@/features/queries/queryStore'
import { useI18n } from 'vue-i18n'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ChevronDownIcon } from '@heroicons/vue/24/outline'
import type { DropdownOption } from 'naive-ui'
import QueryTab from '@/features/queries/QueryTab.vue'
import IndexTab from '@/features/indexes/IndexTab.vue'
import CollectionStatisticsTab from '@/features/statistics/CollectionStatisticsTab.vue'
import DatabaseStatisticsTab from '@/features/statistics/DatabaseStatisticsTab.vue'
import ServerStatisticsTab from '@/features/statistics/ServerStatisticsTab.vue'
import SchemaBrowserPane from '@/features/schema-browser/SchemaBrowserPane.vue'
import { useTabSortable } from '@/features/tabs/useTabSortable'

type UnifiedTab = {
  id: string
  label: string
  type: 'query' | 'index' | 'statistics' | 'schema'
}

const tabStore = useTabStore()
const queryStore = useQueryStore()

function isQueryTabLoading(id: string): boolean {
  if (!id.startsWith('query-')) {
    return false
  }
  return queryStore.queries[id]?.loading ?? false
}
const { t } = useI18n()
const dialog = useDialog()

const activeInnerTabId = computed({
  get: () => tabStore.currentTab?.activeInnerTabId ?? '',
  set: (val: string) => tabStore.setActiveInnerTab(val),
})

const unifiedTabs = computed<UnifiedTab[]>(() => {
  const tab = tabStore.currentTab
  if (!tab) {
    return []
  }
  const queryById = new Map(tab.queries.map((q) => [q.id, q] as const))
  const indexById = new Map((tab.indexTabs ?? []).map((i) => [i.id, i] as const))
  const statsById = new Map((tab.statisticsTabs ?? []).map((s) => [s.id, s] as const))
  const schemaById = new Map((tab.schemaTabs ?? []).map((s) => [s.id, s] as const))

  const result: UnifiedTab[] = []
  for (const id of tab.innerTabOrder) {
    const q = queryById.get(id)
    if (q) {
      result.push({ id, label: tabStore.queryTabLabel(tab, q), type: 'query' })
      continue
    }
    const i = indexById.get(id)
    if (i) {
      result.push({ id, label: tabStore.indexTabLabel(tab, i), type: 'index' })
      continue
    }
    const s = statsById.get(id)
    if (s) {
      result.push({ id, label: tabStore.statisticsTabLabel(tab, s), type: 'statistics' })
      continue
    }
    const sc = schemaById.get(id)
    if (sc) {
      result.push({ id, label: tabStore.schemaTabLabel(tab, sc), type: 'schema' })
      continue
    }
  }
  return result
})

const hasInnerTabs = computed(() => unifiedTabs.value.length > 0)

const innerTabsRef = ref<HTMLElement | null>(null)
const isOverflowing = ref(false)
let resizeObserver: ResizeObserver | null = null

function checkOverflow() {
  const el = innerTabsRef.value?.$el ?? innerTabsRef.value
  if (!el) {
    return
  }
  const scrollWrapper = el.querySelector('.n-tabs-nav-scroll-content')
  const scrollContainer = el.querySelector('.n-tabs-nav-scroll-wrapper')
  if (scrollWrapper && scrollContainer) {
    isOverflowing.value = scrollWrapper.scrollWidth > scrollContainer.clientWidth
  }
}

onMounted(() => {
  resizeObserver = new ResizeObserver(checkOverflow)
  const el = innerTabsRef.value?.$el ?? innerTabsRef.value
  if (el) {
    resizeObserver.observe(el)
  }
})

onBeforeUnmount(() => {
  resizeObserver?.disconnect()
})

watch(() => unifiedTabs.value.length, () => nextTick(checkOverflow))

useTabSortable(innerTabsRef, '.n-tabs-wrapper', (from, to) => {
  const serverId = tabStore.currentTab?.serverId
  if (!serverId) {
    return
  }
  tabStore.reorderInnerTabs(serverId, from, to)
})

const tabDropdownOptions = computed<DropdownOption[]>(() => {
  return unifiedTabs.value.map((tab) => ({
    label: tab.label,
    key: tab.id,
  }))
})

function handleTabSelect(key: string) {
  activeInnerTabId.value = key
}

const contextMenuShown = ref(false)
const contextMenuX = ref(0)
const contextMenuY = ref(0)
const contextMenuTargetId = ref<string | null>(null)
const contextMenuTargetType = ref<UnifiedTab['type'] | null>(null)

const contextMenuOptions = computed<DropdownOption[]>(() => {
  if (contextMenuTargetType.value !== 'query') {
    return []
  }
  return [
    { label: t('query.tabContextMenu.duplicate'), key: 'duplicate' },
  ]
})

function onTabContextMenu(e: MouseEvent, uTab: UnifiedTab) {
  e.preventDefault()
  if (uTab.type !== 'query') {
    contextMenuShown.value = false
    return
  }
  contextMenuTargetId.value = uTab.id
  contextMenuTargetType.value = uTab.type
  contextMenuX.value = e.clientX
  contextMenuY.value = e.clientY
  contextMenuShown.value = false
  nextTick(() => {
    contextMenuShown.value = true
  })
}

function onContextMenuSelect(key: string) {
  const targetId = contextMenuTargetId.value
  contextMenuShown.value = false
  if (!targetId) {
    return
  }
  const serverId = tabStore.currentTab?.serverId
  if (!serverId) {
    return
  }
  if (key === 'duplicate') {
    tabStore.duplicateQuery(serverId, targetId)
  }
}

function onContextMenuClickOutside() {
  contextMenuShown.value = false
}

function findIndexTabById(id: string) {
  return (tabStore.currentTab?.indexTabs ?? []).find((t) => t.id === id)
}

function findStatsTabById(id: string) {
  return (tabStore.currentTab?.statisticsTabs ?? []).find((t) => t.id === id)
}

function findSchemaTabById(id: string) {
  return (tabStore.currentTab?.schemaTabs ?? []).find((t) => t.id === id)
}

async function handleClose(innerTabId: string) {
  const serverId = tabStore.currentTab?.serverId
  if (!serverId) {
    return
  }

  if (innerTabId.startsWith('query-')) {
    const state = queryStore.getQueryState(innerTabId)
    if (state.isDirty) {
      const shouldClose = await promptSaveBeforeClose(innerTabId, state)
      if (!shouldClose) {
        return
      }
    }
    queryStore.removeQueryState(innerTabId)
    tabStore.closeQuery(serverId, innerTabId)
  } else if (innerTabId.startsWith('index-')) {
    tabStore.closeIndexTab(serverId, innerTabId)
  } else if (innerTabId.startsWith('stats-')) {
    tabStore.closeStatisticsTab(serverId, innerTabId)
  } else if (innerTabId.startsWith('schema-')) {
    tabStore.closeSchemaTab(serverId, innerTabId)
  }
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
      v-if="hasInnerTabs"
      ref="innerTabsRef"
      v-model:value="activeInnerTabId"
      type="card"
      closable
      @close="handleClose">
      <template v-if="isOverflowing" #suffix>
        <n-dropdown
          :options="tabDropdownOptions"
          trigger="click"
          @select="handleTabSelect">
          <n-button quaternary size="small" style="margin-right: 4px">
            <template #icon>
              <n-icon>
                <ChevronDownIcon />
              </n-icon>
            </template>
          </n-button>
        </n-dropdown>
      </template>
      <n-tab-pane
        v-for="uTab in unifiedTabs"
        :key="uTab.id"
        :name="uTab.id"
        :tab="uTab.label"
        display-directive="show:lazy">
        <template #tab>
          <span class="inner-tab-label" @contextmenu="onTabContextMenu($event, uTab)">
            <n-spin
              v-if="uTab.type === 'query' && isQueryTabLoading(uTab.id)"
              :size="12"
              class="inner-tab-spinner" />
            {{ uTab.label }}
          </span>
        </template>
        <!-- Query content -->
        <QueryTab v-if="uTab.type === 'query'" :query-id="uTab.id" />

        <!-- Index content -->
        <template v-else-if="uTab.type === 'index'">
          <IndexTab
            v-if="findIndexTabById(uTab.id)"
            :server-id="findIndexTabById(uTab.id)!.serverId"
            :db-name="findIndexTabById(uTab.id)!.dbName"
            :collection-name="findIndexTabById(uTab.id)!.collectionName" />
        </template>

        <!-- Statistics content -->
        <template v-else-if="uTab.type === 'statistics'">
          <CollectionStatisticsTab
            v-if="findStatsTabById(uTab.id)?.level === 'collection'"
            :server-id="findStatsTabById(uTab.id)!.serverId"
            :db-name="findStatsTabById(uTab.id)!.dbName"
            :collection-name="findStatsTabById(uTab.id)!.collectionName" />
          <DatabaseStatisticsTab
            v-else-if="findStatsTabById(uTab.id)?.level === 'database'"
            :server-id="findStatsTabById(uTab.id)!.serverId"
            :db-name="findStatsTabById(uTab.id)!.dbName" />
          <ServerStatisticsTab
            v-else-if="findStatsTabById(uTab.id)?.level === 'server'"
            :server-id="findStatsTabById(uTab.id)!.serverId" />
        </template>

        <!-- Schema content -->
        <template v-else-if="uTab.type === 'schema'">
          <SchemaBrowserPane
            v-if="findSchemaTabById(uTab.id)"
            :tab-id="uTab.id"
            :server-id="findSchemaTabById(uTab.id)!.serverId"
            :db-name="findSchemaTabById(uTab.id)!.dbName"
            :collection-name="findSchemaTabById(uTab.id)!.collectionName" />
        </template>
      </n-tab-pane>
    </n-tabs>
    <div v-else class="empty-state">
      <n-empty :description="t('query.emptyState')" />
    </div>
    <n-dropdown
      trigger="manual"
      placement="bottom-start"
      :show="contextMenuShown"
      :x="contextMenuX"
      :y="contextMenuY"
      :options="contextMenuOptions"
      @select="onContextMenuSelect"
      @clickoutside="onContextMenuClickOutside" />
  </div>
</template>

<style lang="scss" scoped>
.content-container {
  min-width: 0;
  overflow: hidden;

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

.inner-tab-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.inner-tab-spinner {
  flex: 0 0 auto;
}

:deep(.tab-sortable-fallback) {
  opacity: 0 !important;
}
:deep(.tab-sortable-ghost) {
  opacity: 0.4;
}
</style>
