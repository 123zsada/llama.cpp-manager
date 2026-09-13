<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useInstanceStore } from '../stores/instances'
import { useRuntimeStore } from '../stores/runtimes'

const router = useRouter()
const instances = useInstanceStore()
const runtimes = useRuntimeStore()
const error = ref('')

onMounted(async () => {
  await runtimes.load()
  await instances.load()
})

function runtimeLabel(id) {
  const rt = runtimes.byId(id)
  return rt ? `${rt.buildTag} (${rt.backend})` : id || '-'
}

function runtimeMissing(id) {
  return !!id && !runtimes.byId(id)
}

function statusText(status) {
  return (
    {
      stopped: '已停止',
      starting: '启动中',
      running: '运行中',
      error: '异常',
    }[status] || status
  )
}

async function act(fn) {
  error.value = ''
  try {
    await fn()
  } catch (e) {
    error.value = String(e)
  }
}

async function remove(inst) {
  if (!confirm(`确定删除实例「${inst.name}」？`)) return
  await act(() => instances.remove(inst.id))
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>实例</h1>
      <div class="toolbar">
        <button class="primary" @click="router.push('/instances/new')">新建实例</button>
      </div>
    </div>

    <div v-if="error" class="card" style="border-color: #7a2f2f; margin-bottom: 14px">
      {{ error }}
    </div>

    <div v-if="instances.items.length === 0" class="empty">
      暂无实例，点击「新建实例」创建。
    </div>

    <table v-else>
      <thead>
        <tr>
          <th>名称</th>
          <th>版本</th>
          <th>模型</th>
          <th>状态</th>
          <th style="width: 320px">操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="inst in instances.items" :key="inst.id">
          <td>{{ inst.name }}</td>
          <td>
            <template v-if="runtimeMissing(inst.runtimeId)">
              <span style="color: var(--red)">版本缺失</span>
              <span class="hint">（{{ inst.runtimeId }}）</span>
            </template>
            <template v-else>{{ runtimeLabel(inst.runtimeId) }}</template>
          </td>
          <td class="hint" style="max-width: 260px; word-break: break-all">
            {{ inst.modelPath || '-' }}
          </td>
          <td>
            <span class="status-dot" :class="instances.stateOf(inst.id).status"></span>
            {{ statusText(instances.stateOf(inst.id).status) }}
            <span
              v-if="instances.stateOf(inst.id).pid"
              class="hint"
            >PID {{ instances.stateOf(inst.id).pid }}</span>
          </td>
          <td>
            <div class="toolbar">
              <button
                v-if="instances.stateOf(inst.id).status !== 'running'"
                @click="act(() => instances.start(inst.id))"
              >
                启动
              </button>
              <button
                v-else
                @click="act(() => instances.stop(inst.id))"
              >
                停止
              </button>
              <button @click="act(() => instances.restart(inst.id))">重启</button>
              <button @click="router.push(`/instances/${inst.id}/edit`)">编辑</button>
              <button @click="router.push(`/logs/${inst.id}`)">日志</button>
              <button class="danger" @click="remove(inst)">删除</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
