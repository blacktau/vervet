import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { DataNodeType } from '@/features/data-browser/types.ts'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'
import { api } from 'wailsjs/go/models.ts'

vi.mock('wailsjs/go/api/ConnectionsProxy', () => ({}))

vi.mock('wailsjs/go/api/DatabasesProxy', () => ({
  GetDatabases: vi.fn(),
}))

vi.mock('wailsjs/go/api/CollectionsProxy', () => ({
  GetCollections: vi.fn(),
  GetViews: vi.fn(),
  GetNamespaceInventory: vi.fn(),
}))

const openQueryMock = vi.fn()

vi.mock('@/features/tabs/tabs.ts', () => ({
  useTabStore: vi.fn(() => ({
    removeAllTabs: vi.fn(),
    removeTabById: vi.fn(),
    openQuery: openQueryMock,
    currentTabId: 'server1',
  })),
}))

vi.mock('@/features/settings/settingsStore.ts', () => ({
  useSettingsStore: vi.fn(() => ({
    query: { defaultLimit: 42 },
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

  describe('buildInventory', () => {
    const inventoryResult = {
      isSuccess: true,
      data: {
        serverID: 'server1',
        databases: [
          { name: 'db1', collections: ['users', 'orders'], views: ['activeUsers'] },
          { name: 'db2', collections: ['logs'], views: [] },
        ],
      },
    }

    test('populates connection databases from the inventory', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      vi.mocked(collectionsProxy.GetNamespaceInventory).mockResolvedValue(inventoryResult as never)

      await store.buildInventory('server1')

      const connection = store.connections[0] as never as {
        databases: { name: string; collections: { name: string }[]; views: string[] }[]
      }
      expect(connection.databases).toHaveLength(2)
      expect(connection.databases[0]!.name).toBe('db1')
      expect(connection.databases[0]!.collections.map((c) => c.name)).toEqual(['users', 'orders'])
      expect(connection.databases[0]!.views).toEqual(['activeUsers'])
    })

    test('sets status to ready on success', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      vi.mocked(collectionsProxy.GetNamespaceInventory).mockResolvedValue(inventoryResult as never)

      await store.buildInventory('server1')

      expect(store.getInventoryStatus('server1')).toBe('ready')
    })

    test('sets status to error and leaves databases untouched on failure', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      vi.mocked(collectionsProxy.GetNamespaceInventory).mockResolvedValue({
        isSuccess: false,
        errorCode: 'boom',
        errorDetail: 'nope',
      } as never)

      await store.buildInventory('server1')

      expect(store.getInventoryStatus('server1')).toBe('error')
      expect((store.connections[0] as never as { databases?: unknown }).databases).toBeUndefined()
    })

    test('serves collections from the inventory cache without a backend call', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      vi.mocked(collectionsProxy.GetNamespaceInventory).mockResolvedValue(inventoryResult as never)

      await store.buildInventory('server1')
      const collections = await store.getCollectionList('server1', 'db1')

      expect(collectionsProxy.GetCollections).not.toHaveBeenCalled()
      expect(collections.map((c) => c.name)).toEqual(['users', 'orders'])
    })

    test('reports an unknown server as idle', () => {
      const store = useDataBrowserStore()
      expect(store.getInventoryStatus('nope')).toBe('idle')
    })

    test('a superseded response does not clobber a newer one', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never

      type Deferred = { promise: Promise<unknown>; resolve: (value: unknown) => void }
      const deferred = (): Deferred => {
        let resolve!: (value: unknown) => void
        const promise = new Promise((res) => {
          resolve = res
        })
        return { promise, resolve }
      }

      const first = deferred()
      const second = deferred()

      vi.mocked(collectionsProxy.GetNamespaceInventory)
        .mockReturnValueOnce(first.promise as never)
        .mockReturnValueOnce(second.promise as never)

      const firstCall = store.buildInventory('server1')
      const secondCall = store.buildInventory('server1')

      // Second call resolves first, with the newer data.
      second.resolve({
        isSuccess: true,
        data: { serverID: 'server1', databases: [{ name: 'newDb', collections: ['a'], views: [] }] },
      })
      await secondCall

      // First call resolves after, with stale data. It must be dropped.
      first.resolve({
        isSuccess: true,
        data: { serverID: 'server1', databases: [{ name: 'oldDb', collections: [], views: [] }] },
      })
      await firstCall

      const connection = store.connections[0] as never as {
        databases: { name: string }[]
      }
      expect(connection.databases.map((d) => d.name)).toEqual(['newDb'])
      expect(store.getInventoryStatus('server1')).toBe('ready')
    })
  })

  describe('selection and query opening', () => {
    test('setSelectedKeys stores keys against the active server', () => {
      const store = useDataBrowserStore()
      store.getOrCreateTreeState('server1')

      store.setSelectedKeys(['server1:db1'])

      expect(store.currentSelectedKeys).toEqual(['server1:db1'])
    })

    test('currentSelectedKeys is empty for a server with no tree state', () => {
      const store = useDataBrowserStore()
      expect(store.currentSelectedKeys).toEqual([])
    })

    test('openQueryForKey opens a database query for a database key', () => {
      const store = useDataBrowserStore()
      store.openQueryForKey('server1:db1', DataNodeType.Database)

      expect(openQueryMock).toHaveBeenCalledWith('server1', 'db1')
    })

    test('openQueryForKey opens a find query for a collection key', () => {
      const store = useDataBrowserStore()
      store.openQueryForKey('server1:db1:Collections:users', DataNodeType.Collection)

      expect(openQueryMock).toHaveBeenCalledWith(
        'server1',
        'db1',
        expect.stringContaining("db.getCollection('users').find({})"),
        'users',
      )
    })
  })

  describe('revealNode', () => {
    test('expands the ancestor chain then selects the target', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      store.getOrCreateTreeState('server1')

      const expanded: string[] = []
      store.expandNode = vi.fn(async (_serverId: string, key: string) => {
        expanded.push(key)
      }) as never

      await store.revealNode('server1', 'server1:db1:Collections:users')

      expect(expanded).toEqual(['server1', 'server1:db1', 'server1:db1:Collections'])
      expect(store.currentSelectedKeys).toEqual(['server1:db1:Collections:users'])
    })

    test('selects a database key after expanding only the server', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      store.getOrCreateTreeState('server1')

      const expanded: string[] = []
      store.expandNode = vi.fn(async (_serverId: string, key: string) => {
        expanded.push(key)
      }) as never

      await store.revealNode('server1', 'server1:db1')

      expect(expanded).toEqual(['server1'])
      expect(store.currentSelectedKeys).toEqual(['server1:db1'])
    })

    test('failed expansion does not mark a branch expanded', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      store.getOrCreateTreeState('server1')

      // Stub expandNode to do nothing (simulate failed load)
      store.expandNode = vi.fn(async () => {
        // Do nothing - leaves loadedKeys empty
      }) as never

      await store.revealNode('server1', 'server1:db1:Collections:users')

      const state = store.getOrCreateTreeState('server1')
      // Failed branches should NOT be marked as expanded
      expect(state.expandedKeys).not.toContain('server1')
      expect(state.expandedKeys).not.toContain('server1:db1')
      expect(state.expandedKeys).not.toContain('server1:db1:Collections')
    })

    test('a previously-loaded but collapsed branch is re-expanded', async () => {
      const store = useDataBrowserStore()
      store.connections = [{ serverID: 'server1', name: 'Test Server' }] as never
      const state = store.getOrCreateTreeState('server1')

      // Pre-populate loadedKeys with ancestor keys, but leave expandedKeys empty
      state.loadedKeys = ['server1', 'server1:db1', 'server1:db1:Collections']
      state.expandedKeys = []

      // Stub expandNode to do nothing (the branches are already loaded)
      store.expandNode = vi.fn(async () => {
        // Do nothing
      }) as never

      await store.revealNode('server1', 'server1:db1:Collections:users')

      // Previously-loaded but collapsed branches should be re-expanded
      expect(state.expandedKeys).toContain('server1')
      expect(state.expandedKeys).toContain('server1:db1')
      expect(state.expandedKeys).toContain('server1:db1:Collections')
    })
  })
})
