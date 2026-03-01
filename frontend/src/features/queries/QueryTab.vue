<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore'
import { useQueryStore } from '@/features/queries/queryStore'
import { useTabStore } from '@/features/tabs/tabs'
import VerticalResizeableWrapper from '@/features/common/VerticalResizeableWrapper.vue'
import { NButton, NIcon, NSpace, NSpin } from 'naive-ui'
import { PlayIcon, StopIcon } from '@heroicons/vue/24/solid'
import { ref, onMounted, onBeforeUnmount, watch, shallowRef, computed } from 'vue'
import * as monaco from 'monaco-editor'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  queryId: string
}>()

const { t } = useI18n()
const settingsStore = useSettingsStore()
const queryStore = useQueryStore()
const tabStore = useTabStore()

const editorContainer = ref<HTMLElement | null>(null)
const queryContentRef = ref<HTMLElement | null>(null)
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
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

const initMonaco = () => {
  if (!editorContainer.value) {
    return
  }

  const isDark = settingsStore.isDark

  const initialText = queryTabItem.value?.initialText
  const editorValue = initialText != null ? initialText : defaultQuery

  editor.value = monaco.editor.create(editorContainer.value, {
    value: editorValue,
    language: 'javascript',
    theme: isDark ? 'vervet-dark' : 'vervet-light',
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: 14,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    padding: { top: 8 },
  })
}

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

watch(
  () => settingsStore.isDark,
  (newVal) => {
    if (editor.value) {
      monaco.editor.setTheme(newVal ? 'vervet-dark' : 'vervet-light')
    }
  },
)

onMounted(async () => {
  const item = queryTabItem.value
  if (item) {
    queryStore.initQueryState(props.queryId, item.database)
  }

  initMonaco()

  // Clear initialText after use
  if (item) {
    item.initialText = undefined
  }

  await queryStore.checkMongosh()
})

onBeforeUnmount(() => {
  if (editor.value) {
    editor.value.dispose()
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
          pane-style="display: flex; flex-direction: column; flex: 1; min-height: 0;"
        >
          <n-tab-pane name="results" :tab="t('query.results')">
            <n-spin v-if="queryState.loading" :size="12" style="margin: 8px" />
            <pre v-else-if="queryState.error" class="results-content results-error">{{
              queryState.error
            }}</pre>
            <pre v-else class="results-content">{{ queryState.result }}</pre>
          </n-tab-pane>
          <n-tab-pane name="messages" :tab="t('query.messages')">
            <pre class="results-content messages-content">{{ queryState.messages }}</pre>
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
  height: 100%;
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
  height: 100%;

  :deep(.n-tabs) {
    height: 100%;
    display: flex;
    flex-direction: column;
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

  .messages-content {
    overflow: auto;
    min-height: 0;
  }
}
</style>
