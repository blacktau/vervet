import { describe, expect, it, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUpdateStore } from './updateStore'

vi.mock('wailsjs/go/api/UpdatesProxy', () => ({
  CheckNow: vi.fn(),
  DismissUpdate: vi.fn(),
  OpenReleasePage: vi.fn(),
}))

vi.mock('wailsjs/runtime/runtime', () => ({
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
}))

describe('updateStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('applies update-available event payload', () => {
    const store = useUpdateStore()
    store.applyEvent({ available: true, version: '2026.05.1', url: 'https://x', releaseNotes: '' })
    expect(store.available).toBe(true)
    expect(store.version).toBe('2026.05.1')
  })

  it('dismiss clears available', async () => {
    const { DismissUpdate } = await import('wailsjs/go/api/UpdatesProxy')
    ;(DismissUpdate as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({ isSuccess: true })
    const store = useUpdateStore()
    store.applyEvent({ available: true, version: '2026.05.1', url: 'https://x', releaseNotes: '' })
    await store.dismiss()
    expect(store.available).toBe(false)
  })
})
