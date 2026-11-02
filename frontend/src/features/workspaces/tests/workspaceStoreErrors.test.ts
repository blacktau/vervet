import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import * as workspacesProxy from 'wailsjs/go/api/WorkspacesProxy'
import { useWorkspaceStore } from '../workspaceStore'

const notifierError = vi.hoisted(() => vi.fn())
vi.mock('@/utils/dialog', () => ({ useNotifier: () => ({ error: notifierError }) }))
vi.mock('wailsjs/go/api/WorkspacesProxy', () => ({ GetWorkspaces: vi.fn() }))

describe('workspaceStore errors', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    notifierError.mockClear()
  })

  it('leads with the real error, not a headline guessed from its text', async () => {
    const detail = 'open /home/x/workspaces.yaml: connection refused'
    vi.mocked(workspacesProxy.GetWorkspaces).mockResolvedValue({
      isSuccess: false,
      errorCode: 'connection_failed',
      errorDetail: detail,
    } as never)

    await useWorkspaceStore().loadWorkspaces()

    expect(notifierError).toHaveBeenCalledWith(detail)
  })
})
