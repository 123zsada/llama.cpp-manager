<script setup>
import { onMounted, onUnmounted } from 'vue'
import { events } from './api'
import { useInstanceStore } from './stores/instances'
import { useRuntimeStore } from './stores/runtimes'
import iconUrl from './assets/images/icon.png'

const instances = useInstanceStore()
const runtimes = useRuntimeStore()

let cancels = []

onMounted(() => {
  cancels.push(events.on('instance:status', (payload) => instances.applyStatus(payload)))
  cancels.push(events.on('instance:log', (payload) => instances.appendLog(payload)))
  cancels.push(
    events.on('runtime:progress', (payload) => {
      runtimes.progress = payload
    })
  )
})

onUnmounted(() => {
  for (const cancel of cancels) {
    if (typeof cancel === 'function') cancel()
  }
  cancels = []
})
</script>

<template>
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <img :src="iconUrl" alt="logo" />
        <span>llama.cpp Manager</span>
      </div>
      <router-link class="nav-link" to="/instances">实例</router-link>
      <router-link class="nav-link" to="/runtimes">版本</router-link>
      <router-link class="nav-link" to="/logs">日志</router-link>
    </aside>
    <main class="main">
      <router-view />
    </main>
  </div>
</template>
