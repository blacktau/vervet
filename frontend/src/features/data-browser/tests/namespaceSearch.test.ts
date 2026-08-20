import { describe, expect, test } from 'vitest'
import { DataNodeType } from '@/features/data-browser/types.ts'
import {
  type NamespaceRow,
  scoreMatch,
  searchNamespaces,
} from '@/features/data-browser/namespaceSearch.ts'

const row = (db: string, name: string): NamespaceRow => ({
  serverID: 'srv',
  db,
  name,
  type: DataNodeType.Collection,
  path: `${db}.${name}`,
})

describe('scoreMatch', () => {
  test('returns null when the query is not a subsequence of the target', () => {
    expect(scoreMatch('zzz', 'users.orders')).toBeNull()
  })

  test('matches a subsequence spread across the target', () => {
    expect(scoreMatch('usord', 'users.orders')).not.toBeNull()
  })

  test('is case insensitive', () => {
    expect(scoreMatch('ORD', 'users.orders')).not.toBeNull()
  })

  test('scores a contiguous run above a scattered one', () => {
    const contiguous = scoreMatch('ord', 'users.orders')
    const scattered = scoreMatch('ord', 'o_r_d_x')
    expect(contiguous).not.toBeNull()
    expect(scattered).not.toBeNull()
    expect(contiguous!).toBeGreaterThan(scattered!)
  })

  test('scores a word-boundary match above a mid-word match', () => {
    const boundary = scoreMatch('o', 'users.orders')
    const midWord = scoreMatch('o', 'xxo')
    expect(boundary!).toBeGreaterThan(midWord!)
  })

  test('returns a score for an empty query', () => {
    expect(scoreMatch('', 'anything')).toBe(0)
  })
})

describe('searchNamespaces', () => {
  test('returns nothing for an empty query', () => {
    expect(searchNamespaces('', [row('users', 'orders')])).toEqual([])
  })

  test('matches against the row path', () => {
    const rows = [row('users', 'orders'), row('logs', 'events')]
    const results = searchNamespaces('usrord', rows)
    expect(results).toHaveLength(1)
    expect(results[0]!.name).toBe('orders')
  })

  test('scores a database row against its bare name, not a doubled path', () => {
    const database: NamespaceRow = {
      serverID: 'srv',
      db: 'users',
      name: 'users',
      type: DataNodeType.Database,
      path: 'users',
    }
    const results = searchNamespaces('users', [database])
    expect(results).toHaveLength(1)
    expect(results[0]!.type).toBe(DataNodeType.Database)
  })

  test('ranks the better match first', () => {
    const rows = [row('a', 'xorderx'), row('a', 'orders')]
    const results = searchNamespaces('order', rows)
    expect(results[0]!.name).toBe('orders')
  })

  test('breaks ties by preferring the shorter target', () => {
    const rows = [row('a', 'orders_archive_2024'), row('a', 'orders')]
    const results = searchNamespaces('orders', rows)
    expect(results[0]!.name).toBe('orders')
  })

  test('caps results at the limit', () => {
    const rows = Array.from({ length: 80 }, (_, i) => row('a', `orders${i}`))
    expect(searchNamespaces('orders', rows)).toHaveLength(50)
    expect(searchNamespaces('orders', rows, 10)).toHaveLength(10)
  })
})
