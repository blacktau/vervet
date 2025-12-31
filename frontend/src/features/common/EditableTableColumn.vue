<script setup lang="ts">
import IconButton from '@/features/common/IconButton.vue'
import {
  ArrowDownOnSquareIcon,
  ArrowPathIcon,
  DocumentDuplicateIcon,
  PencilSquareIcon,
  TrashIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  bindKey?: string
  editing?: boolean
  readonly?: boolean
  canRefresh?: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'copy'): void
  (e: 'refresh'): void
  (e: 'save'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <div v-if="props.editing" class="flex-box-h edit-column-func">
    <icon-button :icon="ArrowDownOnSquareIcon" @click="emit('save')" />
    <icon-button :icon="XMarkIcon" @click="emit('cancel')" />
    >
  </div>
  <div v-else class="flex-box-h edit-column-func">
    <icon-button
      :icon="DocumentDuplicateIcon"
      :title="$t('interface.copy_value')"
      @click="emit('copy')" />
    <icon-button
      v-if="props.canRefresh"
      :icon="ArrowPathIcon"
      :title="$t('interface.reload')"
      @click="emit('refresh')" />
    <icon-button
      v-if="!props.readonly"
      :icon="PencilSquareIcon"
      :title="$t('interface.edit_row')"
      @click="emit('edit')" />
    <n-popconfirm
      v-if="props.bindKey"
      :negative-text="$t('common.cancel')"
      :positive-text="$t('common.confirm')"
      @positive-click="emit('delete')">
      <template #trigger>
        <icon-button :icon="TrashIcon" :title="$t('interface.delete_row')" />
      </template>
      {{ $t('dialogue.remove_tip', { name: props.bindKey }) }}
    </n-popconfirm>
  </div>
</template>

<style scoped lang="scss">
.edit-column-func {
  align-items: center;
  justify-content: center;
  gap: 10px;
}
</style>
