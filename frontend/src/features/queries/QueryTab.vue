<script lang="ts" setup>
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useMonacoEditor } from './useMonacoEditor'
import VerticalResizeableWrapper from '@/features/common/VerticalResizeableWrapper.vue'
import DocumentTreeTable from '@/features/results-document-tree/DocumentTreeTable.vue'
import JsonResultView from '@/features/results-json-view/JsonResultView.vue'
import { NButton, NIcon, NSpace, NSpin } from 'naive-ui'
import { PlayIcon, StopIcon } from '@heroicons/vue/24/solid'
import { TableCellsIcon, CodeBracketIcon } from '@heroicons/vue/24/outline'
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  queryId: string
}>()

const { t } = useI18n()
const queryStore = useQueryStore()
const tabStore = useTabStore()

const queryContentRef = ref<HTMLElement | null>(null)
const editorHeight = ref(300)

const defaultQuery = `// MongoDB Query
// Example: db.collection.find({ field: value })
`

const queryState = computed(() => queryStore.getQueryState(props.queryId))

const queryTabItem = computed(() => {
  const tab = tabStore.currentTab
  if (!tab) {
    return undefined
  }
  return tab.queries.find((q) => q.id === props.queryId)
})

const resizeOffset = computed(() => {
  return queryContentRef.value?.getBoundingClientRect().top ?? 0
})

const hasDocuments = computed(() => queryState.value.documents.length > 0)
const hasRawOutput = computed(() => queryState.value.rawOutput !== '')

const initialText = queryTabItem.value?.initialText
const { container: editorContainer, editor } = useMonacoEditor({
  language: 'javascript',
  value: initialText != null ? initialText : defaultQuery,
  queryId: props.queryId,
})

const runQuery = async () => {
  if (!editor.value) {
    return
  }
  const query = editor.value.getValue()
  await queryStore.executeQuery(props.queryId, query)
}

const cancelQuery = () => {
  queryStore.cancelQuery(props.queryId)
}

function setResultView(view: 'table' | 'json') {
  queryState.value.resultView = view
}

onMounted(async () => {
  const item = queryTabItem.value
  if (item) {
    queryStore.initQueryState(props.queryId, item.database)
    item.initialText = undefined
  }

  await queryStore.checkMongosh()
})
</script>

<template>
  <div class="query-tab">
    <div class="toolbar">
      <n-space align="center">
        <span class="database-label">
          {{ t('query.database') }}: <strong>{{ queryState.selectedDatabase }}</strong>
        </span>
        <n-button
          v-if="!queryState.loading"
          type="primary"
          size="small"
          :disabled="!queryState.selectedDatabase || queryStore.mongoshAvailable === false"
          @click="runQuery"
        >
          <template #icon>
            <n-icon :component="PlayIcon" />
          </template>
          {{ t('query.run') }}
        </n-button>
        <n-button v-else type="warning" size="small" @click="cancelQuery">
          <template #icon>
            <n-icon :component="StopIcon" />
          </template>
          {{ t('query.cancel') }}
        </n-button>
      </n-space>
      <div v-if="queryStore.mongoshAvailable === false" class="mongosh-warning">
        {{ t('query.mongoshNotFound') }}
      </div>
    </div>
    <div ref="queryContentRef" class="query-content">
      <vertical-resizeable-wrapper
        v-model:size="editorHeight"
        :min-size="100"
        :offset="resizeOffset"
      >
        <div class="editor-pane">
          <div ref="editorContainer" class="monaco-container" />
        </div>
      </vertical-resizeable-wrapper>
      <div class="results-pane">
        <n-tabs
          v-model:value="queryState.activeResultTab"
          type="line"
          size="small"
          :animated="false"
          pane-style="display: flex; flex-direction: column; flex: 1; min-height: 0;"
        >
          <template #suffix>
            <n-space v-if="hasDocuments" :size="2" style="margin-right: 8px">
              <n-button
                size="tiny"
                :type="queryState.resultView === 'table' ? 'primary' : 'default'"
                quaternary
                :title="t('query.tableView')"
                @click="setResultView('table')"
              >
                <template #icon>
                  <n-icon :component="TableCellsIcon" />
                </template>
              </n-button>
              <n-button
                size="tiny"
                :type="queryState.resultView === 'json' ? 'primary' : 'default'"
                quaternary
                :title="t('query.jsonView')"
                @click="setResultView('json')"
              >
                <template #icon>
                  <n-icon :component="CodeBracketIcon" />
                </template>
              </n-button>
            </n-space>
          </template>
          <n-tab-pane name="results" :tab="t('query.results')">
            <n-spin v-if="queryState.loading" :size="12" style="margin: 8px" />
            <pre v-else-if="queryState.error" class="results-content results-error">{{
              queryState.error
            }}</pre>
            <template v-else-if="hasDocuments">
              <div v-if="queryState.resultView === 'table'" class="structured-results">
                <document-tree-table :documents="queryState.documents" />
              </div>
              <div v-else class="json-results">
                <json-result-view :content="queryState.rawJson" />
              </div>
            </template>
            <pre v-else-if="hasRawOutput" class="results-content">{{ queryState.rawOutput }}</pre>
          </n-tab-pane>
          <n-tab-pane name="messages" :tab="t('query.messages')">
            <n-log
              class="messages-log"
              :log="queryState.messages"
              :font-size="13"
              language="vervet-log"
              trim
            />
          </n-tab-pane>
        </n-tabs>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.query-tab {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: 8px;
  gap: 8px;
}

.toolbar {
  flex-shrink: 0;
}

.database-label {
  font-size: 13px;
  color: var(--n-text-color-2);
}

.mongosh-warning {
  margin-top: 4px;
  font-size: 12px;
  color: var(--n-error-color);
}

.query-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.editor-pane {
  height: 100%;

  .monaco-container {
    height: 100%;
    width: 100%;
  }
}

.results-pane {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;

  :deep(.n-tabs) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  :deep(.n-tabs .n-tabs-nav) {
    flex-shrink: 0;
  }

  :deep(.n-tabs .n-tab-pane) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  :deep(.n-tabs .n-tabs-pane-wrapper) {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
  }

  .results-content {
    flex: 1;
    margin: 0;
    padding: 8px;
    overflow: auto;
    font-family: monospace;
    font-size: 13px;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--n-text-color-2);
    user-select: text;
    cursor: text;
  }

  .results-error {
    color: var(--n-error-color);
  }

  :deep(.messages-log) {
    height: 0 !important;
    flex: 1;
  }
}

.structured-results {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.json-results {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
