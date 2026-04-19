import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { TreeOption } from 'naive-ui'
import * as workspacesProxy from 'wailsjs/go/api/WorkspacesProxy'
import { useNotifier } from '@/utils/dialog'

// Temporary local interfaces — will be replaced by generated Wails types
interface Workspace {
  id: string
  name: string
  folders: string[]
}

interface WorkspaceData {
  workspaces: Workspace[]
  activeWorkspaceId: string
}

interface DirectoryEntry {
  name: string
  path: string
  isDirectory: boolean
}

function entriesToTreeOptions(entries: DirectoryEntry[] | null): TreeOption[] {
  if (!entries) {
    return []
  }
  return entries.map((entry) => ({
    key: entry.path,
    label: entry.name,
    isLeaf: !entry.isDirectory,
  }))
}

export const useWorkspaceStore = defineStore('workspaces', () => {
  const workspaces = ref<Workspace[]>([])
  const activeWorkspaceId = ref<string | null>(null)
  const treeData = ref<TreeOption[]>([])
  const expandedKeys = ref<string[]>([])
  const loading = ref(false)

  const activeWorkspace = computed(() => {
    if (!activeWorkspaceId.value) {
      return undefined
    }
    return workspaces.value.find((w) => w.id === activeWorkspaceId.value)
  })

  const hasWorkspaces = computed(() => workspaces.value.length > 0)

  async function loadWorkspaces() {
    loading.value = true
    try {
      const result = await workspacesProxy.GetWorkspaces()
      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(result.errorDetail || result.errorCode)
        return
      }

      const data = result.data as WorkspaceData
      workspaces.value = data.workspaces ?? []
      activeWorkspaceId.value = data.activeWorkspaceId || null

      if (activeWorkspace.value) {
        await loadTree()
      }
    } finally {
      loading.value = false
    }
  }

  async function createWorkspace(name: string) {
    const result = await workspacesProxy.CreateWorkspace(name)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    const workspace = result.data as Workspace
    workspaces.value.push(workspace)
    await setActiveWorkspace(workspace.id)
  }

  async function renameWorkspace(id: string, newName: string) {
    const result = await workspacesProxy.RenameWorkspace(id, newName)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    const workspace = workspaces.value.find((w) => w.id === id)
    if (workspace) {
      workspace.name = newName
    }
  }

  async function deleteWorkspace(id: string) {
    const result = await workspacesProxy.DeleteWorkspace(id)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    workspaces.value = workspaces.value.filter((w) => w.id !== id)

    if (activeWorkspaceId.value === id) {
      if (workspaces.value.length > 0) {
        await setActiveWorkspace(workspaces.value[0]!.id)
      } else {
        activeWorkspaceId.value = null
        treeData.value = []
        expandedKeys.value = []
      }
    }
  }

  async function setActiveWorkspace(id: string) {
    const result = await workspacesProxy.SetActiveWorkspace(id)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    activeWorkspaceId.value = id
    await loadTree()
  }

  async function addFolder() {
    if (!activeWorkspaceId.value) {
      return
    }

    const result = await workspacesProxy.AddFolder(activeWorkspaceId.value)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    if (!result.data) {
      return
    }

    const workspace = workspaces.value.find((w) => w.id === activeWorkspaceId.value)
    if (workspace) {
      workspace.folders.push(result.data as string)
    }
    await loadTree()
  }

  async function removeFolder(folderPath: string) {
    if (!activeWorkspaceId.value) {
      return
    }

    const result = await workspacesProxy.RemoveFolder(activeWorkspaceId.value, folderPath)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return
    }

    const workspace = workspaces.value.find((w) => w.id === activeWorkspaceId.value)
    if (workspace) {
      workspace.folders = workspace.folders.filter((f) => f !== folderPath)
    }
    await loadTree()
  }

  async function loadTree() {
    if (!activeWorkspace.value) {
      treeData.value = []
      return
    }

    const folders = activeWorkspace.value.folders
    const rootNodes: TreeOption[] = []

    for (const folder of folders) {
      const result = await workspacesProxy.ReadDirectory(folder)
      if (!result.isSuccess) {
        // Show folder as a leaf with no children if unreadable
        rootNodes.push({
          key: folder,
          label: folder.split('/').pop() || folder,
          isLeaf: false,
          children: [],
        })
        continue
      }

      const entries = result.data as DirectoryEntry[]
      rootNodes.push({
        key: folder,
        label: folder.split('/').pop() || folder,
        isLeaf: false,
        children: entriesToTreeOptions(entries),
      })
    }

    treeData.value = rootNodes
    expandedKeys.value = folders
  }

  async function loadDirectory(path: string): Promise<TreeOption[]> {
    const result = await workspacesProxy.ReadDirectory(path)
    if (!result.isSuccess) {
      const notifier = useNotifier()
      notifier.error(result.errorDetail || result.errorCode)
      return []
    }

    const entries = result.data as DirectoryEntry[]
    return entriesToTreeOptions(entries)
  }

  async function refreshTree() {
    expandedKeys.value = []
    await loadTree()
  }

  return {
    workspaces,
    activeWorkspaceId,
    treeData,
    expandedKeys,
    loading,
    activeWorkspace,
    hasWorkspaces,
    loadWorkspaces,
    createWorkspace,
    renameWorkspace,
    deleteWorkspace,
    setActiveWorkspace,
    addFolder,
    removeFolder,
    loadTree,
    loadDirectory,
    refreshTree,
  }
})
