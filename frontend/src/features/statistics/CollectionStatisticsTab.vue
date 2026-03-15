<script lang="ts" setup>
import { onMounted, ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatBytes } from '@/utils/formatBytes.ts'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import StatisticsTab from './StatisticsTab.vue'
import type { StatCard } from './StatisticsSummaryCards.vue'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'

const props = defineProps<{
  serverId: string
  dbName: string
  collectionName: string
}>()

const { t } = useI18n()

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const stats = ref<Record<string, any> | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

const SIZE_FIELDS = new Set(['size', 'avgObjSize', 'storageSize', 'totalSize', 'totalIndexSize', 'freeStorageSize'])

function formatSizeValue(bytes: number): string {
  return `${formatBytes(bytes)} (${bytes.toLocaleString()})`
}

const cards = computed<StatCard[]>(() => {
  if (!stats.value) {
    return []
  }
  const s = stats.value
  const result: StatCard[] = []

  if (s.count != null) {
    result.push({
      label: t('statistics.cards.documents'),
      value: Number(s.count).toLocaleString(),
    })
  }
  if (s.avgObjSize != null) {
    result.push({
      label: t('statistics.cards.avgDocSize'),
      value: formatBytes(Number(s.avgObjSize)),
      subValue: t('statistics.bytes', { value: Number(s.avgObjSize).toLocaleString() }),
    })
  }
  if (s.size != null) {
    result.push({
      label: t('statistics.cards.dataSize'),
      value: formatBytes(Number(s.size)),
      subValue: t('statistics.bytes', { value: Number(s.size).toLocaleString() }),
    })
  }
  if (s.storageSize != null) {
    result.push({
      label: t('statistics.cards.storageSize'),
      value: formatBytes(Number(s.storageSize)),
      subValue: t('statistics.bytes', { value: Number(s.storageSize).toLocaleString() }),
    })
  }
  if (s.totalSize != null) {
    result.push({
      label: t('statistics.cards.totalSize'),
      value: formatBytes(Number(s.totalSize)),
      subValue: t('statistics.bytes', { value: Number(s.totalSize).toLocaleString() }),
    })
  }
  if (s.totalIndexSize != null) {
    result.push({
      label: t('statistics.cards.totalIndexSize'),
      value: formatBytes(Number(s.totalIndexSize)),
      subValue: t('statistics.bytes', { value: Number(s.totalIndexSize).toLocaleString() }),
    })
  }

  return result
})

const documents = computed(() => {
  if (!stats.value) {
    return []
  }
  // Clone and rewrite known size fields for display
  const doc = { ...stats.value }
  for (const key of SIZE_FIELDS) {
    if (typeof doc[key] === 'number') {
      doc[key] = formatSizeValue(doc[key])
    }
  }
  return [doc]
})

async function fetchStatistics() {
  loading.value = true
  error.value = null
  try {
    const result = await collectionsProxy.GetStatistics(props.serverId, props.dbName, props.collectionName)
    if (result.isSuccess) {
      stats.value = result.data
    } else {
      error.value = result.error
    }
  } catch (e) {
    error.value = String(e)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchStatistics()
})
</script>

<template>
  <div v-if="error" class="error-container">
    <div class="error-toolbar">
      <n-button size="small" @click="fetchStatistics">
        <template #icon>
          <n-icon :component="ArrowPathIcon" />
        </template>
        {{ t('statistics.toolbar.refresh') }}
      </n-button>
    </div>
    <div class="error-state">
      <n-result status="error" :title="t('statistics.error')" :description="error" />
    </div>
  </div>
  <statistics-tab
    v-else
    :cards="cards"
    :documents="documents"
    :loading="loading"
    @refresh="fetchStatistics" />
</template>

<style lang="scss" scoped>
.error-container {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.error-toolbar {
  padding: 4px 12px 8px;
  flex-shrink: 0;
}

.error-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
}
</style>
