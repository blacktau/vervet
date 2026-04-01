<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ExclamationTriangleIcon } from '@heroicons/vue/24/outline'
import { DialogType, useDialogStore } from '@/stores/dialog.ts'

const props = defineProps<{
  onExport: (includeSensitiveData: boolean) => void
}>()

const { t } = useI18n()
const dialogStore = useDialogStore()

const includeSensitiveData = ref(false)
const visible = computed(() => dialogStore.dialogs[DialogType.Export]?.visible ?? false)

function handleClose() {
  includeSensitiveData.value = false
  dialogStore.hide(DialogType.Export)
}

function handleExport() {
  props.onExport(includeSensitiveData.value)
  handleClose()
}
</script>

<template>
  <n-modal
    :show="visible"
    preset="dialog"
    :title="t('serverPane.dialogs.export.title')"
    :positive-text="t('serverPane.dialogs.export.exportButton')"
    :negative-text="t('common.cancel')"
    @positive-click="handleExport"
    @negative-click="handleClose"
    @close="handleClose">
    <n-space vertical>
      <n-checkbox v-model:checked="includeSensitiveData">
        {{ t('serverPane.dialogs.export.includeSensitiveData') }}
      </n-checkbox>
      <n-alert
        v-if="includeSensitiveData"
        type="warning"
        :bordered="false">
        <template #icon>
          <n-icon :component="ExclamationTriangleIcon" />
        </template>
        {{ t('serverPane.dialogs.export.sensitiveDataWarning') }}
      </n-alert>
    </n-space>
  </n-modal>
</template>
