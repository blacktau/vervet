import { setActivePinia, createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('wailsjs/go/api/BuildInfoProxy', () => ({
  GetChannel: vi.fn(),
}))

import { GetChannel } from 'wailsjs/go/api/BuildInfoProxy'
import { useBuildInfoStore } from './buildInfoStore'

describe('buildInfoStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('defaults to github before load', () => {
    const store = useBuildInfoStore()
    expect(store.channel).toBe('github')
    expect(store.isMSStore).toBe(false)
  })

  it('loads channel from backend', async () => {
    ;(GetChannel as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: 'msstore',
    })
    const store = useBuildInfoStore()
    await store.load()
    expect(store.channel).toBe('msstore')
    expect(store.isMSStore).toBe(true)
  })

  it('keeps github default on failure', async () => {
    ;(GetChannel as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: false,
      errorDetail: 'oops',
    })
    const store = useBuildInfoStore()
    await store.load()
    expect(store.channel).toBe('github')
    expect(store.isMSStore).toBe(false)
  })
})
