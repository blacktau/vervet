<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { DialogType, useDialogStore } from '@/stores/dialog'
import { useDataBrowserStore } from '@/features/data-browser/browserStore'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'
import { useNotifier } from '@/utils/dialog'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const dialogStore = useDialogStore()
const browserStore = useDataBrowserStore()
const tabStore = useTabStore()
const queryStore = useQueryStore()
const notifier = useNotifier()

const selectedServerId = ref<string | null>(null)
const selectedDatabase = ref<string | null>(null)
const databases = ref<string[]>([])
const loadingDatabases = ref(false)

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

const serverOptions = computed(() => {
  return browserStore.connections.map((c) => ({
    label: c.name,
    value: c.serverID,
  }))
})

const databaseOptions = computed(() => {
  return databases.value.map((db) => ({
    label: db,
    value: db,
  }))
})

const canConfirm = computed(() => {
  return selectedServerId.value && selectedDatabase.value
})

watch(visible, (isVisible) => {
  if (isVisible) {
    selectedServerId.value = dialogData.value?.serverId ?? null
    selectedDatabase.value = dialogData.value?.database ?? null
    databases.value = []

    if (selectedServerId.value) {
      loadDatabases(selectedServerId.value)
    }
  }
})

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
    loadDatabases(value)
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
          :disabled="dialogData?.skipServerSelection"
          :options="serverOptions"
          :value="selectedServerId"
          :placeholder="t('workspaces.selectServer')"
          @update:value="handleServerChange" />
      </n-form-item>
      <n-form-item :label="t('workspaces.database')">
        <n-select
          :disabled="!selectedServerId"
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
