import { describe, expect, it, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUpdateStore } from './updateStore'

vi.mock('bindings/vervet/internal/api/updatesproxy', () => ({
  CheckIfDue: vi.fn(),
  CheckNow: vi.fn(),
  DismissUpdate: vi.fn(),
  InstallUpdate: vi.fn(),
  OpenReleasePage: vi.fn(),
  RestartToUpdate: vi.fn(),
}))

vi.mock('@/utils/wailsEvents', () => ({
  onEvent: vi.fn(() => vi.fn()),
}))

describe('updateStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('applies update-available event payload', () => {
    const store = useUpdateStore()
    store.applyEvent({
      available: true,
      version: '2026.05.1',
      url: 'https://x',
      releaseNotes: '',
      canInstall: false,
    })
    expect(store.available).toBe(true)
    expect(store.version).toBe('2026.05.1')
  })

  it('dismiss clears available', async () => {
    const { DismissUpdate } = await import('bindings/vervet/internal/api/updatesproxy')
    ;(DismissUpdate as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({ isSuccess: true })
    const store = useUpdateStore()
    store.applyEvent({
      available: true,
      version: '2026.05.1',
      url: 'https://x',
      releaseNotes: '',
      canInstall: false,
    })
    await store.dismiss()
    expect(store.available).toBe(false)
  })

  describe('in-app install', () => {
    const info = {
      available: true,
      version: '2026.11.2',
      url: 'https://x',
      releaseNotes: '',
      canInstall: true,
    }

    async function proxy() {
      const m = await import('bindings/vervet/internal/api/updatesproxy')
      return {
        InstallUpdate: m.InstallUpdate as unknown as ReturnType<typeof vi.fn>,
        RestartToUpdate: m.RestartToUpdate as unknown as ReturnType<typeof vi.fn>,
      }
    }

    function eventHandler(name: string) {
      return async () => {
        const { onEvent } = await import('@/utils/wailsEvents')
        const calls = (onEvent as unknown as ReturnType<typeof vi.fn>).mock.calls.filter(
          (c: unknown[]) => c[0] === name,
        )
        const call = calls[calls.length - 1]
        return call?.[1] as (data: unknown) => void
      }
    }

    it('downloads, then is ready to restart', async () => {
      const { InstallUpdate, RestartToUpdate } = await proxy()
      InstallUpdate.mockResolvedValue({ isSuccess: true })
      RestartToUpdate.mockResolvedValue({ isSuccess: true })
      const store = useUpdateStore()
      store.applyEvent(info)

      const pending = store.install()
      expect(store.phase).toBe('downloading')
      await pending
      expect(store.phase).toBe('ready')

      await store.restart()
      expect(RestartToUpdate).toHaveBeenCalledOnce()
    })

    it('reports a failed download', async () => {
      const { InstallUpdate } = await proxy()
      InstallUpdate.mockResolvedValue({ isSuccess: false, errorDetail: 'digest mismatch' })
      const store = useUpdateStore()
      store.applyEvent(info)

      await store.install()
      expect(store.phase).toBe('error')
      expect(store.installError).toContain('digest mismatch')
    })

    it('does nothing when this copy cannot install', async () => {
      const { InstallUpdate } = await proxy()
      InstallUpdate.mockClear()
      const store = useUpdateStore()
      store.applyEvent({ ...info, canInstall: false })

      await store.install()
      expect(InstallUpdate).not.toHaveBeenCalled()
      expect(store.phase).toBe('idle')
    })

    it('tracks download progress and verification from updater events', async () => {
      const { InstallUpdate } = await proxy()
      let finish: (r: { isSuccess: boolean }) => void = () => {}
      InstallUpdate.mockReturnValue(new Promise((resolve) => (finish = resolve)))
      const store = useUpdateStore()
      store.subscribe()
      store.applyEvent(info)
      const pending = store.install()

      ;(await eventHandler('wails:updater:download-progress')())({ written: 25, total: 100 })
      expect(store.progress).toBe(25)
      ;(await eventHandler('wails:updater:download-progress')())({ written: 25, total: 0 })
      expect(store.progress).toBeNull()
      ;(await eventHandler('wails:updater:verifying')())(null)
      expect(store.phase).toBe('verifying')

      finish({ isSuccess: true })
      await pending
      expect(store.phase).toBe('ready')
    })

    // Seen on Linux: the result came back before the verifying event.
    it('stays ready when the verifying event arrives late', async () => {
      const { InstallUpdate } = await proxy()
      InstallUpdate.mockResolvedValue({ isSuccess: true })
      const store = useUpdateStore()
      store.subscribe()
      store.applyEvent(info)

      await store.install()
      ;(await eventHandler('wails:updater:verifying')())(null)
      expect(store.phase).toBe('ready')
    })

    it('reports an install call that throws', async () => {
      const { InstallUpdate } = await proxy()
      InstallUpdate.mockRejectedValue(new Error('connection lost'))
      const store = useUpdateStore()
      store.applyEvent(info)

      await store.install()
      expect(store.phase).toBe('error')
      expect(store.installError).toBe('connection lost')
    })

    it('shows a restart that failed after the unsaved-queries prompt', async () => {
      const store = useUpdateStore()
      store.subscribe()

      ;(await eventHandler('update-restart-failed')())('helper did not start')
      expect(store.phase).toBe('error')
      expect(store.installError).toBe('helper did not start')
    })
  })
})
