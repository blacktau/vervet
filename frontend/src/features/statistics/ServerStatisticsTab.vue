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
}>()

const { t } = useI18n()

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const stats = ref<Record<string, any> | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) {
    parts.push(`${days}${t('common.unit_day')}`)
  }
  if (hours > 0) {
    parts.push(`${hours}${t('common.unit_hour')}`)
  }
  if (mins > 0) {
    parts.push(`${mins}${t('common.unit_minute')}`)
  }
  if (parts.length === 0) {
    parts.push(`${Math.floor(seconds)}${t('common.unit_second')}`)
  }
  return parts.join(' ')
}

const cards = computed<StatCard[]>(() => {
  if (!stats.value) {
    return []
  }
  const s = stats.value
  const result: StatCard[] = []

  if (s.version != null) {
    result.push({
      label: t('statistics.serverCards.version'),
      value: String(s.version),
    })
  }
  if (s.uptime != null) {
    result.push({
      label: t('statistics.serverCards.uptime'),
      value: formatUptime(Number(s.uptime)),
    })
  }
  if (s.connections && s.connections.current != null) {
    result.push({
      label: t('statistics.serverCards.connections'),
      value: Number(s.connections.current).toLocaleString(),
      subValue: s.connections.available != null
        ? `${Number(s.connections.available).toLocaleString()} ${t('statistics.serverCards.available')}`
        : undefined,
    })
  }
  if (s.mem && s.mem.resident != null) {
    result.push({
      label: t('statistics.serverCards.memResident'),
      value: formatBytes(Number(s.mem.resident) * 1024 * 1024),
    })
  }
  if (s.mem && s.mem.virtual != null) {
    result.push({
      label: t('statistics.serverCards.memVirtual'),
      value: formatBytes(Number(s.mem.virtual) * 1024 * 1024),
    })
  }
  if (s.opcounters) {
    const total = ['insert', 'query', 'update', 'delete', 'command']
      .reduce((sum, key) => sum + (Number(s.opcounters[key]) || 0), 0)
    result.push({
      label: t('statistics.serverCards.totalOperations'),
      value: total.toLocaleString(),
    })
  }

  return result
})

const documents = computed(() => {
  if (!stats.value) {
    return []
  }
  return [stats.value]
})

async function fetchStatistics() {
  loading.value = true
  error.value = null
  try {
    const result = await collectionsProxy.GetServerStatistics(props.serverId)
    if (result.isSuccess) {
      stats.value = result.data
    } else {
      error.value = t(`errors.${result.errorCode}`)
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
