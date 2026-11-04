// @vitest-environment happy-dom
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'

type Listener = () => Promise<void>
const listeners = new Map<string, Listener>()
vi.mock('wailsjs/runtime/runtime', () => ({
  EventsOn: (name: string, listener: Listener) => listeners.set(name, listener),
}))

vi.mock('wailsjs/go/api/CloseGuardProxy', () => ({
  SetUnsavedChanges: vi.fn(async () => ({ isSuccess: true })),
  AcknowledgeCloseRequest: vi.fn(async () => ({ isSuccess: true })),
  CancelClose: vi.fn(async () => ({ isSuccess: true })),
  ConfirmClose: vi.fn(async () => ({ isSuccess: true })),
}))

const tabStore = reactive({
  hasUnsavedQueries: false,
  resolveAllUnsavedQueries: vi.fn(async () => true),
})
vi.mock('@/features/tabs/tabs', () => ({ useTabStore: () => tabStore }))

import * as closeGuardProxy from 'wailsjs/go/api/CloseGuardProxy'
import { useAppCloseGuard } from '@/features/tabs/useAppCloseGuard'

async function requestClose() {
  await listeners.get('app-close-requested')!()
  await flushPromises()
}

describe('useAppCloseGuard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listeners.clear()
    tabStore.hasUnsavedQueries = false
    tabStore.resolveAllUnsavedQueries = vi.fn(async () => true)
  })

  it('reports the current state straight away, then every change', async () => {
    useAppCloseGuard()
    expect(closeGuardProxy.SetUnsavedChanges).toHaveBeenLastCalledWith(false)

    tabStore.hasUnsavedQueries = true
    await nextTick()
    expect(closeGuardProxy.SetUnsavedChanges).toHaveBeenLastCalledWith(true)
  })

  it('acknowledges the request before asking the user', async () => {
    const order: string[] = []
    vi.mocked(closeGuardProxy.AcknowledgeCloseRequest).mockImplementation(async () => {
      order.push('acknowledge')
      return { isSuccess: true } as never
    })
    tabStore.resolveAllUnsavedQueries = vi.fn(async () => {
      order.push('ask')
      return true
    })
    useAppCloseGuard()

    await requestClose()

    expect(order).toEqual(['acknowledge', 'ask'])
  })

  it('closes once the unsaved queries are dealt with', async () => {
    useAppCloseGuard()

    await requestClose()

    expect(closeGuardProxy.ConfirmClose).toHaveBeenCalledOnce()
    expect(closeGuardProxy.CancelClose).not.toHaveBeenCalled()
  })

  it('can be asked again if something goes wrong while asking', async () => {
    tabStore.resolveAllUnsavedQueries = vi.fn(async () => {
      throw new Error('bridge down')
    })
    useAppCloseGuard()

    await requestClose()

    expect(closeGuardProxy.CancelClose).toHaveBeenCalledOnce()
    expect(closeGuardProxy.ConfirmClose).not.toHaveBeenCalled()
  })

  it('stays open if the user cancels', async () => {
    tabStore.resolveAllUnsavedQueries = vi.fn(async () => false)
    useAppCloseGuard()

    await requestClose()

    expect(closeGuardProxy.CancelClose).toHaveBeenCalledOnce()
    expect(closeGuardProxy.ConfirmClose).not.toHaveBeenCalled()
  })
})
