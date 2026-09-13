<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useInstanceStore } from '../stores/instances'

const route = useRoute()
const instances = useInstanceStore()

const selected = ref('')
const search = ref('')
const streamFilter = ref('all')
const autoScroll = ref(true)
const logView = ref(null)

const logs = computed(() => instances.logs[selected.value] || [])

const filtered = computed(() => {
  let list = logs.value
  if (streamFilter.value !== 'all') {
    list = list.filter((l) => l.stream === streamFilter.value)
  }
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter((l) => (l.line || '').toLowerCase().includes(q))
  }
  return list
})

onMounted(async () => {
  await instances.load()
  const id = route.params.id || instances.items[0]?.id || ''
  selected.value = id
  if (id) await instances.loadLogs(id)
})

watch(selected, async (id) => {
  if (id) await instances.loadLogs(id)
})

watch(
  () => filtered.value.length,
  async () => {
    if (!autoScroll.value) return
    await nextTick()
    if (logView.value) logView.value.scrollTop = logView.value.scrollHeight
  }
)

async function clear() {
  if (!selected.value) return
  await instances.clearLogs(selected.value)
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>日志</h1>
    </div>

    <div class="log-toolbar">
      <select v-model="selected" style="min-width: 220px">
        <option value="">请选择实例</option>
        <option v-for="i in instances.items" :key="i.id" :value="i.id">
          {{ i.name }}
        </option>
      </select>
      <select v-model="streamFilter">
        <option value="all">全部</option>
        <option value="stdout">stdout</option>
        <option value="stderr">stderr</option>
        <option value="system">system</option>
      </select>
      <input v-model="search" type="text" placeholder="搜索…" style="flex: 1" />
      <label class="hint">
        <input v-model="autoScroll" type="checkbox" /> 自动滚动
      </label>
      <button class="danger" @click="clear">清空</button>
    </div>

    <div ref="logView" class="log-view">
      <div v-if="!selected" class="hint">请选择实例</div>
      <div v-else-if="filtered.length === 0" class="hint">暂无日志</div>
      <div v-for="(l, idx) in filtered" :key="idx" class="log-line" :class="l.stream">
        <span class="stream">[{{ l.stream }}]</span><span class="text">{{ l.line }}</span>
      </div>
    </div>
  </div>
</template>
