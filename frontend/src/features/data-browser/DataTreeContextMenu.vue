<script lang="ts" setup>
import { h } from 'vue'
import { type DropdownOption, NDropdown, NIcon } from 'naive-ui'
import {
  ArrowPathIcon,
  ArrowRightStartOnRectangleIcon,
  ChartBarIcon,
  EyeIcon,
  InformationCircleIcon,
  PencilSquareIcon,
  PlayIcon,
  PlusCircleIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { type ContextMenuOption } from '@/features/data-browser/types.ts'

interface Props {
  show?: boolean
  x?: number
  y?: number
  options?: ContextMenuOption[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  select: [key: string]
}>()

const iconMap: Record<string, typeof InformationCircleIcon> = {
  addDatabase: PlusCircleIcon,
  serverStatus: ChartBarIcon,
  disconnect: ArrowRightStartOnRectangleIcon,
  openQuery: PlayIcon,
  dropDatabase: TrashIcon,
  statistics: InformationCircleIcon,
  refresh: ArrowPathIcon,
  addCollection: PlusCircleIcon,
  rename: PencilSquareIcon,
  dropCollection: TrashIcon,
  viewIndexes: EyeIcon,
}

function renderIcon(option: DropdownOption) {
  const icon = iconMap[option.key as string]
  if (!icon) return undefined
  return h(NIcon, { size: 16 }, () => h(icon))
}

function renderLabel(option: DropdownOption) {
  return h('span', { class: 'context-menu-item' }, (option.label as string) || '')
}

function handleSelect(option: string) {
  emit('select', option)
  emit('close')
}
</script>

<template>
  <n-dropdown
    v-if="props.show"
    :keyboard="true"
    :options="props.options as DropdownOption[]"
    :render-icon="renderIcon"
    :render-label="renderLabel"
    :show="props.show"
    :x="props.x"
    :y="props.y"
    placement="bottom-start"
    trigger="manual"
    @clickoutside="emit('close')"
    @select="handleSelect" />
</template>

<style lang="scss" scoped>
:deep(.context-menu-item) {
  font-size: 13px;
}
</style>
