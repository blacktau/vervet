import { defineStore } from 'pinia'
import { ref } from 'vue'
import { SampleSchema, CancelSampleSchema } from 'wailsjs/go/api/CollectionsProxy'
import type { models } from 'wailsjs/go/models'

export interface SchemaTabState {
  loading: boolean
  requestId?: string
  result?: models.CollectionSchema
  error?: string
  sampleSize: number
}

export const useSchemaStore = defineStore('schema-browser', () => {
  const states = ref<Map<string, SchemaTabState>>(new Map())

  function stateFor(tabId: string): SchemaTabState | undefined {
    return states.value.get(tabId)
  }

  async function sample(
    tabId: string,
    serverId: string,
    db: string,
    coll: string,
    size: number,
  ) {
    const requestId = crypto.randomUUID()
    states.value.set(tabId, { loading: true, requestId, sampleSize: size })

    const res = await SampleSchema(serverId, db, coll, size, requestId)
    const cur = states.value.get(tabId)
    if (!cur || cur.requestId !== requestId) {
      return
    }
    if (res.isSuccess) {
      states.value.set(tabId, { loading: false, result: res.data, sampleSize: size })
    } else {
      const detail = (res as unknown as { errorDetail?: string }).errorDetail
      states.value.set(tabId, { loading: false, error: detail ?? 'Sampling failed', sampleSize: size })
    }
  }

  function cancel(tabId: string) {
    const s = states.value.get(tabId)
    if (s?.requestId) {
      void CancelSampleSchema(s.requestId)
    }
  }

  function clear(tabId: string) {
    states.value.delete(tabId)
  }

  return { stateFor, sample, cancel, clear }
})
