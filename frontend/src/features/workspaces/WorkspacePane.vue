<script lang="ts" setup>
import { onMounted } from 'vue'
import { useWorkspaceStore } from '@/features/workspaces/workspaceStore'
import WorkspaceToolbar from '@/features/workspaces/WorkspaceToolbar.vue'
import WorkspaceTree from '@/features/workspaces/WorkspaceTree.vue'
import WorkspaceEmptyState from '@/features/workspaces/WorkspaceEmptyState.vue'

const workspaceStore = useWorkspaceStore()

onMounted(() => {
  workspaceStore.loadWorkspaces()
})

function handleCreateWorkspace() {
  // Trigger the create flow via the toolbar's create handler
  // Re-use WorkspaceToolbar's create logic by calling the store directly
  // A simple prompt — the toolbar has the full dialog version
  workspaceStore.createWorkspace('New Workspace')
}
</script>

<template>
  <div class="nav-pane-container flex-box-v">
    <template v-if="workspaceStore.hasWorkspaces">
      <workspace-toolbar />
      <template v-if="workspaceStore.activeWorkspace">
        <template v-if="workspaceStore.treeData.length > 0">
          <workspace-tree />
        </template>
        <template v-else>
          <workspace-empty-state
            type="no-folders"
            @add-folder="workspaceStore.addFolder()" />
        </template>
      </template>
    </template>
    <template v-else>
      <workspace-empty-state
        type="no-workspaces"
        @create-workspace="handleCreateWorkspace" />
    </template>
  </div>
</template>

<style lang="scss" scoped>
.nav-pane-container {
  height: 100%;
}
</style>
