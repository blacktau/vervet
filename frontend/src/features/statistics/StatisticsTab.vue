<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import StatisticsSummaryCards, { type StatCard } from './StatisticsSummaryCards.vue'
import DocumentTreeTable from '@/features/results-document-tree/DocumentTreeTable.vue'

defineProps<{
  cards: StatCard[]
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  documents: any[]
  loading: boolean
}>()

const emit = defineEmits<{
  refresh: []
}>()

const { t } = useI18n()
</script>

<template>
  <div class="statistics-tab">
    <statistics-summary-cards v-if="cards.length > 0" :cards="cards" />
    <div class="statistics-toolbar">
      <n-button size="small" @click="emit('refresh')">
        <template #icon>
          <n-icon :component="ArrowPathIcon" />
        </template>
        {{ t('statistics.toolbar.refresh') }}
      </n-button>
    </div>
    <document-tree-table
      v-if="!loading && documents.length > 0"
      :documents="documents"
      :default-expand-depth="1"
      class="flex-item-expand" />
    <div v-else-if="loading" class="loading-state">
      <n-spin />
    </div>
  </div>
</template>

<style lang="scss" scoped>
.statistics-tab {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

.statistics-toolbar {
  padding: 4px 12px 8px;
  flex-shrink: 0;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
}
</style>
