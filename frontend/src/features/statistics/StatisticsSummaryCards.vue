<script lang="ts" setup>
export interface StatCard {
  label: string
  value: string
  subValue?: string
}

defineProps<{
  cards: StatCard[]
}>()
</script>

<template>
  <div class="summary-cards">
    <div v-for="card in cards" :key="card.label" class="summary-card">
      <div class="card-label">{{ card.label }}</div>
      <div class="card-value">{{ card.value }}</div>
      <div v-if="card.subValue" class="card-sub-value">{{ card.subValue }}</div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.summary-cards {
  display: flex;
  gap: 10px;
  padding: 12px;
  flex-wrap: wrap;
  flex-shrink: 0;
}

.summary-card {
  flex: 1;
  min-width: 120px;
  padding: 10px 14px;
  background: rgba(56, 176, 0, 0.08);
  border: 1px solid rgba(56, 176, 0, 0.2);
  border-radius: 6px;
}

.card-label {
  font-size: 11px;
  color: var(--n-text-color-3, #888);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.card-value {
  font-size: 20px;
  font-weight: 600;
  color: #b5cea8;
  margin-top: 4px;
}

.card-sub-value {
  font-size: 11px;
  color: var(--n-text-color-4, #555);
  margin-top: 2px;
}
</style>
