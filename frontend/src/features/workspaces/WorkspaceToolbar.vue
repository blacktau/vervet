<script lang="ts" setup>
import { h, ref, computed } from 'vue'
import { NInput } from 'naive-ui'
import { useWorkspaceStore } from '@/features/workspaces/workspaceStore'
import { useDialoger } from '@/utils/dialog'
import IconButton from '@/features/common/IconButton.vue'
import {
  PlusIcon,
  FolderPlusIcon,
  Cog6ToothIcon,
  ArrowPathIcon,
} from '@heroicons/vue/24/outline'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const workspaceStore = useWorkspaceStore()
const dialoger = useDialoger()

const workspaceOptions = computed(() => {
  return workspaceStore.workspaces.map((w) => ({
    label: w.name,
    value: w.id,
  }))
})

const gearOptions = computed(() => [
  { label: t('workspaces.rename'), key: 'rename' },
  { label: t('workspaces.delete'), key: 'delete' },
])

function handleWorkspaceChange(value: string) {
  workspaceStore.setActiveWorkspace(value)
}

function handleCreate() {
  const nameRef = ref('')
  dialoger.show({
    type: 'info',
    title: t('workspaces.createWorkspace'),
    positiveText: t('common.create'),
    negativeText: t('common.cancel'),
    content: () => h(NInput, {
      value: nameRef.value,
      onUpdateValue: (v: string) => { nameRef.value = v },
      placeholder: t('workspaces.workspaceName'),
      autofocus: true,
    }),
    onPositiveClick: () => {
      if (nameRef.value.trim()) {
        workspaceStore.createWorkspace(nameRef.value.trim())
      }
    },
  })
}

function handleGearSelect(key: string) {
  if (!workspaceStore.activeWorkspace) {
    return
  }

  if (key === 'rename') {
    const nameRef = ref(workspaceStore.activeWorkspace.name)
    dialoger.show({
      type: 'info',
      title: t('workspaces.renameWorkspace'),
      positiveText: t('common.save'),
      negativeText: t('common.cancel'),
      content: () => h(NInput, {
        value: nameRef.value,
        onUpdateValue: (v: string) => { nameRef.value = v },
        placeholder: t('workspaces.workspaceName'),
        autofocus: true,
      }),
      onPositiveClick: () => {
        if (nameRef.value.trim() && workspaceStore.activeWorkspaceId) {
          workspaceStore.renameWorkspace(workspaceStore.activeWorkspaceId, nameRef.value.trim())
        }
      },
    })
  }

  if (key === 'delete') {
    dialoger.warning({
      title: t('workspaces.deleteWorkspace'),
      content: t('workspaces.deleteWorkspaceConfirm', { name: workspaceStore.activeWorkspace.name }),
      positiveText: t('common.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => {
        if (workspaceStore.activeWorkspaceId) {
          workspaceStore.deleteWorkspace(workspaceStore.activeWorkspaceId)
        }
      },
    })
  }
}
</script>

<template>
  <div class="workspace-toolbar flex-box-h">
    <n-select
      :options="workspaceOptions"
      :value="workspaceStore.activeWorkspaceId"
      size="small"
      style="flex: 1; min-width: 0"
      @update:value="handleWorkspaceChange" />

    <icon-button
      :icon="PlusIcon"
      :stroke-width="3"
      size="18"
      t-tooltip="workspaces.createWorkspace"
      @click="handleCreate" />

    <n-tooltip :delay="800" :keep-alive-on-hover="false" :show-arrow="false">
      <template #trigger>
        <span style="display: inline-flex">
          <n-dropdown
            :options="gearOptions"
            trigger="click"
            @select="handleGearSelect">
            <icon-button
              :disabled="!workspaceStore.activeWorkspace"
              :icon="Cog6ToothIcon"
              :stroke-width="2.5"
              size="18" />
          </n-dropdown>
        </span>
      </template>
      {{ t('workspaces.settings') }}
    </n-tooltip>

    <icon-button
      :disabled="!workspaceStore.activeWorkspace"
      :icon="FolderPlusIcon"
      :stroke-width="2.5"
      size="18"
      t-tooltip="workspaces.addFolder"
      @click="workspaceStore.addFolder()" />

    <icon-button
      :disabled="!workspaceStore.activeWorkspace"
      :icon="ArrowPathIcon"
      :stroke-width="2.5"
      size="18"
      t-tooltip="workspaces.refresh"
      @click="workspaceStore.refreshTree()" />
  </div>
</template>

<style lang="scss" scoped>
.workspace-toolbar {
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
}
</style>
