import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'
import { api } from 'wailsjs/go/models.ts'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({}))

vi.mock('wailsjs/go/api/DatabasesProxy', () => ({
  GetDatabases: vi.fn(),
}))

vi.mock('wailsjs/go/api/CollectionsProxy', () => ({
  GetCollections: vi.fn(),
}))

vi.mock('@/features/tabs/tabs.ts', () => ({
  useTabStore: vi.fn(() => ({
    removeAllTabs: vi.fn(),
    removeTabById: vi.fn(),
  })),
}))

vi.mock('@/utils/dialog.ts', () => ({
  useNotifier: vi.fn(() => ({
    error: vi.fn(),
  })),
}))

describe('browserStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('getDatabaseList', () => {
    test('should fetch databases from backend when connection exists', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never

      vi.mocked(databasesProxy.GetDatabases).mockResolvedValue({
        isSuccess: true,
        data: ['db1', 'db2'],
      } as api.Result___string_)

      const result = await store.getDatabaseList('server1', true)

      expect(databasesProxy.GetDatabases).toHaveBeenCalledWith('server1')
      expect(result).toEqual([
        { name: 'db1', collections: [], views: [] },
        { name: 'db2', collections: [], views: [] },
      ])
      expect(store.connections[0]?.databases).toEqual([
        { name: 'db1', collections: [], views: [] },
        { name: 'db2', collections: [], views: [] },
      ])
    })

    test('should return cached databases when not forcing reload', async () => {
      const store = useDataBrowserStore()
      store.connections = [
        {
          serverID: 'server1',
          name: 'Test Server',
          databases: [{ name: 'cachedDb', collections: [] }],
        },
      ] as never

      const result = await store.getDatabaseList('server1', false)

      expect(databasesProxy.GetDatabases).not.toHaveBeenCalled()
      expect(result).toEqual([{ name: 'cachedDb', collections: [] }])
    })

    test('should return empty array when connection not found', async () => {
      const store = useDataBrowserStore()
      store.connections = []

      const result = await store.getDatabaseList('nonexistent', true)

      expect(result).toEqual([])
    })

    test('should return empty array when backend returns error', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never

      vi.mocked(databasesProxy.GetDatabases).mockResolvedValue({
        isSuccess: false,
        error: 'Connection failed',
      } as api.Result___string_)

      const result = await store.getDatabaseList('server1', true)

      expect(result).toEqual([])
    })
  })

  describe('getCollectionList', () => {
    test('should fetch collections from backend when database exists', async () => {
      const store = useDataBrowserStore()
      store.connections = [
        {
          serverID: 'server1',
          name: 'Test Server',
          databases: [{ name: 'db1', collections: [] }],
        },
      ] as never

      vi.mocked(collectionsProxy.GetCollections).mockResolvedValue({
        isSuccess: true,
        data: ['collection1', 'collection2'],
      } as api.Result___string_)

      const result = await store.getCollectionList('server1', 'db1', true)

      expect(collectionsProxy.GetCollections).toHaveBeenCalledWith('server1', 'db1')
      expect(result).toEqual([
        { name: 'collection1', indexes: [] },
        { name: 'collection2', indexes: [] },
      ])
    })

    test('should return cached collections when not forcing reload', async () => {
      const store = useDataBrowserStore()
      store.connections = [
        {
          serverID: 'server1',
          name: 'Test Server',
          databases: [
            {
              name: 'db1',
              collections: [{ name: 'cachedCol', indexes: [] }],
            },
          ],
        },
      ] as never

      const result = await store.getCollectionList('server1', 'db1', false)

      expect(collectionsProxy.GetCollections).not.toHaveBeenCalled()
      expect(result).toEqual([{ name: 'cachedCol', indexes: [] }])
    })
  })

  describe('findDatabase', () => {
    test('should find database by server and database name', () => {
      const store = useDataBrowserStore()
      store.connections = [
        {
          serverID: 'server1',
          name: 'Test Server',
          databases: [{ name: 'db1', collections: [] }],
        },
      ] as never

      const result = store.findDatabase('server1', 'db1')

      expect(result).toEqual({ name: 'db1', collections: [] })
    })

    test('should return undefined when database not found', () => {
      const store = useDataBrowserStore()
      store.connections = [
        {
          serverID: 'server1',
          name: 'Test Server',
          databases: [],
        },
      ] as never

      const result = store.findDatabase('server1', 'nonexistent')

      expect(result).toBeUndefined()
    })
  })
})
