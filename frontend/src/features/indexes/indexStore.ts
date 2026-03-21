import { defineStore } from 'pinia'
import { useNotifier } from '@/utils/dialog.ts'
import { i18nGlobal } from '@/i18n'
import * as indexesProxy from 'wailsjs/go/api/IndexesProxy'

type IndexKeyField = {
  field: string
  direction: number | string
}

export type IndexInfo = {
  name: string
  keys: IndexKeyField[]
  unique: boolean
  sparse: boolean
  ttl?: number
  size: number
  usage: number
}

interface IndexStoreState {
  indexes: Record<string, IndexInfo[]>
  loading: Record<string, boolean>
}

function cacheKey(serverId: string, dbName: string, collectionName: string): string {
  return `${serverId}:${dbName}:${collectionName}`
}

export const useIndexStore = defineStore('indexes', {
  state: (): IndexStoreState => ({
    indexes: {},
    loading: {},
  }),
  actions: {
    async getIndexes(serverId: string, dbName: string, collectionName: string) {
      const key = cacheKey(serverId, dbName, collectionName)
      this.loading[key] = true

      try {
        const result = await indexesProxy.GetIndexes(serverId, dbName, collectionName)
        if (!result.isSuccess) {
          useNotifier().error(i18nGlobal.t(`errors.${result.errorCode}`))
          return
        }
        this.indexes[key] = result.data ?? []
      } catch (e) {
        const err = e as Error
        useNotifier().error(err.message)
      } finally {
        this.loading[key] = false
      }
    },

    async createIndex(
      serverId: string,
      dbName: string,
      collectionName: string,
      request: {
        keys: IndexKeyField[]
        name?: string
        unique: boolean
        sparse: boolean
        ttl?: number
      },
    ): Promise<boolean> {
      try {
        const result = await indexesProxy.CreateIndex(
          serverId,
          dbName,
          collectionName,
          request,
        )
        if (!result.isSuccess) {
          useNotifier().error(i18nGlobal.t(`errors.${result.errorCode}`))
          return false
        }
        await this.getIndexes(serverId, dbName, collectionName)
        return true
      } catch (e) {
        const err = e as Error
        useNotifier().error(err.message)
        return false
      }
    },

    async editIndex(
      serverId: string,
      dbName: string,
      collectionName: string,
      request: {
        oldName: string
        keys: IndexKeyField[]
        name?: string
        unique: boolean
        sparse: boolean
        ttl?: number
      },
    ): Promise<boolean> {
      try {
        const result = await indexesProxy.EditIndex(
          serverId,
          dbName,
          collectionName,
          request,
        )
        if (!result.isSuccess) {
          useNotifier().error(i18nGlobal.t(`errors.${result.errorCode}`))
          return false
        }
        await this.getIndexes(serverId, dbName, collectionName)
        return true
      } catch (e) {
        const err = e as Error
        useNotifier().error(err.message)
        return false
      }
    },

    async dropIndex(
      serverId: string,
      dbName: string,
      collectionName: string,
      indexName: string,
    ): Promise<boolean> {
      try {
        const result = await indexesProxy.DropIndex(
          serverId,
          dbName,
          collectionName,
          indexName,
        )
        if (!result.isSuccess) {
          useNotifier().error(i18nGlobal.t(`errors.${result.errorCode}`))
          return false
        }
        await this.getIndexes(serverId, dbName, collectionName)
        return true
      } catch (e) {
        const err = e as Error
        useNotifier().error(err.message)
        return false
      }
    },

    isLoading(serverId: string, dbName: string, collectionName: string): boolean {
      return this.loading[cacheKey(serverId, dbName, collectionName)] ?? false
    },

    getIndexList(serverId: string, dbName: string, collectionName: string): IndexInfo[] {
      return this.indexes[cacheKey(serverId, dbName, collectionName)] ?? []
    },
  },
})
