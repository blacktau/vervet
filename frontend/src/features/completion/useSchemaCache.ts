import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'
import { EventsOn } from 'wailsjs/runtime/runtime'

interface FieldInfo {
  path: string
  types: string[]
  children?: FieldInfo[]
}

interface CollectionSchema {
  fields: FieldInfo[]
}

const schemaCache = new Map<string, CollectionSchema>()
const pendingRequests = new Map<string, Promise<CollectionSchema | null>>()
let listenerRegistered = false

function cacheKey(serverId: string, dbName: string, collectionName: string): string {
  return `${serverId}:${dbName}:${collectionName}`
}

function registerDisconnectListener() {
  if (listenerRegistered) {
    return
  }
  listenerRegistered = true
  EventsOn('connection-disconnected', (serverId: string) => {
    for (const key of schemaCache.keys()) {
      if (key.startsWith(`${serverId}:`)) {
        schemaCache.delete(key)
      }
    }
  })
}

export async function getCollectionSchema(
  serverId: string,
  dbName: string,
  collectionName: string,
): Promise<CollectionSchema | null> {
  registerDisconnectListener()

  const key = cacheKey(serverId, dbName, collectionName)

  const cached = schemaCache.get(key)
  if (cached) {
    return cached
  }

  const pending = pendingRequests.get(key)
  if (pending) {
    return pending
  }

  const request = (async () => {
    try {
      const result = await collectionsProxy.GetCollectionSchema(serverId, dbName, collectionName)
      if (result.isSuccess && result.data) {
        schemaCache.set(key, result.data)
        return result.data
      }
      return null
    } finally {
      pendingRequests.delete(key)
    }
  })()

  pendingRequests.set(key, request)
  return request
}

export async function getCollectionNames(
  serverId: string,
  dbName: string,
): Promise<string[]> {
  const result = await collectionsProxy.GetCollections(serverId, dbName)
  if (result.isSuccess && result.data) {
    return result.data
  }
  return []
}

export function clearSchemaCache() {
  schemaCache.clear()
}
