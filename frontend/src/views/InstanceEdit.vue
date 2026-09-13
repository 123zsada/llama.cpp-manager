<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useInstanceStore } from '../stores/instances'
import { useRuntimeStore } from '../stores/runtimes'
import ParamGroup from '../components/ParamGroup.vue'
import ParamItem from '../components/ParamItem.vue'
import CommandPreview from '../components/CommandPreview.vue'
import CommandExplainer from '../components/CommandExplainer.vue'
import api from '../api'

const route = useRoute()
const router = useRouter()
const instances = useInstanceStore()
const runtimes = useRuntimeStore()

const form = reactive({
  id: '',
  name: '',
  runtimeId: '',
  modelPath: '',
  params: {},
  extraArgs: [],
  workDir: '',
  autoStart: false,
})

const extraArgsText = ref('')
const paramDefs = ref([])
const selectedGroup = ref('基本信息')
const explanations = ref([])
const error = ref('')
const saving = ref(false)

const showImport = ref(false)
const importText = ref('')
const importError = ref('')
const importInfo = ref('')

const groups = computed(() => {
  const map = {}
  for (const p of paramDefs.value) {
    if (!map[p.group]) map[p.group] = []
    map[p.group].push(p)
  }
  return map
})

const groupNames = computed(() => Object.keys(groups.value))

const unknownParams = computed(() => {
  if (paramDefs.value.length < 20) return []
  const known = new Set()
  for (const d of paramDefs.value) {
    known.add(d.flag)
    if (d.alias) known.add(d.alias)
  }
  return Object.keys(form.params).filter((k) => !known.has(k))
})

function keepUnknownAsExtra() {
  const existing = extraArgsText.value
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean)
  for (const flag of unknownParams.value) {
    existing.push(flag)
    const v = form.params[flag]
    if (v) existing.push(v)
    delete form.params[flag]
  }
  extraArgsText.value = existing.join('\n')
  refreshExplain()
}

function removeUnknownParams() {
  for (const flag of unknownParams.value) {
    delete form.params[flag]
  }
  refreshExplain()
}

const visibleParams = computed(() => {
  if (selectedGroup.value === '基本信息') return []
  return groups.value[selectedGroup.value] || []
})

const previewCommand = computed(() => {
  const exe = runtimes.byId(form.runtimeId)?.executable || ''
  const args = []
  if (form.modelPath) args.push('-m', form.modelPath)
  const keys = Object.keys(form.params)
    .filter((k) => k !== '-m' && k !== '--model')
    .sort()
  for (const k of keys) {
    args.push(k)
    if (form.params[k]) args.push(form.params[k])
  }
  args.push(...form.extraArgs)
  return [exe, ...args].filter(Boolean).map(quote).join(' ')
})

function quote(s) {
  if (s === '' || /[\s"]/.test(s)) return `"${s}"`
  return s
}

function setParam(flag, value) {
  if (value === null || value === undefined) {
    delete form.params[flag]
  } else {
    form.params[flag] = value
  }
}

function enabledCount(group) {
  const list = groups.value[group] || []
  return list.filter((p) => form.params[p.flag] !== undefined).length
}

async function loadParams(runtimeId) {
  if (!runtimeId) {
    paramDefs.value = []
    return
  }
  try {
    paramDefs.value = await runtimes.loadParams(runtimeId)
  } catch (e) {
    error.value = String(e)
  }
}

watch(
  () => form.runtimeId,
  (id) => loadParams(id)
)

let explainTimer = null
watch(
  () => JSON.stringify({ p: form.params, m: form.modelPath, e: form.extraArgs }),
  () => {
    clearTimeout(explainTimer)
    explainTimer = setTimeout(refreshExplain, 250)
  }
)

async function refreshExplain() {
  try {
    explanations.value = await api.explainCommand({ ...form })
  } catch (e) {
    explanations.value = []
  }
}

onMounted(async () => {
  await runtimes.load()
  const id = route.params.id
  if (id) {
    await instances.load()
    const inst = instances.byId(id)
    if (inst) {
      Object.assign(form, {
        id: inst.id,
        name: inst.name,
        runtimeId: inst.runtimeId,
        modelPath: inst.modelPath,
        params: { ...(inst.params || {}) },
        extraArgs: [...(inst.extraArgs || [])],
        workDir: inst.workDir,
        autoStart: inst.autoStart,
      })
      extraArgsText.value = (inst.extraArgs || []).join('\n')
    }
  } else if (runtimes.items.length > 0) {
    form.runtimeId = runtimes.items[0].id
  }
  await loadParams(form.runtimeId)
  refreshExplain()
})

async function pickModel() {
  const file = await api.selectFile('GGUF 模型 (*.gguf)|*.gguf|所有文件 (*.*)|*.*')
  if (file) form.modelPath = file
}

async function pickWorkDir() {
  const dir = await api.selectDirectory()
  if (dir) form.workDir = dir
}

function openImport() {
  importText.value = ''
  importError.value = ''
  importInfo.value = ''
  showImport.value = true
}

async function doImport() {
  importError.value = ''
  importInfo.value = ''
  const text = importText.value.trim()
  if (!text) {
    importError.value = '请粘贴完整命令'
    return
  }
  try {
    const parsed = await api.parseCommand(text, form.runtimeId)
    form.modelPath = parsed.modelPath || ''
    form.params = { ...(parsed.params || {}) }
    extraArgsText.value = (parsed.extraArgs || []).join('\n')

    const parts = []
    if (parsed.executable) parts.push(`可执行文件：${parsed.executable}`)
    parts.push(`识别到参数 ${Object.keys(form.params).length} 个`)
    if (parsed.extraArgs && parsed.extraArgs.length) {
      parts.push(`附加参数 ${parsed.extraArgs.length} 个`)
    }
    importInfo.value = parts.join('，')
    refreshExplain()
  } catch (e) {
    importError.value = String(e)
  }
}

async function save() {
  if (!form.name) {
    error.value = '请填写实例名称'
    return
  }
  if (!form.runtimeId) {
    error.value = '请选择绑定版本'
    return
  }
  saving.value = true
  error.value = ''
  const payload = {
    ...form,
    extraArgs: extraArgsText.value
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean),
  }
  try {
    if (form.id) {
      await instances.update(payload)
    } else {
      await instances.create(payload)
    }
    router.push('/instances')
  } catch (e) {
    error.value = String(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="page-header">
      <h1>{{ form.id ? '编辑实例' : '新建实例' }}</h1>
      <div class="toolbar">
        <button @click="openImport">从命令导入</button>
        <button @click="router.push('/instances')">返回</button>
        <button class="primary" :disabled="saving" @click="save">
          {{ saving ? '保存中…' : '保存' }}
        </button>
      </div>
    </div>

    <div v-if="error" class="card" style="border-color: #7a2f2f; margin-bottom: 14px">
      {{ error }}
    </div>

    <div
      v-if="unknownParams.length"
      class="card"
      style="border-color: #7a5a2f; margin-bottom: 14px"
    >
      <div style="margin-bottom: 8px">
        以下参数在当前绑定版本中不存在：<code>{{ unknownParams.join(', ') }}</code>
      </div>
      <div class="toolbar">
        <button @click="keepUnknownAsExtra">保留为附加参数</button>
        <button class="danger" @click="removeUnknownParams">移除这些参数</button>
      </div>
    </div>

    <div class="edit-layout">
      <div class="panel">
        <h3>参数分组</h3>
        <ParamGroup
          name="基本信息"
          :count="0"
          :active="selectedGroup === '基本信息'"
          @select="selectedGroup = $event"
        />
        <ParamGroup
          v-for="g in groupNames"
          :key="g"
          :name="g"
          :count="enabledCount(g)"
          :active="selectedGroup === g"
          @select="selectedGroup = $event"
        />
      </div>

      <div class="panel">
        <template v-if="selectedGroup === '基本信息'">
          <h3>基本信息</h3>
          <div class="field">
            <label>实例名称</label>
            <input v-model="form.name" type="text" placeholder="例如：Qwen3-8B" />
          </div>
          <div class="field">
            <label>绑定版本</label>
            <select v-model="form.runtimeId">
              <option value="">请选择</option>
              <option v-for="rt in runtimes.items" :key="rt.id" :value="rt.id">
                {{ rt.buildTag }} ({{ rt.backend }})
              </option>
            </select>
          </div>
          <div class="field">
            <label>模型文件</label>
            <div class="row">
              <input v-model="form.modelPath" type="text" placeholder=".gguf 路径" />
              <button style="flex: 0 0 auto" @click="pickModel">浏览…</button>
            </div>
          </div>
          <div class="field">
            <label>工作目录（可选）</label>
            <div class="row">
              <input v-model="form.workDir" type="text" placeholder="默认使用版本目录" />
              <button style="flex: 0 0 auto" @click="pickWorkDir">浏览…</button>
            </div>
          </div>
          <div class="field">
            <label>额外参数（每行一个）</label>
            <textarea v-model="extraArgsText" rows="4" placeholder="--verbose&#10;--no-webui"></textarea>
          </div>
          <div class="field">
            <label>
              <input v-model="form.autoStart" type="checkbox" />
              随管理器启动
            </label>
          </div>
        </template>

        <template v-else>
          <h3>{{ selectedGroup }}</h3>
          <div v-if="visibleParams.length === 0" class="hint">该分组暂无参数</div>
          <ParamItem
            v-for="p in visibleParams"
            :key="p.flag"
            :def="p"
            :model-value="form.params[p.flag] ?? null"
            @update:model-value="setParam(p.flag, $event)"
          />
        </template>
      </div>

      <div class="panel">
        <h3>命令预览</h3>
        <CommandPreview :command="previewCommand" />
        <h3 style="margin-top: 18px">命令解释</h3>
        <CommandExplainer :explanations="explanations" />
      </div>
    </div>

    <div v-if="showImport" class="modal-mask" @click.self="showImport = false">
      <div class="modal" style="width: 680px">
        <h2>从命令导入</h2>
        <p class="hint">
          粘贴完整的 llama-server 命令行，将自动解析出模型路径、参数与附加参数。
        </p>
        <div class="field">
          <textarea
            v-model="importText"
            rows="6"
            placeholder="llama-server.exe -m D:\models\qwen.gguf -ngl 99 -c 8192 --host 0.0.0.0 --port 8080"
          ></textarea>
        </div>
        <div v-if="importInfo" class="hint" style="color: var(--green)">{{ importInfo }}</div>
        <div v-if="importError" class="hint" style="color: var(--red)">{{ importError }}</div>
        <div class="actions">
          <button @click="showImport = false">关闭</button>
          <button class="primary" @click="doImport">解析并导入</button>
        </div>
      </div>
    </div>
  </div>
</template>
