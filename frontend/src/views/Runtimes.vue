<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRuntimeStore } from '../stores/runtimes'
import api from '../api'

const store = useRuntimeStore()

const showAdd = ref(false)
const addTab = ref('download')
const backend = ref('cuda')
const selectedTag = ref('')
const localDir = ref('')
const error = ref('')
const busy = ref(false)

const showParams = ref(false)
const paramsTitle = ref('')
const params = ref([])

const missing = ref({})
const repairingId = ref('')
const orphans = ref([])

const groupedParams = computed(() => {
  const map = {}
  for (const p of params.value) {
    if (!map[p.group]) map[p.group] = []
    map[p.group].push(p)
  }
  return Object.entries(map).map(([name, list]) => ({ name, list }))
})

onMounted(async () => {
  await store.load()
  await refreshMissing()
  await refreshOrphans()
})

async function refreshOrphans() {
  try {
    orphans.value = await api.listOrphanRuntimes()
  } catch (e) {
    orphans.value = []
  }
}

async function deleteOrphan(o) {
  if (!confirm(`确定删除未注册目录「${o.name}」及其全部文件？`)) return
  error.value = ''
  try {
    await api.deleteOrphanRuntime(o.name)
    await refreshOrphans()
  } catch (e) {
    error.value = String(e)
  }
}

async function refreshMissing() {
  const next = {}
  for (const rt of store.items) {
    if (rt.backend !== 'cuda') continue
    try {
      next[rt.id] = await api.checkRuntime(rt.id)
    } catch (e) {
      next[rt.id] = []
    }
  }
  missing.value = next
}

async function repair(rt) {
  repairingId.value = rt.id
  error.value = ''
  try {
    await api.repairRuntime(rt.id)
    await store.load()
    await refreshMissing()
    await refreshOrphans()
  } catch (e) {
    error.value = String(e)
  } finally {
    repairingId.value = ''
    store.progress = null
  }
}

function phaseLabel(phase) {
  return { main: '主程序', cudart: 'CUDA 运行库' }[phase] || phase || ''
}

async function openAdd() {
  error.value = ''
  showAdd.value = true
  if (store.releases.length === 0) {
    try {
      await store.fetchReleases()
    } catch (e) {
      error.value = String(e)
    }
  }
}

async function download() {
  if (!selectedTag.value) {
    error.value = '请选择要下载的版本'
    return
  }
  busy.value = true
  error.value = ''
  try {
    await store.download(selectedTag.value, backend.value)
    showAdd.value = false
    await refreshMissing()
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}

async function pickDir() {
  const dir = await api.selectDirectory()
  if (dir) localDir.value = dir
}

async function addLocal() {
  if (!localDir.value) {
    error.value = '请选择目录'
    return
  }
  busy.value = true
  error.value = ''
  try {
    await store.addLocal(localDir.value)
    showAdd.value = false
    localDir.value = ''
  } catch (e) {
    error.value = String(e)
  } finally {
    busy.value = false
  }
}

async function removeRuntime(rt) {
  if (!confirm(`确定移除版本 ${rt.buildTag} (${rt.backend})？`)) return
  await store.remove(rt.id)
  await refreshMissing()
  await refreshOrphans()
}

async function viewParams(rt) {
  error.value = ''
  try {
    params.value = await store.loadParams(rt.id)
    paramsTitle.value = `${rt.buildTag} (${rt.backend}) 参数表`
    showParams.value = true
  } catch (e) {
    error.value = String(e)
  }
}

function formatDate(v) {
  if (!v) return '-'
  return new Date(v).toLocaleString()
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>版本管理</h1>
      <div class="toolbar">
        <button class="primary" @click="openAdd">添加版本</button>
      </div>
    </div>

    <div v-if="store.progress" class="card" style="margin-bottom: 14px">
      <div>
        正在下载 {{ store.progress.tag }} ({{ store.progress.backend }})
        <span v-if="store.progress.phase">· {{ phaseLabel(store.progress.phase) }}</span>
        — {{ (store.progress.percent || 0).toFixed(1) }}%
      </div>
      <div class="progress">
        <div :style="{ width: (store.progress.percent || 0) + '%' }"></div>
      </div>
    </div>

    <div v-if="error" class="card" style="border-color: #7a2f2f; margin-bottom: 14px">
      {{ error }}
    </div>

    <div v-if="store.items.length === 0" class="empty">
      尚未安装任何版本，点击「添加版本」开始。
    </div>

    <div class="card-grid">
      <div v-for="rt in store.items" :key="rt.id" class="card">
        <h3>
          {{ rt.buildTag }}
          <span class="badge" :class="rt.backend">{{ rt.backend }}</span>
          <span v-if="rt.variant" class="badge">{{ rt.variant }}</span>
        </h3>
        <div class="meta">
          <div>来源：{{ rt.source === 'download' ? '在线下载' : '本地目录' }}</div>
          <div>架构：{{ rt.arch }}</div>
          <div>安装于：{{ formatDate(rt.installedAt) }}</div>
          <div>路径：{{ rt.executable }}</div>
        </div>
        <div
          v-if="missing[rt.id] && missing[rt.id].length"
          class="hint"
          style="color: var(--yellow); margin-top: 10px"
        >
          ⚠ 缺少运行库：{{ missing[rt.id].join('、') }}
        </div>
        <div class="toolbar" style="margin-top: 12px">
          <button
            v-if="missing[rt.id] && missing[rt.id].length"
            class="primary"
            :disabled="repairingId === rt.id"
            @click="repair(rt)"
          >
            {{ repairingId === rt.id ? '补全中…' : '补全运行库' }}
          </button>
          <button @click="viewParams(rt)">参数表</button>
          <button class="danger" @click="removeRuntime(rt)">移除</button>
        </div>
      </div>
    </div>

    <div v-if="orphans.length" style="margin-top: 26px">
      <h2 style="font-size: 16px">未注册目录</h2>
      <p class="hint">这些目录位于 runtimes 下但未在版本列表中，可安全删除以回收磁盘。</p>
      <table>
        <thead>
          <tr>
            <th>目录名</th>
            <th>路径</th>
            <th style="width: 120px">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in orphans" :key="o.name">
            <td>{{ o.name }}</td>
            <td class="hint" style="word-break: break-all">{{ o.path }}</td>
            <td><button class="danger" @click="deleteOrphan(o)">删除</button></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showAdd" class="modal-mask" @click.self="showAdd = false">
      <div class="modal">
        <h2>添加版本</h2>
        <div class="toolbar" style="margin-bottom: 14px">
          <button :class="{ primary: addTab === 'download' }" @click="addTab = 'download'">
            从 GitHub 下载
          </button>
          <button :class="{ primary: addTab === 'local' }" @click="addTab = 'local'">
            本地目录
          </button>
        </div>

        <div v-if="addTab === 'download'">
          <div class="row">
            <div class="field">
              <label>后端</label>
              <select v-model="backend">
                <option value="cuda">CUDA</option>
                <option value="vulkan">Vulkan</option>
                <option value="cpu">CPU</option>
                <option value="avx2">AVX2</option>
              </select>
            </div>
            <div class="field">
              <label>版本</label>
              <select v-model="selectedTag">
                <option value="">请选择</option>
                <option v-for="r in store.releases" :key="r.tagName" :value="r.tagName">
                  {{ r.tagName }} — {{ new Date(r.publishedAt).toLocaleDateString() }}
                </option>
              </select>
            </div>
          </div>
          <p class="hint">资产命名：llama-&lt;tag&gt;-bin-win-&lt;backend&gt;-x64.zip</p>
        </div>

        <div v-else>
          <div class="field">
            <label>构建目录（包含 llama-server.exe）</label>
            <div class="row">
              <input v-model="localDir" type="text" placeholder="C:\\path\\to\\llama.cpp" />
              <button style="flex: 0 0 auto" @click="pickDir">浏览…</button>
            </div>
          </div>
        </div>

        <div class="actions">
          <button @click="showAdd = false">取消</button>
          <button
            class="primary"
            :disabled="busy"
            @click="addTab === 'download' ? download() : addLocal()"
          >
            {{ busy ? '处理中…' : '确定' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showParams" class="modal-mask" @click.self="showParams = false">
      <div class="modal" style="width: 720px">
        <h2>{{ paramsTitle }}</h2>
        <div v-for="g in groupedParams" :key="g.name" style="margin-bottom: 16px">
          <h3 style="color: var(--accent)">{{ g.name }} ({{ g.list.length }})</h3>
          <table>
            <thead>
              <tr>
                <th style="width: 200px">参数</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="p in g.list" :key="p.flag">
                <td><code>{{ p.flag }}{{ p.alias ? ', ' + p.alias : '' }}</code></td>
                <td>{{ p.desc }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="actions">
          <button class="primary" @click="showParams = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>
