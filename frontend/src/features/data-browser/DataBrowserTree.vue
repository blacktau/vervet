<script lang="ts" setup>
import { useThemeVars } from 'naive-ui'
import { useRender } from '@/utils/render.ts'
import { useI18n } from 'vue-i18n'
import { computed } from 'vue'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

const props = defineProps<{
  loading: boolean
  pattern?: string
}>()

const themeVars = useThemeVars()
const render = useRender()
const i18n = useI18n()
const browserStore = useDataBrowserStore()

const treeroots = computed(() => {
  return browserStore.connections.map((x) => {
    return {
      label: x.name,
      key: x.serverId,
      isLeaf: false,
      type: DataNodeType.Server,
    } as DataTreeNode
  })
})
</script>

<template>
  <div class="flex-box-v browser-treee-wrapper" @contextmenu="(e) => e.preventDefault()">
    <n-tree :data="treeroots" block-line></n-tree>
  </div>
</template>

<style lang="scss" scoped></style>
