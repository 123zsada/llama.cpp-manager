import { defineStore } from 'pinia'
import api from '../api'

function emptyState(id) {
  return { instanceId: id, status: 'stopped', pid: 0, startedAt: null, lastError: '' }
}

export const useInstanceStore = defineStore('instances', {
  state: () => ({
    items: [],
    states: {},
    logs: {},
  }),
  getters: {
    byId: (state) => (id) => state.items.find((i) => i.id === id),
    stateOf: (state) => (id) => state.states[id] || emptyState(id),
  },
  actions: {
    async load() {
      this.items = await api.listInstances()
      const states = await api.getInstanceStates()
      for (const st of states) {
        this.states[st.instanceId] = st
      }
      for (const inst of this.items) {
        if (!this.states[inst.id]) this.states[inst.id] = emptyState(inst.id)
      }
    },
    async create(inst) {
      const created = await api.createInstance(inst)
      await this.load()
      return created
    },
    async update(inst) {
      await api.updateInstance(inst)
      await this.load()
    },
    async remove(id) {
      await api.deleteInstance(id)
      delete this.states[id]
      delete this.logs[id]
      await this.load()
    },
    async start(id) {
      await api.startInstance(id)
    },
    async stop(id) {
      await api.stopInstance(id)
    },
    async restart(id) {
      await api.restartInstance(id)
    },
    applyStatus(payload) {
      this.states[payload.id] = {
        instanceId: payload.id,
        status: payload.status,
        pid: payload.pid,
        lastError: payload.error,
      }
    },
    appendLog(payload) {
      if (!this.logs[payload.id]) this.logs[payload.id] = []
      this.logs[payload.id].push({
        time: new Date().toISOString(),
        line: payload.line,
        stream: payload.stream,
      })
      if (this.logs[payload.id].length > 3000) {
        this.logs[payload.id].splice(0, this.logs[payload.id].length - 3000)
      }
    },
    async loadLogs(id) {
      this.logs[id] = await api.getLogs(id)
    },
    async clearLogs(id) {
      await api.clearLogs(id)
      this.logs[id] = []
    },
  },
})
