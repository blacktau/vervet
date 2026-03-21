<script lang="ts" setup>
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useMonacoEditor } from './useMonacoEditor'
import VerticalResizeableWrapper from '@/features/common/VerticalResizeableWrapper.vue'
import DocumentTreeTable from '@/features/results-document-tree/DocumentTreeTable.vue'
import JsonResultView from '@/features/results-json-view/JsonResultView.vue'
import { NButton, NIcon, NSpace, NSpin, useThemeVars } from 'naive-ui'
import { PlayIcon, StopIcon } from '@heroicons/vue/24/solid'
import { CodeBracketIcon, FolderOpenIcon, ArrowDownTrayIcon, DocumentArrowDownIcon } from '@heroicons/vue/24/outline'
import ListTreeIcon from '@/features/icon/ListTreeIcon.vue'
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CollectionContext } from '@/features/results-document-tree/useDocumentContextMenu'
import * as monaco from 'monaco-editor'

const props = defineProps<{
  queryId: string
}>()

const { t } = useI18n()
const queryStore = useQueryStore()
const tabStore = useTabStore()
const dialog = useDialog()
const themeVars = useThemeVars()

const filenameBarStyle = computed(() => ({
  background: `${themeVars.value.primaryColor}15`,
}))

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

const collectionContext = computed<CollectionContext | undefined>(() => {
  const item = queryTabItem.value
  const tab = tabStore.currentTab
  if (!item?.collectionName || !tab) {
    return undefined
  }
  return {
    serverId: tab.serverId,
    dbName: item.database,
    collectionName: item.collectionName,
  }
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

const fileName = computed(() => {
  const fp = queryState.value.filePath
  if (!fp) {
    return null
  }
  return fp.split('/').pop() ?? fp
})

const saveFile = async () => {
  if (!editor.value) {
    return
  }
  await queryStore.saveFile(props.queryId, editor.value.getValue())
}

const saveFileAs = async () => {
  if (!editor.value) {
    return
  }
  await queryStore.saveFileAs(props.queryId, editor.value.getValue())
}

const promptSaveIfDirty = async (): Promise<boolean> => {
  return new Promise((resolve) => {
    dialog.warning({
      title: t('query.unsavedChangesTitle'),
      content: t('query.unsavedChangesMessage', { filename: fileName.value ?? 'Untitled' }),
      positiveText: t('query.unsavedChangesSave'),
      negativeText: t('query.unsavedChangesDontSave'),
      onPositiveClick: async () => {
        await saveFile()
        resolve(true)
      },
      onNegativeClick: () => {
        resolve(true)
      },
      onClose: () => {
        resolve(false)
      },
    })
  })
}

const openFile = async () => {
  if (queryState.value.isDirty) {
    const shouldContinue = await promptSaveIfDirty()
    if (!shouldContinue) {
      return
    }
  }

  const content = await queryStore.openFile(props.queryId)
  if (content !== null && editor.value) {
    editor.value.setValue(content)
  }
}

function setResultView(view: 'table' | 'json') {
  queryState.value.resultView = view
}

async function handleDocumentChanged() {
  const editor_val = editor.value
  if (!editor_val) {
    return
  }
  const query = editor_val.getValue()
  await queryStore.executeQuery(props.queryId, query)
}

watch(
  () => queryState.value.filePath,
  (newPath) => {
    const item = queryTabItem.value
    if (item) {
      item.filePath = newPath ?? undefined
    }
  }
)

onMounted(async () => {
  const item = queryTabItem.value
  if (item) {
    queryStore.initQueryState(props.queryId, item.database)
    item.initialText = undefined
  }

  await queryStore.checkMongosh()

  if (editor.value) {
    editor.value.addAction({
      id: 'vervet.openFile',
      label: 'Open File',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyO],
      run: () => { openFile() },
    })

    editor.value.addAction({
      id: 'vervet.saveFile',
      label: 'Save File',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
      run: () => { saveFile() },
    })

    editor.value.addAction({
      id: 'vervet.saveFileAs',
      label: 'Save File As',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyS],
      run: () => { saveFileAs() },
    })

    editor.value.onDidChangeModelContent(() => {
      const currentContent = editor.value?.getValue() ?? ''
      queryStore.setCurrentContent(props.queryId, currentContent)
      const saved = queryState.value.savedContent
      if (saved !== null) {
        queryStore.setDirty(props.queryId, currentContent !== saved)
      }
    })
  }
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
      <n-space align="center" :size="4">
        <n-button size="small" quaternary @click="openFile" :title="t('query.openFile')">
          <template #icon>
            <n-icon :component="FolderOpenIcon" />
          </template>
        </n-button>
        <n-button size="small" quaternary @click="saveFile" :title="t('query.saveFile')">
          <template #icon>
            <n-icon :component="ArrowDownTrayIcon" />
          </template>
        </n-button>
        <n-button size="small" quaternary @click="saveFileAs" :title="t('query.saveFileAs')">
          <template #icon>
            <n-icon :component="DocumentArrowDownIcon" />
          </template>
        </n-button>
      </n-space>
    </div>
    <div v-if="queryStore.mongoshAvailable === false" class="mongosh-warning">
      {{ t('query.mongoshNotFound') }}
    </div>
    <div v-if="queryState.filePath" class="filename-bar" :style="filenameBarStyle" :title="queryState.filePath">
      <span class="filename-text">{{ fileName }}</span>
      <span v-if="queryState.isDirty" class="dirty-indicator">&bull;</span>
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
                  <n-icon :component="ListTreeIcon" />
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
                <document-tree-table
                  :documents="queryState.documents"
                  :enable-context-menu="true"
                  :collection-context="collectionContext"
                  @document-changed="handleDocumentChanged" />
              </div>
              <div v-else class="json-results">
                <json-result-view :content="queryState.rawJson" />
              </div>
            </template>
            <pre v-else-if="hasRawOutput" class="results-content">{{ queryState.rawOutput }}</pre>
          </n-tab-pane>
          <n-tab-pane name="messages" :tab="t('query.messagesTab')">
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
  display: flex;
  align-items: center;
  justify-content: space-between;
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

.filename-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 12px;
  color: var(--n-text-color-3);
  border-bottom: 1px solid var(--n-border-color);
  flex-shrink: 0;
}

.filename-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dirty-indicator {
  color: var(--n-warning-color);
  font-size: 16px;
  line-height: 1;
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
    -webkit-user-select: text;
    user-select: text;
    cursor: text;
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
