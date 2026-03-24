<script lang="ts" setup>
import { FolderPlusIcon, BriefcaseIcon } from '@heroicons/vue/24/outline'
import { computed } from 'vue'

const props = defineProps<{
  type: 'no-workspaces' | 'no-folders'
}>()

const emit = defineEmits<{
  (e: 'createWorkspace'): void
  (e: 'addFolder'): void
}>()

const icon = computed(() => {
  return props.type === 'no-workspaces' ? BriefcaseIcon : FolderPlusIcon
})

const actionLabel = computed(() => {
  return props.type === 'no-workspaces'
    ? 'workspaces.createWorkspace'
    : 'workspaces.addFolder'
})

function handleAction() {
  if (props.type === 'no-workspaces') {
    emit('createWorkspace')
  } else {
    emit('addFolder')
  }
}
</script>

<template>
  <div class="workspace-empty-state">
    <n-icon :component="icon" :size="48" color="var(--n-text-color-3)" />
    <p class="workspace-empty-message">
      {{ $t(props.type === 'no-workspaces' ? 'workspaces.noWorkspacesMessage' : 'workspaces.noFoldersMessage') }}
    </p>
    <n-button secondary type="primary" @click="handleAction">
      {{ $t(actionLabel) }}
    </n-button>
  </div>
</template>

<style lang="scss" scoped>
.workspace-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 24px;
  height: 100%;
  text-align: center;
}

.workspace-empty-message {
  margin: 0;
  opacity: 0.6;
  font-size: 13px;
}
</style>
