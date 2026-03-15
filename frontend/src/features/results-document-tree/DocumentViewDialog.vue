<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useNotifier } from '@/utils/dialog'

const props = defineProps<{
  show: boolean
  document: unknown
}>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const { t } = useI18n()
const notifier = useNotifier()

const jsonText = computed(() => JSON.stringify(props.document, null, 2))

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(jsonText.value)
    notifier.success(t('query.contextMenu.copied'))
  } catch {
    notifier.error('Failed to copy to clipboard')
  }
}

function close() {
  emit('update:show', false)
}
</script>

<template>
  <n-modal
    :show="props.show"
    preset="card"
    :title="t('query.dialogs.viewDocument')"
    style="width: 700px; max-height: 80vh"
    :mask-closable="true"
    @update:show="emit('update:show', $event)"
  >
    <n-scrollbar style="max-height: 60vh">
      <n-code :code="jsonText" language="json" word-wrap />
    </n-scrollbar>
    <template #footer>
      <n-space justify="end">
        <n-button @click="copyToClipboard">
          {{ t('query.contextMenu.copyDocument') }}
        </n-button>
        <n-button @click="close">
          {{ t('common.cancel') }}
        </n-button>
      </n-space>
    </template>
  </n-modal>
</template>
