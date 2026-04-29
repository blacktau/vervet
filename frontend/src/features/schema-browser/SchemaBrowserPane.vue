<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSchemaStore } from './schemaStore'
import SchemaFieldTree from './SchemaFieldTree.vue'
import { DialogType, useDialogStore } from '@/stores/dialog'

const props = defineProps<{
  tabId: string
  serverId: string
  dbName: string
  collectionName: string
}>()

const i18n = useI18n()
const store = useSchemaStore()
const message = useMessage()
const dialogStore = useDialogStore()

const sizeOptions = [
  { label: '100', value: 100 },
  { label: '500', value: 500 },
  { label: '1000', value: 1000 },
  { label: '5000', value: 5000 },
]

const state = computed(() => store.stateFor(props.tabId))

async function run(size: number) {
  await store.sample(props.tabId, props.serverId, props.dbName, props.collectionName, size)
}

function onSizeChange(val: number) {
  void run(val)
}

function onCancel() {
  store.cancel(props.tabId)
}

function onCopyPath(path: string) {
  void navigator.clipboard.writeText(path)
  message.success(i18n.t('schemaBrowser.copiedPath'))
}

function onCreateIndex(path: string) {
  dialogStore.showNewDialog(DialogType.CreateIndex, {
    serverID: props.serverId,
    dbName: props.dbName,
    collectionName: props.collectionName,
    presetField: path,
  })
}

onMounted(() => {
  if (!state.value) {
    void run(1000)
  }
})
</script>

<template>
  <div class="schema-pane">
    <div class="schema-pane__header">
      <NSelect
        :value="state?.sampleSize ?? 1000"
        :options="sizeOptions"
        size="small"
        style="width: 110px"
        @update:value="onSizeChange"
      />
      <NButton size="small" :disabled="state?.loading" @click="run(state?.sampleSize ?? 1000)">
        {{ $t('schemaBrowser.resample') }}
      </NButton>
      <span v-if="state?.result" class="schema-pane__count">
        {{
          $t('schemaBrowser.sampledOf', {
            sampled: state.result.sampledCount,
            total: state.result.totalCount,
          })
        }}
      </span>
      <NButton v-if="state?.loading" size="small" type="warning" @click="onCancel">
        {{ $t('schemaBrowser.cancel') }}
      </NButton>
    </div>

    <div class="schema-pane__body">
      <div v-if="state?.loading" class="schema-pane__centered">
        <NSpin />
      </div>
      <div v-else-if="state?.error" class="schema-pane__centered schema-pane__error">
        <span>{{ state.error }}</span>
        <NButton size="small" @click="run(state.sampleSize)">{{ $t('schemaBrowser.retry') }}</NButton>
      </div>
      <div
        v-else-if="state?.result && state.result.fields.length === 0"
        class="schema-pane__centered"
      >
        {{ $t('schemaBrowser.empty') }}
      </div>
      <SchemaFieldTree
        v-else-if="state?.result"
        :fields="state.result.fields"
        :sampled-count="state.result.sampledCount"
        @copy-path="onCopyPath"
        @create-index="onCreateIndex"
      />
    </div>
  </div>
</template>

<style scoped>
.schema-pane {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.schema-pane__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-bottom: 1px solid var(--n-border-color, rgba(127, 127, 127, 0.2));
}
.schema-pane__count {
  font-size: 12px;
  opacity: 0.75;
}
.schema-pane__body {
  flex: 1;
  overflow: auto;
  padding: 8px 0;
}
.schema-pane__centered {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px;
  opacity: 0.75;
}
.schema-pane__error {
  flex-direction: column;
}
</style>
