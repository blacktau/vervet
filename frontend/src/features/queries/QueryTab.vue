<script lang="ts" setup>
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useMonacoEditor } from './useMonacoEditor'
import VerticalResizeableWrapper from '@/features/common/VerticalResizeableWrapper.vue'
import DocumentTreeTable from '@/features/results-document-tree/DocumentTreeTable.vue'
import JsonResultView from '@/features/results-json-view/JsonResultView.vue'
import { NButton, NIcon, NSpace, NSpin, useThemeVars } from 'naive-ui'
import { PlayIcon, StopIcon } from '@heroicons/vue/24/solid'
import {
  CodeBracketIcon,
  FolderOpenIcon,
  ArrowDownTrayIcon,
  ArrowDownOnSquareIcon,
  DocumentArrowDownIcon,
  TrashIcon,
} from '@heroicons/vue/24/outline'
import { useDialogStore } from '@/stores/dialog'
import ListTreeIcon from '@/features/icon/ListTreeIcon.vue'
import { useSettingsStore } from '@/features/settings/settingsStore'
import { isMacOS } from '@/init/environment'
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CollectionContext } from '@/features/results-document-tree/useDocumentContextMenu'
import * as monaco from 'monaco-editor'

const props = defineProps<{
  queryId: string
}>()

const { t } = useI18n()
const queryStore = useQueryStore()
const tabStore = useTabStore()
const settingsStore = useSettingsStore()
const dialogStore = useDialogStore()
const dialog = useDialog()
const themeVars = useThemeVars()

const filenameBarStyle = computed(() => ({
  background: `${themeVars.value.primaryColor}15`,
}))

const messagesFontSize = computed(() => settingsStore.terminal.font.size || 13)

const messagesFontKey = computed(
  () => `${settingsStore.terminal.font.family}-${settingsStore.terminal.font.size}`,
)

const messagesLogStyle = computed(() => {
  const family = settingsStore.terminal.font.family
  return family ? { '--n-font-family': `"${family}"` } : {}
})

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

const modKey = isMacOS() ? 'Cmd' : 'Ctrl'

const runButtonTooltip = computed(() => {
  if (!queryState.value.selectedDatabase) {
    return t('errors.no_database_selected')
  }
  if (queryStore.mongoshAvailable === false) {
    return t('errors.shell_not_found')
  }
  return `${t('query.run')} (F5 / ${modKey}+Enter)`
})

const openFileTooltip = computed(() => `${t('query.openFile')} (${modKey}+O)`)
const saveFileTooltip = computed(() => `${t('query.saveFile')} (${modKey}+S)`)
const saveFileAsTooltip = computed(() => `${t('query.saveFileAs')} (${modKey}+Shift+S)`)

const hasDocuments = computed(() => queryState.value.documents.length > 0)
const limitTruncated = computed(() => {
  const limit = queryState.value.activeLimit
  return limit !== null && queryState.value.documents.length >= limit
})
const hasRawOutput = computed(() => queryState.value.rawOutput !== '')
const hasExecuted = computed(() => queryState.value.messages.length > 0)
const hasNoResults = computed(
  () =>
    !queryState.value.error &&
    !hasDocuments.value &&
    !hasRawOutput.value &&
    hasExecuted.value &&
    !queryState.value.loading,
)

function clearMessages() {
  queryState.value.messages = ''
}

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

function openExport() {
  dialogStore.openExportResultsDialog({
    ejson: queryStore.getRawJson(props.queryId),
    collectionName: queryTabItem.value?.collectionName,
  })
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
  },
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
      run: () => {
        openFile()
      },
    })

    editor.value.addAction({
      id: 'vervet.saveFile',
      label: 'Save File',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
      run: () => {
        saveFile()
      },
    })

    editor.value.addAction({
      id: 'vervet.saveFileAs',
      label: 'Save File As',
      keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyMod.Shift | monaco.KeyCode.KeyS],
      run: () => {
        saveFileAs()
      },
    })

    editor.value.addAction({
      id: 'vervet.runQuery',
      label: 'Run Query',
      keybindings: [monaco.KeyCode.F5, monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
      run: () => {
        runQuery()
      },
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

watch(
  [() => tabStore.pendingFocusQueryId, editor],
  async ([queryId, editorInstance]) => {
    if (queryId === props.queryId && editorInstance) {
      tabStore.pendingFocusQueryId = null
      await nextTick()
      editorInstance.focus()
    }
  },
  { immediate: true },
)
</script>

<template>
  <div class="query-tab">
    <div class="toolbar">
      <n-space align="center">
        <span class="database-label">
          {{ t('query.database') }}:
          <strong>{{ queryState.selectedDatabase }}</strong>
        </span>
        <n-tooltip v-if="!queryState.loading" :delay="800">
          <template #trigger>
            <n-button
              type="primary"
              size="small"
              :disabled="!queryState.selectedDatabase || queryStore.mongoshAvailable === false"
              @click="runQuery">
              <template #icon>
                <n-icon :component="PlayIcon" />
              </template>
              {{ t('query.run') }}
            </n-button>
          </template>
          {{ runButtonTooltip }}
        </n-tooltip>
        <n-button v-else type="warning" size="small" @click="cancelQuery">
          <template #icon>
            <n-icon :component="StopIcon" />
          </template>
          {{ t('query.cancel') }}
        </n-button>
      </n-space>
      <n-space align="center" :size="4">
        <n-tooltip>
          <template #trigger>
            <n-button size="small" @click="openFile">
              <template #icon>
                <n-icon :component="FolderOpenIcon" />
              </template>
            </n-button>
          </template>
          {{ openFileTooltip }}
        </n-tooltip>
        <n-tooltip>
          <template #trigger>
            <n-button size="small" @click="saveFile">
              <template #icon>
                <n-icon :component="ArrowDownTrayIcon" />
              </template>
            </n-button>
          </template>
          {{ saveFileTooltip }}
        </n-tooltip>
        <n-tooltip>
          <template #trigger>
            <n-button size="small" @click="saveFileAs">
              <template #icon>
                <n-icon :component="DocumentArrowDownIcon" />
              </template>
            </n-button>
          </template>
          {{ saveFileAsTooltip }}
        </n-tooltip>
      </n-space>
    </div>
    <div v-if="queryStore.mongoshAvailable === false" class="mongosh-warning">
      {{ t('errors.shell_not_found') }}
    </div>
    <div
      v-if="queryState.filePath"
      class="filename-bar"
      :style="filenameBarStyle"
      :title="queryState.filePath">
      <span class="filename-text">{{ fileName }}</span>
      <span v-if="queryState.isDirty" class="dirty-indicator">&bull;</span>
    </div>
    <div ref="queryContentRef" class="query-content">
      <vertical-resizeable-wrapper
        v-model:size="editorHeight"
        :min-size="100"
        :offset="resizeOffset">
        <div class="editor-pane">
          <div ref="editorContainer" class="monaco-container" />
        </div>
      </vertical-resizeable-wrapper>
      <div class="results-pane">
        <n-tabs
          v-model:value="queryState.activeResultTab"
          type="card"
          size="small"
          :animated="false"
          :closable="false"
          pane-style="display: flex; flex-direction: column; flex: 1; min-height: 0; padding-top: 0;">
          <template #suffix>
            <n-space
              v-if="hasDocuments && queryState.activeResultTab === 'results'"
              :size="4"
              style="margin-right: 8px">
              <n-tooltip>
                <template #trigger>
                  <n-button
                    size="small"
                    :type="queryState.resultView === 'table' ? 'primary' : 'default'"
                    @click="setResultView('table')">
                    <template #icon>
                      <n-icon :component="ListTreeIcon" />
                    </template>
                  </n-button>
                </template>
                {{ t('query.tableView') }}
              </n-tooltip>
              <n-tooltip>
                <template #trigger>
                  <n-button
                    size="small"
                    :type="queryState.resultView === 'json' ? 'primary' : 'default'"
                    @click="setResultView('json')">
                    <template #icon>
                      <n-icon :component="CodeBracketIcon" />
                    </template>
                  </n-button>
                </template>
                {{ t('query.jsonView') }}
              </n-tooltip>
              <n-tooltip>
                <template #trigger>
                  <n-button
                    size="small"
                    :disabled="!hasDocuments"
                    data-testid="export-results-button"
                    @click="openExport">
                    <template #icon>
                      <n-icon :component="ArrowDownOnSquareIcon" />
                    </template>
                  </n-button>
                </template>
                {{ t('export.button') }}
              </n-tooltip>
            </n-space>
            <n-space
              v-if="queryState.activeResultTab === 'messages'"
              :size="4"
              style="margin-right: 8px">
              <n-tooltip>
                <template #trigger>
                  <n-button size="small" :disabled="!queryState.messages" @click="clearMessages">
                    <template #icon>
                      <n-icon :component="TrashIcon" />
                    </template>
                  </n-button>
                </template>
                {{ t('query.clearMessages') }}
              </n-tooltip>
            </n-space>
          </template>
          <n-tab-pane name="results" :tab="t('query.results')">
            <div v-if="queryState.loading" class="loading-state">
              <n-spin size="medium">
                <template #description>
                  {{ t('query.messages.executing') }}
                </template>
                <div class="loading-state-spacer" />
              </n-spin>
              <n-button size="small" @click="cancelQuery">
                {{ t('query.cancel') }}
              </n-button>
            </div>
            <pre v-else-if="queryState.error" class="results-content results-error">{{
              queryState.error
            }}</pre>
            <template v-else-if="hasDocuments">
              <div v-if="limitTruncated" class="limit-hint">
                {{ t('query.limitInEffect', { limit: queryState.activeLimit }) }}
              </div>
              <div v-if="queryState.resultView === 'table'" class="structured-results">
                <document-tree-table
                  :documents="queryState.documents"
                  :enable-context-menu="true"
                  :collection-context="collectionContext"
                  @document-changed="handleDocumentChanged"
                  @export-requested="openExport" />
              </div>
              <div v-else class="json-results">
                <json-result-view :content="queryStore.getRawJson(props.queryId)" />
              </div>
            </template>
            <pre v-else-if="hasRawOutput" class="results-content">{{ queryState.rawOutput }}</pre>
            <div v-else-if="hasNoResults" class="empty-results">
              <span class="empty-results-text">{{ t('query.noResults') }}</span>
            </div>
          </n-tab-pane>
          <n-tab-pane name="messages" :tab="t('query.messagesTab')">
            <n-log
              :key="messagesFontKey"
              class="messages-log"
              :log="queryState.messages"
              :font-size="messagesFontSize"
              :style="messagesLogStyle"
              language="vervet-log"
              trim />
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

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
    flex: 1;

    :deep(.n-spin-description) {
      white-space: nowrap;
    }
  }

  .loading-state-spacer {
    height: 60px;
    min-width: 200px;
  }

  .empty-results {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--n-text-color-3);
  }

  .empty-results-text {
    font-size: 14px;
  }

  :deep(.messages-log) {
    height: 0 !important;
    flex: 1;
    padding-top: 8px;
    -webkit-user-select: text;
    user-select: text;
    cursor: text;
  }
}

.limit-hint {
  flex-shrink: 0;
  padding: 4px 8px;
  font-size: 12px;
  color: var(--n-text-color-3);
  background-color: color-mix(in srgb, var(--n-warning-color) 12%, transparent);
  border-bottom: 1px solid var(--n-border-color);
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
