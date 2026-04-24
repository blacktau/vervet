<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ExportResults } from 'wailsjs/go/api/ExportProxy'
import { useNotifier } from '@/utils/dialog'
import { useExportStore } from './exportStore'
import { buildDefaultFilename, type ExportFormat } from './defaultFilename'
import {
  buildExportPayload,
  separatorChoiceFromValue,
  separatorFromChoice,
  type SeparatorChoice,
} from './exportDialogHelpers'

interface Props {
  show: boolean
  ejson: string
  collectionName?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
}>()

const { t } = useI18n()
const store = useExportStore()

const format = computed({
  get: () => store.format,
  set: (v: ExportFormat) => store.setFormat(v),
})

const isCsv = computed(() => format.value === 'csv')

const separatorChoice = ref<SeparatorChoice>('comma')
const customSeparator = ref('')

// Seed separator state from store on mount / when dialog opens
watch(
  () => props.show,
  (visible) => {
    if (!visible) {
      return
    }
    const stored = store.csv.separator
    const choice = separatorChoiceFromValue(stored)
    separatorChoice.value = choice
    if (choice === 'custom') {
      customSeparator.value = stored
    } else {
      customSeparator.value = ''
    }
  },
  { immediate: true },
)

const defaultFilename = computed(() => buildDefaultFilename(props.collectionName, format.value))

const exporting = ref(false)

async function onExport() {
  const separator = separatorFromChoice(separatorChoice.value, customSeparator.value)

  if (isCsv.value) {
    store.setCsv({
      separator,
      includeHeader: store.csv.includeHeader,
      utf8Bom: store.csv.utf8Bom,
    })
  }

  const payload = buildExportPayload({
    format: format.value,
    ejson: props.ejson,
    collectionName: props.collectionName,
    defaultFilename: defaultFilename.value,
    isCsv: isCsv.value,
    separator,
    includeHeader: store.csv.includeHeader,
    utf8Bom: store.csv.utf8Bom,
  })

  exporting.value = true
  try {
    const res = await ExportResults(payload)
    if (!res.isSuccess) {
      const notifier = useNotifier()
      notifier.error(t('export.error', { message: res.errorCode ?? '' }))
      return
    }
    if (res.data) {
      const notifier = useNotifier()
      notifier.success(t('export.saved', { path: res.data }))
    }
    emit('update:show', false)
  } finally {
    exporting.value = false
  }
}

function onCancel() {
  emit('update:show', false)
}
</script>

<template>
  <n-modal
    :show="show"
    :closable="false"
    :mask-closable="false"
    :show-icon="false"
    :title="t('export.title')"
    close-on-esc
    preset="dialog"
    style="width: 480px"
    transform-origin="center"
    @esc="onCancel"
    @update:show="(v) => emit('update:show', v)">
    <n-space vertical size="large">
      <!-- Format picker -->
      <n-form-item :label="t('export.format.label')" :show-feedback="false" label-placement="top">
        <n-radio-group v-model:value="format">
          <n-radio value="csv">{{ t('export.format.csv') }}</n-radio>
          <n-radio value="json">{{ t('export.format.json') }}</n-radio>
          <n-radio value="ndjson">{{ t('export.format.ndjson') }}</n-radio>
        </n-radio-group>
      </n-form-item>

      <!-- CSV options -->
      <template v-if="isCsv">
        <n-form-item
          :label="t('export.csv.separator.label')"
          :show-feedback="false"
          label-placement="top">
          <n-space vertical>
            <n-radio-group v-model:value="separatorChoice">
              <n-space>
                <n-radio value="comma">{{ t('export.csv.separator.comma') }}</n-radio>
                <n-radio value="tab">{{ t('export.csv.separator.tab') }}</n-radio>
                <n-radio value="semicolon">{{ t('export.csv.separator.semicolon') }}</n-radio>
                <n-radio value="pipe">{{ t('export.csv.separator.pipe') }}</n-radio>
                <n-radio value="custom">{{ t('export.csv.separator.custom') }}</n-radio>
              </n-space>
            </n-radio-group>
            <n-input
              v-if="separatorChoice === 'custom'"
              v-model:value="customSeparator"
              :maxlength="1"
              :placeholder="t('export.csv.separator.customPlaceholder')"
              style="width: 160px" />
          </n-space>
        </n-form-item>

        <n-checkbox v-model:checked="store.csv.includeHeader">
          {{ t('export.csv.includeHeader') }}
        </n-checkbox>

        <n-space vertical size="small">
          <n-checkbox v-model:checked="store.csv.utf8Bom">
            {{ t('export.csv.utf8Bom') }}
          </n-checkbox>
          <n-text depth="3" style="font-size: 0.85em">
            {{ t('export.csv.utf8BomHelp') }}
          </n-text>
        </n-space>
      </template>

      <!-- Filename preview -->
      <n-text depth="3">
        {{ t('export.filenamePreview', { name: defaultFilename }) }}
      </n-text>
    </n-space>

    <template #action>
      <n-button :focusable="false" @click="onCancel">
        {{ t('export.cancel') }}
      </n-button>
      <n-button
        :focusable="false"
        :loading="exporting"
        data-testid="export-confirm"
        type="primary"
        @click="onExport">
        {{ t('export.button') }}
      </n-button>
    </template>
  </n-modal>
</template>
