// @vitest-environment happy-dom
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useTabStore } from '@/features/tabs/tabs'
import { useQueryStore } from '@/features/queries/queryStore'
import { completionsAt, labelsAt, filterTextFor } from './monacoHarness'

// The schema cache talks to the backend over wailsjs. Everything it returns is
// fixed here so the tests exercise the provider, not the transport.
vi.mock('@/features/completion/useSchemaCache', () => ({
  getCollectionNames: vi.fn(async () => ['users', 'orders', 'userEvents']),
  getDatabaseNames: vi.fn(async () => ['shop', 'analytics']),
  getCollectionSchema: vi.fn(async () => ({
    fields: [
      { path: 'name', types: ['string'] },
      { path: 'age', types: ['int'] },
      {
        path: 'address',
        types: ['object'],
        children: [
          { path: 'city', types: ['string'] },
          { path: 'postcode', types: ['string'] },
        ],
      },
    ],
  })),
}))

const QUERY_ID = 'query-1'

beforeEach(() => {
  setActivePinia(createPinia())
  const tabStore = useTabStore()
  tabStore.tabItems = [
    {
      serverId: 'server-1',
      name: 'Server 1',
      queries: [],
      innerTabOrder: [],
      activeInnerTabId: QUERY_ID,
    },
  ] as unknown as typeof tabStore.tabItems
  tabStore.activeTabIndex = 0
  useQueryStore().initQueryState(QUERY_ID, 'shop')
})

describe('collection names', () => {
  it('suggests collections after db.', async () => {
    expect(await labelsAt('db.|')).toEqual(expect.arrayContaining(['users', 'orders']))
  })

  it('filters collections by the typed prefix', async () => {
    const labels = await labelsAt('db.user|')
    expect(labels).toEqual(expect.arrayContaining(['users', 'userEvents']))
    expect(labels).not.toContain('orders')
  })

  it('offers db-level methods alongside collections', async () => {
    expect(await labelsAt('db.|')).toContain('getCollection')
  })

  it('suggests collections inside db.getCollection("', async () => {
    const labels = await labelsAt('db.getCollection("|')
    expect(labels).toEqual(expect.arrayContaining(['users', 'orders']))
  })
})

describe('methods', () => {
  it('suggests collection methods after db.users.', async () => {
    expect(await labelsAt('db.users.|')).toEqual(expect.arrayContaining(['find', 'aggregate']))
  })

  it('filters methods by prefix', async () => {
    const labels = await labelsAt('db.users.fi|')
    expect(labels).toContain('find')
    expect(labels).not.toContain('aggregate')
  })

  it('suggests cursor methods after a find()', async () => {
    expect(await labelsAt('db.users.find({}).|')).toEqual(expect.arrayContaining(['limit', 'sort']))
  })

  it('resolves the collection through db.getCollection', async () => {
    expect(await labelsAt('db.getCollection("users").|')).toContain('find')
  })
})

describe('field names', () => {
  it('suggests schema fields inside a filter', async () => {
    expect(await labelsAt('db.users.find({ |')).toEqual(
      expect.arrayContaining(['name', 'age', 'address']),
    )
  })

  it('flattens nested fields to dotted paths', async () => {
    expect(await labelsAt('db.users.find({ |')).toEqual(
      expect.arrayContaining(['address.city', 'address.postcode']),
    )
  })

  it('quotes the field when the caret is not already inside quotes', async () => {
    const items = await completionsAt('db.users.find({ na|')
    expect(items.find((i) => i.label === 'name')?.insertText).toBe('"name": ')
  })

  it('does not double-quote when the caret is inside quotes', async () => {
    const items = await completionsAt('db.users.find({ "na|')
    expect(items.find((i) => i.label === 'name')?.insertText).toBe('name')
  })

  it('replaces the whole dotted path when completing inside quotes', async () => {
    const source = 'db.users.find({ "address.ci|'
    const items = await completionsAt(source)
    const city = items.find((i) => i.label === 'address.city')
    expect(city).toBeDefined()
    // The replaced range must cover the full path, not just the last segment,
    // or accepting the item yields "address.address.city".
    expect(filterTextFor(source, city!)).toBe('address.ci')
  })
})

describe('operators and stages', () => {
  it('suggests query operators after a field', async () => {
    expect(await labelsAt('db.users.find({ age: { |')).toEqual(
      expect.arrayContaining(['$gt', '$lt']),
    )
  })

  it('suggests aggregation stages inside an aggregate pipeline', async () => {
    expect(await labelsAt('db.users.aggregate([{ |')).toEqual(
      expect.arrayContaining(['$match', '$group']),
    )
  })

  it('suggests update operators in an update document', async () => {
    expect(await labelsAt('db.users.updateOne({}, { |')).toEqual(
      expect.arrayContaining(['$set', '$inc']),
    )
  })
})

describe('databases and keywords', () => {
  it('suggests databases after use', async () => {
    expect(await labelsAt('use |')).toEqual(expect.arrayContaining(['shop', 'analytics']))
  })

  it('filters databases by prefix', async () => {
    const labels = await labelsAt('use sh|')
    expect(labels).toContain('shop')
    expect(labels).not.toContain('analytics')
  })

  it('suggests top-level keywords on an empty line', async () => {
    expect(await labelsAt('|')).toEqual(expect.arrayContaining(['db', 'use', 'EJSON']))
  })

  it('suggests EJSON methods', async () => {
    expect(await labelsAt('EJSON.|')).toEqual(expect.arrayContaining(['stringify', 'parse']))
  })
})

describe('multi-line scripts', () => {
  it('uses the context on the caret line, not the first line', async () => {
    expect(await labelsAt('db.orders.find({})\ndb.users.|')).toContain('find')
  })

  it('picks up a use statement from an earlier line', async () => {
    expect(await labelsAt('use shop\ndb.|')).toEqual(expect.arrayContaining(['users', 'orders']))
  })
})

describe('replacement ranges', () => {
  // Monaco filters the suggest list against the model text under each item's
  // range. If the range excludes what the user typed, correct items get scored
  // out and the list looks empty — the classic "completions stopped working".
  it('covers the typed word so Monaco filters against it', async () => {
    const source = 'db.users.fi|'
    const find = (await completionsAt(source)).find((i) => i.label === 'find')
    expect(filterTextFor(source, find!)).toBe('fi')
  })

  it('covers the typed prefix for collection names', async () => {
    const source = 'db.user|'
    const users = (await completionsAt(source)).find((i) => i.label === 'users')
    expect(filterTextFor(source, users!)).toBe('user')
  })

  it('covers the $ for operators so $-prefixed labels still match', async () => {
    const source = 'db.users.find({ age: { $g|'
    const gt = (await completionsAt(source)).find((i) => i.label === '$gt')
    expect(gt).toBeDefined()
    expect(filterTextFor(source, gt!)).toBe('$g')
  })

  it('covers the $ for aggregation stages', async () => {
    const source = 'db.users.aggregate([{ $ma|'
    const match = (await completionsAt(source)).find((i) => i.label === '$match')
    expect(match).toBeDefined()
    expect(filterTextFor(source, match!)).toBe('$ma')
  })
})

describe('without a selected database', () => {
  // An unknown query id gets a fresh state with no database, so anything that
  // needs the backend comes back empty while the static tables still resolve.
  it('offers no collections', async () => {
    expect(await labelsAt('db.|', 'query-unknown')).not.toContain('users')
  })

  it('offers no fields', async () => {
    expect(await completionsAt('db.users.find({ |', 'query-unknown')).toEqual([])
  })

  it('still offers static method names', async () => {
    expect(await labelsAt('db.users.|', 'query-unknown')).toContain('find')
  })
})
