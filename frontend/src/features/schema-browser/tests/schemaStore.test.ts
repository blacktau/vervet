import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSchemaStore } from '../schemaStore'

vi.mock('bindings/vervet/internal/api/collectionsproxy', () => ({
  SampleSchema: vi.fn(),
  CancelSampleSchema: vi.fn(),
}))

import { SampleSchema, CancelSampleSchema } from 'bindings/vervet/internal/api/collectionsproxy'

describe('schemaStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('starts with no state for a tab', () => {
    const store = useSchemaStore()
    expect(store.stateFor('tab-1')).toBeUndefined()
  })

  it('sample populates state on success', async () => {
    ;(SampleSchema as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: true,
      data: { sampledCount: 3, totalCount: 3, fields: [] },
    })
    const store = useSchemaStore()
    await store.sample('tab-1', 's1', 'db', 'coll', 100)

    const s = store.stateFor('tab-1')!
    expect(s.loading).toBe(false)
    expect(s.result?.sampledCount).toBe(3)
  })

  it('sample keeps the detail beneath a recognised error code', async () => {
    ;(SampleSchema as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: false,
      errorCode: 'not_authorized',
      errorDetail: 'not allowed on db.coll',
    })
    const store = useSchemaStore()
    await store.sample('tab-1', 's1', 'db', 'coll', 100)
    expect(store.stateFor('tab-1')!.error).toContain('not allowed on db.coll')
    expect(store.stateFor('tab-1')!.error).toContain('\n')
  })

  it('sample sets error on failure', async () => {
    ;(SampleSchema as unknown as ReturnType<typeof vi.fn>).mockResolvedValue({
      isSuccess: false,
      errorDetail: 'nope',
    })
    const store = useSchemaStore()
    await store.sample('tab-1', 's1', 'db', 'coll', 100)
    expect(store.stateFor('tab-1')!.error).toBe('nope')
  })

  it('cancel calls CancelSampleSchema with active requestId', async () => {
    ;(SampleSchema as unknown as ReturnType<typeof vi.fn>).mockImplementation(
      () => new Promise(() => {}),
    )
    const store = useSchemaStore()
    void store.sample('tab-1', 's1', 'db', 'coll', 100)
    await Promise.resolve()
    const reqId = store.stateFor('tab-1')!.requestId!
    store.cancel('tab-1')
    expect(CancelSampleSchema).toHaveBeenCalledWith(reqId)
  })
})
