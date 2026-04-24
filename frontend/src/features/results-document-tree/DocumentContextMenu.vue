<script lang="ts" setup>
import { h } from 'vue'
import { type DropdownOption, NDropdown, NIcon } from 'naive-ui'
import {
  ArrowDownOnSquareIcon,
  ClipboardIcon,
  ClipboardDocumentIcon,
  EyeIcon,
  PencilSquareIcon,
  PlusIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  show?: boolean
  x?: number
  y?: number
  options?: DropdownOption[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  select: [key: string]
}>()

const iconMap: Record<string, typeof EyeIcon> = {
  viewDocument: EyeIcon,
  editDocument: PencilSquareIcon,
  insertDocument: PlusIcon,
  copyDocument: ClipboardIcon,
  exportResults: ArrowDownOnSquareIcon,
  deleteDocument: TrashIcon,
  copyValue: ClipboardIcon,
  copyField: ClipboardDocumentIcon,
}

function renderIcon(option: DropdownOption) {
  const icon = iconMap[option.key as string]
  if (!icon) {
    return undefined
  }
  return h(NIcon, { size: 16 }, () => h(icon))
}

function renderLabel(option: DropdownOption) {
  const isDelete = option.key === 'deleteDocument'
  return h(
    'span',
    {
      class: isDelete ? 'context-menu-item context-menu-item--danger' : 'context-menu-item',
    },
    (option.label as string) || '',
  )
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
    :options="props.options"
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
:deep(.context-menu-item--danger) {
  color: #d03050;
}
</style>
