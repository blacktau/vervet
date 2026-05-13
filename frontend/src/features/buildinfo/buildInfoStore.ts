import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { GetChannel } from 'wailsjs/go/api/BuildInfoProxy'

export type Channel = 'github' | 'msstore'

export const useBuildInfoStore = defineStore('buildInfo', () => {
  const channel = ref<Channel>('github')
  const loaded = ref(false)

  async function load() {
    const result = await GetChannel()
    if (result.isSuccess && (result.data === 'github' || result.data === 'msstore')) {
      channel.value = result.data
    }
    loaded.value = true
  }

  const isMSStore = computed(() => channel.value === 'msstore')

  return { channel, isMSStore, loaded, load }
})
