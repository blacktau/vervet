import { defineStore } from 'pinia'
import * as serversProxy from 'wailsjs/go/api/ServersProxy'
import * as filesProxy from 'wailsjs/go/api/FilesProxy'
import { type models } from 'wailsjs/go/models.ts'
import { isEmpty } from 'lodash'
import { useNotifier } from '@/utils/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { i18nGlobal } from '@/i18n'
import type { ConnectionConfig } from '@/types/ConnectionConfig'

export interface RegisteredServerNode extends models.RegisteredServer {
  children?: RegisteredServerNode[]
  isLeaf?: boolean
}

type ServerStoreState = {
  serverTree: RegisteredServerNode[]
}

export const useServerStore = defineStore('server', {
  state: () =>
    ({
      serverTree: [],
    }) as ServerStoreState,
  actions: {
    refreshServers: async function (force: boolean = false) {
      if (!force && !isEmpty(this.serverTree)) {
        return
      }

      const nodeMap: Record<string, RegisteredServerNode> = {}
      const tree: RegisteredServerNode[] = []

      const result = await serversProxy.GetServers()

      if (!result.isSuccess) {
        const notifier = useNotifier()
        notifier.error(i18nGlobal.t(`errors.${result.errorCode}`), { title: i18nGlobal.t('errorTitles.loadServers'), detail: result.errorDetail })
        return
      }

      for (const node of result.data) {
        nodeMap[node.id] = {
          ...node,
          children: node.isGroup ? [] : undefined,
        }
      }

      for (const node of result.data) {
        if (!node.parentID || node.parentID === '') {
          const treeNode = nodeMap[node.id]
          if (treeNode) {
            tree.push(treeNode)
          }
        } else {
          const parentNode = nodeMap[node.parentID]
          if (parentNode) {
            const child = nodeMap[node.id]
            if (
              child &&
              parentNode.children &&
              !parentNode.children.some((x) => x.id === child.id)
            ) {
              parentNode.children.push(child)
              parentNode.children.sort(nodeComparator)
            }
          }
        }
      }

      tree.sort(nodeComparator)
      this.serverTree = tree
    },
    async getServerDetails(id?: string) {
      if (id == null) {
        return undefined
      }

      const response = await serversProxy.GetServer(id)
      if (!response.isSuccess) {
        const notifier = useNotifier()
        notifier.error(i18nGlobal.t(`errors.${response.errorCode}`), { title: i18nGlobal.t('errorTitles.loadServerDetails'), detail: response.errorDetail })
        return undefined
      }

      const configResult = await serversProxy.GetConnectionConfig(id)
      if (!configResult.isSuccess) {
        // Fallback to legacy URI fetch
        const uriResponse = await serversProxy.GetURI(id)
        if (!uriResponse.isSuccess) {
          const notifier = useNotifier()
          notifier.error(i18nGlobal.t(`errors.${uriResponse.errorCode}`), { title: i18nGlobal.t('errorTitles.loadServerDetails'), detail: uriResponse.errorDetail })
          return undefined
        }
        return {
          ...response.data,
          uri: uriResponse.data,
          authMethod: 'password' as const,
        }
      }

      return {
        ...response.data,
        uri: configResult.data.uri,
        authMethod: configResult.data.authMethod,
        oidcConfig: configResult.data.oidcConfig,
      }
    },
    async saveServer(name: string, connectionString: string, parentId?: string, colour?: string) {
      const result = await serversProxy.SaveServer(
        parentId || '',
        name,
        connectionString,
        colour || '',
      )
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async saveServerWithConfig(name: string, parentId: string, colour: string, config: ConnectionConfig) {
      const result = await serversProxy.SaveServerWithConfig(parentId, name, colour, config)
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }
      await this.refreshServers(true)
      return { success: true }
    },
    async updateServer(
      serverId: string | null,
      name: string,
      connectionString: string,
      parentId?: string,
      colour?: string,
    ) {
      if (serverId == null) {
        return { success: false, msg: 'serverId is required' }
      }
      const result = await serversProxy.UpdateServer(
        serverId,
        name,
        connectionString,
        parentId || '',
        colour || '',
      )
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async updateServerWithConfig(serverId: string | null, name: string, parentId: string, colour: string, config: ConnectionConfig) {
      if (serverId == null) {
        return { success: false, msg: 'serverId is required' }
      }
      const result = await serversProxy.UpdateServerWithConfig(serverId, name, parentId, colour, config)
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }
      await this.refreshServers(true)
      return { success: true }
    },
    async deleteServer(serverId: string) {
      const browserStore = useDataBrowserStore()
      await browserStore.disconnect(serverId)
      const result = await serversProxy.RemoveNode(serverId)
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }
      await this.refreshServers(true)
      return { success: true }
    },
    async createGroup(name: string, parentId?: string) {
      const result = await serversProxy.CreateGroup(name, parentId || '')
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }
      await this.refreshServers(true)
      return { success: true }
    },
    async updateGroup(groupId: string, newName: string, parentId?: string) {
      const group = this.findServerById(groupId)
      if (!group) {
        return { success: false, msg: 'group not found' }
      }

      if (group.name === newName && group.parentID === (parentId || '')) {
        return { success: true }
      }

      const result = await serversProxy.UpdateGroup(groupId, newName, parentId || '')
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async exportServers(serverIDs: string[], includeSensitiveData: boolean) {
      const exportResult = await serversProxy.ExportServers(serverIDs, includeSensitiveData)
      if (!exportResult.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${exportResult.errorCode}`) }
      }

      const title = i18nGlobal.t('serverPane.exportServers')
      const defaultName = 'servers.json'
      const filters = [{ displayName: 'JSON', pattern: '*.json' }]

      const pathResult = await filesProxy.SaveFile(title, defaultName, filters)
      if (!pathResult.isSuccess || !pathResult.data) {
        return { success: true } // User cancelled — not an error
      }

      const writeResult = await filesProxy.WriteFile(pathResult.data, exportResult.data)
      if (!writeResult.isSuccess) {
        return { success: false, msg: writeResult.errorDetail || writeResult.errorCode }
      }

      return { success: true }
    },
    async importServers() {
      const title = i18nGlobal.t('serverPane.importServers')
      const filters = [
        { displayName: 'JSON', pattern: '*.json' },
        { displayName: 'All Files', pattern: '*.*' },
      ]

      const pathResult = await filesProxy.SelectFile(title, filters)
      if (!pathResult.isSuccess || !pathResult.data) {
        return { success: true } // User cancelled
      }

      const readResult = await filesProxy.ReadFile(pathResult.data)
      if (!readResult.isSuccess) {
        return { success: false, msg: readResult.errorDetail || readResult.errorCode }
      }

      const importResult = await serversProxy.ImportServers(readResult.data)
      if (!importResult.isSuccess) {
        return { success: false, msg: importResult.errorDetail || importResult.errorCode }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async deleteGroup(groupId: string) {
      const result = await serversProxy.RemoveNode(groupId)
      if (!result.isSuccess) {
        return { success: false, msg: i18nGlobal.t(`errors.${result.errorCode}`) }
      }
      await this.refreshServers(true)
      return { success: true }
    },
  },
  getters: {
    findServerById(state: ServerStoreState) {
      return (id: string) => {
        return findServerById(id, state.serverTree)
      }
    },
  },
})

function findServerById(
  id: string,
  nodeList?: RegisteredServerNode[],
): RegisteredServerNode | undefined {
  if (!nodeList) {
    return undefined
  }

  for (const node of nodeList) {
    if (node.id === id) {
      return node
    }
    const child = findServerById(id, node.children)
    if (child) {
      return child
    }
  }

  return undefined
}

function nodeComparator(
  a: RegisteredServerNode | models.RegisteredServer,
  b: RegisteredServerNode | models.RegisteredServer,
) {
  if (a.isGroup && !b.isGroup) {
    return -1
  }
  if (!a.isGroup && b.isGroup) {
    return 1
  }

  return a.name.localeCompare(b.name)
}
