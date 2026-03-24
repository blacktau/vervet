<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { DialogType, useDialogStore } from '@/stores/dialog'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'
import { useServerStore, type RegisteredServerNode } from '@/features/server-pane/serverStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'
import { useNotifier } from '@/utils/dialog'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const serverStore = useServerStore()
const tabStore = useTabStore()
const queryStore = useQueryStore()
const notifier = useNotifier()

const selectedServerId = ref<string | null>(null)
const selectedDatabase = ref<string | null>(null)
const databases = ref<string[]>([])
const loadingDatabases = ref(false)
const connecting = ref(false)

interface ServerPickerData {
  filePath: string
  skipServerSelection?: boolean
  serverId?: string
  database?: string
}

const dialogData = computed<ServerPickerData | undefined>(() => {
  return dialogStore.getDialogData<ServerPickerData>(DialogType.ServerPicker)
})

const visible = computed({
  get: () => dialogStore.isVisible(DialogType.ServerPicker),
  set: () => dialogStore.hide(DialogType.ServerPicker),
})

function flattenServers(nodes: RegisteredServerNode[]): RegisteredServerNode[] {
  const result: RegisteredServerNode[] = []
  for (const node of nodes) {
    if (!node.isGroup) {
      result.push(node)
    }
    if (node.children) {
      result.push(...flattenServers(node.children))
    }
  }
  return result
}

const serverOptions = computed(() => {
  return flattenServers(serverStore.serverTree).map((s) => ({
    label: s.name,
    value: s.id,
  }))
})

const databaseOptions = computed(() => {
  return databases.value.map((db) => ({
    label: db,
    value: db,
  }))
})

const canConfirm = computed(() => {
  return selectedServerId.value && selectedDatabase.value && !connecting.value
})

watch(visible, (isVisible) => {
  if (isVisible) {
    selectedServerId.value = dialogData.value?.serverId ?? null
    selectedDatabase.value = dialogData.value?.database ?? null
    databases.value = []
    connecting.value = false

    if (selectedServerId.value) {
      ensureConnectedAndLoadDatabases(selectedServerId.value)
    }
  }
})

async function ensureConnectedAndLoadDatabases(serverId: string) {
  if (!browserStore.isConnected(serverId)) {
    connecting.value = true
    try {
      const result = await browserStore.connect(serverId)
      if (!result.success) {
        return
      }
    } finally {
      connecting.value = false
    }
  }

  await loadDatabases(serverId)
}

async function loadDatabases(serverId: string) {
  loadingDatabases.value = true
  try {
    const result = await databasesProxy.GetDatabases(serverId)
    if (result.isSuccess) {
      databases.value = result.data
    } else {
      notifier.error(result.errorDetail || result.errorCode)
    }
  } finally {
    loadingDatabases.value = false
  }
}

function handleServerChange(value: string) {
  selectedServerId.value = value
  selectedDatabase.value = null
  databases.value = []
  if (value) {
    ensureConnectedAndLoadDatabases(value)
  }
}

function handleDatabaseChange(value: string) {
  selectedDatabase.value = value
}

function onConfirm() {
  if (!selectedServerId.value || !selectedDatabase.value || !dialogData.value?.filePath) {
    return
  }

  const serverId = selectedServerId.value
  const database = selectedDatabase.value
  const filePath = dialogData.value.filePath

  const queryId = tabStore.openQuery(serverId, database)
  if (queryId) {
    queryStore.loadFileByPath(queryId, filePath)
  }

  dialogStore.hide(DialogType.ServerPicker)
}

function onClose() {
  dialogStore.hide(DialogType.ServerPicker)
}
</script>

<template>
  <n-modal
    v-model:show="visible"
    :closable="false"
    :mask-closable="false"
    :negative-button-props="{ size: 'medium' }"
    :negative-text="t('common.cancel')"
    :positive-button-props="{ size: 'medium', disabled: !canConfirm }"
    :positive-text="t('common.confirm')"
    :show-icon="false"
    :title="t('workspaces.selectServerAndDatabase')"
    close-on-esc
    preset="dialog"
    transform-origin="center"
    @esc="onClose"
    @positive-click="onConfirm"
    @negative-click="onClose">
    <div class="server-picker-form">
      <n-form-item :label="t('workspaces.server')">
        <n-select
          :disabled="dialogData?.skipServerSelection || connecting"
          :loading="connecting"
          :options="serverOptions"
          :value="selectedServerId"
          :placeholder="t('workspaces.selectServer')"
          @update:value="handleServerChange" />
      </n-form-item>
      <n-form-item :label="t('workspaces.database')">
        <n-select
          :disabled="!selectedServerId || connecting"
          :loading="loadingDatabases"
          :options="databaseOptions"
          :value="selectedDatabase"
          :placeholder="t('workspaces.selectDatabase')"
          @update:value="handleDatabaseChange" />
      </n-form-item>
    </div>
  </n-modal>
</template>

<style lang="scss" scoped>
.server-picker-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
