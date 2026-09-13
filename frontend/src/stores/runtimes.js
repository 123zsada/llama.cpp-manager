import { defineStore } from 'pinia'
import api from '../api'

export const useRuntimeStore = defineStore('runtimes', {
  state: () => ({
    items: [],
    releases: [],
    params: {},
    loading: false,
    progress: null,
  }),
  getters: {
    byId: (state) => (id) => state.items.find((r) => r.id === id),
  },
  actions: {
    async load() {
      this.items = await api.listRuntimes()
    },
    async fetchReleases() {
      this.loading = true
      try {
        this.releases = await api.fetchAvailableReleases()
      } finally {
        this.loading = false
      }
    },
    async download(tag, backend) {
      this.progress = { tag, backend, percent: 0 }
      try {
        const rt = await api.downloadRuntime(tag, backend)
        await this.load()
        return rt
      } finally {
        this.progress = null
      }
    },
    async addLocal(dir) {
      const rt = await api.addLocalRuntime(dir)
      await this.load()
      return rt
    },
    async remove(id) {
      await api.removeRuntime(id)
      await this.load()
    },
    async loadParams(id) {
      if (this.params[id]) return this.params[id]
      const defs = await api.getRuntimeParams(id)
      this.params[id] = defs
      return defs
    },
  },
})
