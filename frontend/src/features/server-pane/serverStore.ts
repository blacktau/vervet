import { defineStore } from 'pinia'
import * as serversProxy from 'wailsjs/go/api/ServersProxy'
import { type models } from 'wailsjs/go/models.ts'
import { isEmpty, union } from 'lodash'
import { useNotifier } from '@/utils/dialog.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'

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
        notifier.error(`error retrieving registered servers: ${result.error}`)
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
      if (response.isSuccess) {
        return {
          ...response.data,
        }
      }
      const notifier = useNotifier()
      notifier.error(`error retrieving registered server: ${response.error}`)
      return undefined
    },
    mergeServerDetails(dst: models.RegisteredServer, src?: models.RegisteredServer) {
      if (!src) {
        return dst
      }

      return merge(
        dst as unknown as Record<string, unknown>,
        src as unknown as Record<string, unknown>,
      ) as unknown as models.RegisteredServer
    },
    createDefaultServer(serverId: string): models.RegisteredServer {
      return {
        id: serverId,
        name: '',
        parentID: '',
        connectionID: '',
        serverName: '',
        isGroup: false,
      } as unknown as models.RegisteredServer
    },
    async saveServer(name: string, connectionString: string, parentId?: string, colour?: string) {
      const result = await serversProxy.SaveServer(
        parentId || '',
        name,
        connectionString,
        colour || '',
      )
      if (!result.isSuccess) {
        return { success: false, msg: result.error }
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
      console.log('updateServer', serverId, name, connectionString, parentId, colour)
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
        return { success: false, msg: result.error }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async deleteServer(serverId: string) {
      const browserStore = useDataBrowserStore()
      await browserStore.disconnect(serverId)
      const result = await serversProxy.RemoveNode(serverId)
      if (!result.isSuccess) {
        return { success: false, msg: result.error }
      }
      await this.refreshServers(true)
      return { success: true }
    },
    async createGroup(name: string, parentId?: string) {
      const result = await serversProxy.CreateGroup(name, parentId || '')
      if (!result.isSuccess) {
        return { success: false, msg: result.error }
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
        return { success: false, msg: result.error }
      }

      await this.refreshServers(true)
      return { success: true }
    },
    async deleteGroup(groupId: string) {
      const result = await serversProxy.RemoveNode(groupId)
      if (!result.isSuccess) {
        return { success: false, msg: result.error }
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

function merge(dst: Record<string, unknown>, src: Record<string, unknown>) {
  const keys = union(Object.keys(dst), Object.keys(src))
  for (const key of keys) {
    const t = typeof src[key]
    if (t === 'string') {
      dst[key] = src[key] || dst[key] || ''
    } else if (t === 'number') {
      dst[key] = src[key] || dst[key] || 0
    } else if (t === 'boolean') {
      dst[key] = src[key] || dst[key] || false
    } else if (t === 'object') {
      merge(dst[key] as Record<string, unknown>, (src[key] as Record<string, unknown>) || {})
    } else {
      dst[key] = src[key]
    }
  }
  return dst
}
