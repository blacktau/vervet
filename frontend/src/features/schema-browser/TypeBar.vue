<script setup lang="ts">
import { computed } from 'vue'
import type { models } from 'wailsjs/go/models'
import { useThemeVars } from 'naive-ui'
import { computeSegments } from './typeBarHelpers'

const props = defineProps<{ types: models.TypeStat[]; total: number }>()
const themeVars = useThemeVars()

const segments = computed(() =>
  computeSegments(props.types, props.total, themeVars.value.primaryColor),
)
</script>

<template>
  <div class="type-bar">
    <div
      v-for="seg in segments"
      :key="seg.type"
      data-test="segment"
      class="type-bar__seg"
      :style="{ width: seg.pct + '%', background: seg.color }"
      :title="`${seg.type}: ${seg.count} (${seg.pct.toFixed(1)}%)`"
    ></div>
  </div>
</template>

<style scoped>
.type-bar {
  display: flex;
  height: 8px;
  width: 100%;
  border-radius: 2px;
  overflow: hidden;
  background: rgba(127, 127, 127, 0.15);
}
.type-bar__seg {
  height: 100%;
}
</style>
