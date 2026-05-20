import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, test } from 'vitest'

import { useQueryStore } from '@/features/queries/queryStore'

describe('queryStore runStartedAt lifecycle', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  test('new query state starts with runStartedAt = null', () => {
    const store = useQueryStore()
    const state = store.getQueryState('q1')
    expect(state.runStartedAt).toBeNull()
  })
})
