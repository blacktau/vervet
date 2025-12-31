<script setup lang="ts">
import { useThemeVars } from 'naive-ui'
import { useTabStore } from '@/stores/tabs'
import { computed } from 'vue'
import { find, map } from 'lodash'

const themeVars = useThemeVars()
const props = defineProps<{
  server: string
}>()

const tabStore = useTabStore()
const tab = computed(() => {
  return map(tabStore.tabs, (item) => ({
    key: item,
    label: item.title,
  }))
})

const tabContent = computed(() => {
  const tab = find(tabStore.tabs, { name: props.server })
  if (!tab) {
    return {}
  }

  return {
    name: tab.name,
    subTab: tab.subTab

  }
})
</script>

<template>
  <div class="content-container flex-box-v">
    <n-tabs
      ref="tabsRef"
      :tabs-padding="5"
      :theme-overrides="{
        tabFontWeightActive: 'normal',
        tabGapSmallLine: '10px',
        tabGapMediumLine: '10px',
        tabGapLargeLine: '10px',
      }"
      :value="selectedSubtab"
      class="content-sub-tab"
      :default-value="BrowserTabType.Status.toString()"
      pane-class="content-sub-tab-pane"
      placement="top"
      tab-style="padding-left: 10px; padding-right: 10px;"
      type="line"
      @update:value="tabStore.switchSubTab"></n-tabs>
  </div>
</template>

<style scoped lang="scss"></style>
