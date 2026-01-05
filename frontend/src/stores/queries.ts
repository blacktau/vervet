import { defineStore } from 'pinia'

type Query = {
  name: string,
  connectionId: string,
  serverName: string,
  query: string
}

export const useQueryStore = defineStore('query',{
  state: () => ({
    queries: [] as Query[]
  }),
  actions: {

  },
  getters: {

  }
})
