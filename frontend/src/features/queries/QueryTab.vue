<script lang="ts" setup>
import { useSettingsStore } from '@/features/settings/settingsStore'
import { useQueryStore } from '@/features/queries/queryStore'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'
import { useTabStore } from '@/features/tabs/tabs'
import VerticalResizeableWrapper from '@/features/common/VerticalResizeableWrapper.vue'
import { NButton, NIcon, NSpace, NSelect, NSpin } from 'naive-ui'
import { PlayIcon, StopIcon } from '@heroicons/vue/24/solid'
import { ref, onMounted, onBeforeUnmount, watch, shallowRef, computed } from 'vue'
import * as monaco from 'monaco-editor'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const settingsStore = useSettingsStore()
const queryStore = useQueryStore()
const browserStore = useDataBrowserStore()
const tabStore = useTabStore()

const editorContainer = ref<HTMLElement | null>(null)
const queryContentRef = ref<HTMLElement | null>(null)
const editor = shallowRef<monaco.editor.IStandaloneCodeEditor | null>(null)
const editorHeight = ref(300)

const defaultQuery = `// MongoDB Query
// Example: db.collection.find({ field: value })

db.users.find({})
`

const databaseOptions = computed(() => {
  const serverId = tabStore.currentTabId
  if (!serverId) return []
  const connection = browserStore.connections.find((c) => c.serverID === serverId)
  if (!connection?.databases) return []
  return connection.databases.map((db) => ({ label: db.name, value: db.name }))
})

const resizeOffset = computed(() => {
  return queryContentRef.value?.getBoundingClientRect().top ?? 0
})

const initMonaco = () => {
  if (!editorContainer.value) return

  const isDark = settingsStore.isDark

  editor.value = monaco.editor.create(editorContainer.value, {
    value: defaultQuery,
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
  if (!editor.value) return
  const query = editor.value.getValue()
  await queryStore.executeQuery(query)
}

const cancelQuery = () => {
  queryStore.cancelQuery()
}

const handleDatabaseChange = (value: string) => {
  queryStore.setDatabase(value)
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
  initMonaco()
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
        <n-select
          :value="queryStore.selectedDatabase"
          :options="databaseOptions"
          :placeholder="t('query.selectDatabase')"
          size="small"
          style="width: 200px"
          @update:value="handleDatabaseChange"
        />
        <n-button
          v-if="!queryStore.loading"
          type="primary"
          size="small"
          :disabled="!queryStore.selectedDatabase || queryStore.mongoshAvailable === false"
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
        <div class="results-header">
          {{ t('query.results') }}
          <n-spin v-if="queryStore.loading" :size="12" style="margin-left: 8px" />
        </div>
        <pre v-if="queryStore.error" class="results-content results-error">{{
          queryStore.error
        }}</pre>
        <pre v-else class="results-content">{{ queryStore.result }}</pre>
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
  min-height: 100px;
  background-color: var(--n-color);
  border: 1px solid var(--n-border-color);
  border-radius: 4px;

  .results-header {
    display: flex;
    align-items: center;
    padding: 4px 8px;
    font-size: 12px;
    font-weight: 500;
    color: var(--n-text-color-3);
    border-bottom: 1px solid var(--n-border-color);
    flex-shrink: 0;
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
}
</style>
