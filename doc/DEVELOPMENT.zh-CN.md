# llama.cpp Manager — 开发文档

> 版本：v0.2
> 最后更新：2026-09-13
> 阶段：Phase 1–4 已实现；Phase 5 部分待完成

## 1. 项目概述

**名称**：llama.cpp Manager
**技术栈**：Go + Wails v2 + Vue 3（Pinia + Vue Router）
**目标平台**：Windows x86_64（优先）；后续可扩展其他平台
**目的**：图形化管理多个 llama.cpp 实例——多版本切换、可视化命令构建、进程生命周期管理与日志监控。

## 2. 功能范围

### 2.1 版本管理
- 从 GitHub Releases 下载官方预构建二进制（Windows x86_64）。
- 添加本地已有的构建目录。
- 每个版本独立存放于 `runtimes/<id>/`。
- 后端识别（CUDA / Vulkan / CPU / AVX2）。
- 通过 `llama-server --version` 读取构建号。
- **CUDA 拆分资产支持**：近期版本起，CUDA 主程序与 CUDA 运行时（cudart/cublas）作为**两个**资产发布，两者都会下载并解压到同一目录。
- **驱动感知的 CUDA 选择**：查询 `nvidia-smi`，选择不高于驱动支持版本的最新 CUDA 变体。
- **运行库校验与修复**：检测缺失的 CUDA/Vulkan DLL，在界面告警并可修复（仅补 cudart，或整包重下）。
- **孤儿目录清理**：列出并删除 `runtimes/` 下已不在注册表中的子目录。

### 2.2 实例管理
- 创建多个实例，每个绑定一个版本。
- 操作：启动、停止、重启、删除。
- 实时状态：已停止 / 启动中 / 运行中 / 异常。
- 配置以 JSON 持久化。
- **启动对账**：当存在唯一匹配「构建号 + 后端」的版本时，自动改绑指向已删除版本的实例。
- **启动前预检**：缺失运行库、或使用了绑定版本不支持的参数时，阻止启动并给出明确错误。

### 2.3 可视化命令构建
- 依据所选版本的 `--help` 动态生成参数面板。
- 参数按功能分组：模型 / GPU / 上下文 / 采样 / 服务器 / 性能 / 其他。
- 每个参数提供说明与 tooltip。
- 实时命令预览。
- 完整命令解释器（逐参数解释）。
- **从完整命令行导入**：粘贴完整的 `llama-server` 命令，解析为模型路径、参数与附加参数。

### 2.4 日志与监控
- 实时捕获 `stdout` / `stderr`。
- 每实例日志缓冲，支持过滤 / 搜索 / 清空。
- 每次启动前记录**完整命令行**，便于排查。
- 简单异常检测计划于 Phase 5。

## 3. 技术架构

### 3.1 目录结构

```
llama-manager/
├── main.go                       # Wails 入口
├── app.go                        # App 结构体与绑定
├── fatal_windows.go              # 原生错误对话框（Windows）
├── fatal_other.go                # 致命错误回退（非 Windows）
├── wails.json
├── go.mod
├── .gitignore
├── .gitattributes
├── internal/
│   ├── config/
│   │   └── store.go              # 以程序目录为根的 JSON 持久化
│   ├── runtime/
│   │   ├── runtime.go            # LlamaRuntime + Registry
│   │   ├── download.go           # GitHub Release 获取 / 下载 / 解压
│   │   ├── scan.go               # 本地目录扫描 + 后端识别
│   │   ├── parse_help.go         # --help 解析 + 分组 + 回退表
│   │   ├── cuda.go               # 驱动探测、CUDA 资产选择、运行库校验
│   │   └── orphans.go            # 未注册版本目录管理
│   ├── instance/
│   │   ├── instance.go           # LlamaInstance / InstanceState / 事件
│   │   ├── manager.go            # CRUD、重绑定、对账、日志缓冲
│   │   ├── process.go            # 启动 / 停止 / 重启、预检、日志
│   │   ├── process_windows.go    # CREATE_NO_WINDOW、taskkill /T
│   │   └── process_other.go
│   └── cmd/
│       ├── builder.go            # 由模型 + 参数 + 附加参数构建命令行
│       ├── explainer.go          # 依据 ParamDef 解释参数
│       └── parser.go             # 将完整命令行解析为结构化字段
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   ├── public/
│   │   └── icon.png              # favicon
│   └── src/
│       ├── App.vue               # 布局、侧边栏、事件订阅
│       ├── main.js
│       ├── router.js
│       ├── style.css
│       ├── api/index.js          # Wails 绑定与事件封装
│       ├── views/
│       │   ├── Runtimes.vue      # 版本管理
│       │   ├── Instances.vue     # 实例列表
│       │   ├── InstanceEdit.vue  # 实例编辑 / 命令构建器
│       │   └── Logs.vue          # 日志查看
│       ├── components/
│       │   ├── ParamGroup.vue
│       │   ├── ParamItem.vue
│       │   ├── CommandPreview.vue
│       │   └── CommandExplainer.vue
│       ├── stores/
│       │   ├── runtimes.js
│       │   └── instances.js
│       └── assets/images/icon.png
├── build/
│   ├── appicon.png               # 图标源（1024x1024）
│   ├── windows/icon.ico          # 由 Wails 从 appicon.png 生成
│   └── ...
├── tools/
│   └── genicon.ps1               # 图标生成脚本（PowerShell + System.Drawing）
└── doc/
    └── DEVELOPMENT.md / DEVELOPMENT.zh-CN.md  # 本文档
```

### 3.2 数据流

```
用户操作 (Vue)
    ↓
Wails 绑定（*App 上的 Go 方法）
    ↓
internal/runtime | instance | cmd | config
    ↓
文件系统 / 进程 / 网络
    ↓
事件回推 (runtime.EventsEmit)
    ↓
前端状态更新 (Pinia)
```

## 4. 数据模型

### 4.1 LlamaRuntime（版本）

```go
type LlamaRuntime struct {
    ID          string    `json:"id"`          // 例如 "b10930-cuda-12.4"
    BuildTag    string    `json:"buildTag"`    // "b10930"
    Backend     string    `json:"backend"`     // "cuda" / "vulkan" / "cpu" / "avx2"
    Variant     string    `json:"variant"`     // CUDA 版本，例如 "12.4"
    Arch        string    `json:"arch"`        // "x64"
    Executable  string    `json:"executable"`  // llama-server.exe 绝对路径
    WorkDir     string    `json:"workDir"`
    Source      string    `json:"source"`      // "download" / "local"
    InstalledAt time.Time `json:"installedAt"`
}
```

### 4.2 LlamaInstance（实例）

```go
type LlamaInstance struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    RuntimeID string            `json:"runtimeId"`
    ModelPath string            `json:"modelPath"`
    Params    map[string]string `json:"params"`     // flag -> value（"" 表示仅标志）
    ExtraArgs []string          `json:"extraArgs"`
    WorkDir   string            `json:"workDir"`
    AutoStart bool              `json:"autoStart"`
    CreatedAt time.Time         `json:"createdAt"`
}
```

### 4.3 运行时状态（内存）

```go
type InstanceState struct {
    InstanceID string
    Status     string    // "stopped" / "starting" / "running" / "error"
    PID        int
    StartedAt  time.Time
    LastError  string
    cmd        *exec.Cmd // 不序列化
}
```

### 4.4 ParamDef（来自 `--help`）

```go
type ParamDef struct {
    Flag     string `json:"flag"`      // "-m"
    Alias    string `json:"alias"`     // "--model"
    Desc     string `json:"desc"`
    HasValue bool   `json:"hasValue"`
    Group    string `json:"group"`
}
```

### 4.5 ParsedCommand（命令导入）

```go
type ParsedCommand struct {
    Executable string            `json:"executable"`
    ModelPath  string            `json:"modelPath"`
    Params     map[string]string `json:"params"`    // 归一化为主标志
    ExtraArgs  []string          `json:"extraArgs"` // 保留未知标志
}
```

## 5. 持久化与数据目录

数据目录为**可执行文件所在目录**（便携式布局），在 `internal/config/store.go`
中通过 `os.Executable()` + `EvalSymlinks` 解析。启动时执行写入探测；若目录不可写
（例如安装在 `Program Files` 下），应用弹出原生错误对话框并退出。

```
<程序目录>/
├── runtimes.json     # 已注册版本
├── instances.json    # 实例
├── runtimes/         # 已安装的 llama.cpp 构建
└── logs/             # 预留的磁盘日志目录
```

- JSON 采用原子写入（临时文件 + 重命名）。
- 目前**没有全局设置文件**；模型路径是实例级字段。

## 6. 后端核心模块

### 6.1 config（`internal/config/store.go`）
- `NewStore() (*Store, error)` —— 解析程序目录并校验可写性。
- `Path`、`RuntimesDir`、`LogsDir`、`ReadJSON`、`WriteJSON`。

### 6.2 runtime

**Registry（`runtime.go`）**
- 读写 `runtimes.json`；`List`、`Get`、`Add`、`Remove`。

**本地扫描（`scan.go`）**
- `ScanLocalDir(dir)` —— 查找 `llama-server.exe`，读取 `--version`，识别后端。
- `DetectBackend` —— 路径关键字 + `ggml-cuda.dll` / `ggml-vulkan.dll` / cudart 是否存在。

**--help 解析（`parse_help.go`）**
- `ParseHelp(help)` —— 将每行拆为标志列与说明，提取标志/别名/值占位符，并分配分组。
- `DefaultParams()` —— 解析失败时的内置回退表。
- `GetParams(exe)` —— 执行 `--help`、解析、失败回退默认表。

**下载（`download.go`）**
- `FetchReleases()` —— GitHub API，缓存 1 小时，识别限流。
- `DownloadRuntime(store, tag, backend, onProgress)`：
  - 非 CUDA：选择 `llama-<tag>-bin-win-<backend>-x64.zip`。
  - CUDA：`SelectCudaAssets` 同时选择主程序**和**匹配的
    `cudart-llama-bin-win-cuda-<ver>-x64.zip`，均解压到同一目录。
  - 目标 id：`<tag>-<backend>[-<variant>]`。
- `DownloadCudartFor(tag, variant, workDir, onProgress)` —— 供修复使用。
- `downloadAndExtract` —— 临时下载 + 安全解压（防 zip-slip）。

**CUDA 辅助（`cuda.go`）**
- `DetectDriverCUDA()` —— 从 `nvidia-smi`（PATH / System32 / NVSMI）解析 `CUDA Version`。
- `SelectCudaAssets(release, driverCUDA)` —— 选择不高于驱动版本的最新变体（否则取最低），并匹配 cudart。
- `MissingRuntimeLibs(rt)` / `ValidateRuntime(rt)` —— 检查
  `ggml-cuda.dll`、`cublas64_*.dll`、`cublasLt64_*.dll`、`cudart64_*.dll`
  （Vulkan：`ggml-vulkan.dll`）。

**孤儿目录（`orphans.go`）**
- `ListOrphanDirs(store, reg)` / `DeleteOrphanDir(store, name)`（防路径穿越）。

### 6.3 cmd

**命令构建（`builder.go`）**
- `Build(modelPath, params, extra)` —— 固定顺序：先 `-m`，其余参数按 flag 字典序，最后附加参数。
- `Quote(args)` —— 展示辅助。

**命令解释（`explainer.go`）**
- `Explain(args, defs, buildTag)` —— 按 flag 边界切分参数，查找说明，并附加版本相关告警（如 b10290 的 token 重复问题）。

**命令解析（`parser.go`）**
- `Tokenize(s)` —— 支持双引号的空白切分。
- `Parse(command, defs)` —— 生成 `ParsedCommand`；已知标志归一化为主标志，未知标志/值放入 `ExtraArgs`。

### 6.4 instance

**管理器（`manager.go`）**
- CRUD：`List`、`Get`、`Create`、`Update`、`Delete`。
- `RebindRuntime(oldID, newID)` —— 将实例改绑到新版本 id。
- `ReconcileBindings()` —— 启动时修复指向已删除版本的实例：唯一「构建号 + 后端」匹配则自动改绑。
- `runtimeDefs(rt)` —— 按版本缓存的 `--help` 参数表。
- `UnknownParams(rt, params)` —— 返回绑定版本不存在的标志（参数表过小时跳过，避免误报）。
- 内存日志缓冲（上限 3000 行）与状态跟踪。

**进程（`process.go`）**
- `Start`：
  1. `ValidateRuntime`（缺失 DLL）。
  2. `UnknownParams`（版本不支持的标志）。
  3. 记录**完整命令行**。
  4. 以管道方式启动，流式读取 `stdout`/`stderr`，监控退出。
- `Stop` —— Windows 下 `taskkill /F /T /PID`（进程树），否则直接 kill。
- `Restart` —— 停止 + 启动。
- `process_windows.go` 使用 `CREATE_NO_WINDOW`。

### 6.5 Wails 绑定（`app.go`）

版本管理：
- `ListRuntimes() []LlamaRuntime`
- `GetRuntime(id) (LlamaRuntime, error)`
- `FetchAvailableReleases() ([]Release, error)`
- `DownloadRuntime(tag, backend) (*LlamaRuntime, error)`
- `RepairRuntime(id) (*LlamaRuntime, error)`
- `CheckRuntime(id) ([]string, error)`
- `ListOrphanRuntimes() ([]OrphanDir, error)`
- `DeleteOrphanRuntime(name) error`
- `AddLocalRuntime(dir) (*LlamaRuntime, error)`
- `RemoveRuntime(id) error`
- `GetRuntimeParams(id) ([]ParamDef, error)`

实例管理：
- `ListInstances() []*LlamaInstance`
- `CreateInstance(inst) (*LlamaInstance, error)`
- `UpdateInstance(inst) error`
- `DeleteInstance(id) error`
- `StartInstance(id) error`
- `StopInstance(id) error`
- `RestartInstance(id) error`
- `GetInstanceState(id) InstanceState`
- `GetInstanceStates() []InstanceState`

命令工具：
- `PreviewCommand(inst) string`
- `ExplainCommand(inst) ([]Explanation, error)`
- `ParseCommand(command, runtimeID) ParsedCommand`

路径：
- `SelectDirectory() (string, error)`
- `SelectFile(filters) (string, error)`

日志 / 其他：
- `GetLogs(instanceID) []LogLine`
- `ClearLogs(instanceID)`
- `BaseDir() string`

### 6.6 事件（Go → Vue）

| 事件 | 载荷 |
|------|------|
| `instance:status` | `{ id, status, pid, error }` |
| `instance:log` | `{ id, line, stream }` |
| `runtime:progress` | `{ tag, backend, phase, name, downloaded, total, percent }` |

CUDA 下载期间 `phase` 为 `main` 或 `cudart`。

## 7. 前端设计

### 7.1 页面
- **Runtimes.vue** —— 已安装版本卡片（构建号、后端、CUDA 变体、路径）；添加（下载 / 本地目录）；带阶段的下载进度；缺库告警与「补全运行库」按钮；未注册目录列表与删除；参数表查看。
- **Instances.vue** —— 列表显示名称、绑定版本、模型、状态与操作（启动 / 停止 / 重启 / 编辑 / 日志 / 删除）；缺失版本显示红色标记。
- **InstanceEdit.vue** —— 左：参数分组树；中：表单（基本信息或选中分组的参数）；右：实时命令预览 + 解释器；顶部操作含「从命令导入」；未知参数告警条，提供「保留为附加参数」/「移除」。
- **Logs.vue** —— 实例选择、流过滤、搜索、自动滚动、清空。

### 7.2 参数面板动态渲染

```vue
<ParamGroup
  v-for="g in groupNames"
  :key="g"
  :name="g"
  :count="enabledCount(g)"
  :active="selectedGroup === g"
  @select="selectedGroup = $event"
/>
...
<ParamItem
  v-for="p in visibleParams"
  :key="p.flag"
  :def="p"
  :model-value="form.params[p.flag] ?? null"
  @update:model-value="setParam(p.flag, $event)"
/>
```

### 7.3 命令预览

本地计算（无后端往返）：先 `-m <模型>`，再按 flag 顺序排列参数，最后附加参数；含空格路径自动加引号。

### 7.4 Store 与 API
- `stores/runtimes.js`、`stores/instances.js`（Pinia）。
- `api/index.js` 封装生成的 `wailsjs/go/main/App` 绑定与 runtime 的 `EventsOn/EventsOff`。
- `App.vue` 订阅 `instance:status`、`instance:log`、`runtime:progress` 并更新 store。

## 8. 图标与构建

- 图标源：`build/appicon.png`（1024×1024），由 `tools/genicon.ps1` 生成（PowerShell + System.Drawing，无外部工具依赖）。
- Wails **仅在 `build/windows/icon.ico` 不存在时**生成它，且只在打包阶段进行。因此：
  - `tools/genicon.ps1` 在写入 `appicon.png` 后删除旧的 `icon.ico`。
  - 使用完整 `wails build` 构建；**`-nopackage` 会跳过图标/资源生成**。
- 前端图标：`frontend/public/icon.png`（favicon）与 `frontend/src/assets/images/icon.png`（侧边栏 logo）。

## 9. 开发路线

### Phase 1 —— 骨架与版本管理 ✅
- Wails + Vue 3 + Pinia 项目。
- `ListRuntimes` / `AddLocalRuntime` / `RemoveRuntime`。
- 版本管理 UI。
- `--version` / `--help` 解析。

### Phase 2 —— 实例与进程 ✅
- 实例 CRUD + 持久化。
- 启动 / 停止 / 重启。
- 状态事件推送。
- 实例列表 UI。

### Phase 3 —— 命令构建器 ✅
- 动态参数面板。
- 命令预览。
- 命令解释器。
- 实例编辑 UI。

### Phase 4 —— 日志与下载 ✅
- 实时日志流。
- GitHub Release 下载（含 CUDA cudart 拆分资产）。
- 下载进度事件。
- 日志页 UI。

### Phase 5 —— 优化（部分）
- ✅ CUDA 运行库校验 / 修复、孤儿目录清理。
- ✅ 启动对账、启动时参数校验。
- ✅ 从命令行导入。
- ⬜ 异常检测（token 重复等）。
- ⬜ 预设管理。
- ⬜ 多语言。
- ⬜ 打包与自动更新。
- ⬜ 全局设置页（默认模型目录、默认参数、日志上限等）。

## 10. 关键技术风险

| 风险 | 影响 | 缓解方案 |
|------|------|---------|
| GitHub API 限流 | 无法获取版本列表 | 缓存 1 小时；支持手动输入 tag |
| `--help` 格式变更 | 参数解析失败 | 多正则兼容；内置回退表；参数表过小时跳过校验 |
| CUDA 拆分资产 | CUDA 静默回退 CPU | 同时下载 cudart；按驱动选择变体；启动前 DLL 预检；修复按钮 |
| Windows 进程树 | 停止时残留子进程 | `taskkill /F /T`；杀进程树 |
| 修复时版本 id 变化 | 实例指向不存在的版本 | `RebindRuntime` + 启动 `ReconcileBindings` |
| 切换版本遗留旧 flag | llama.cpp 报未知参数 | `UnknownParams` 预检 + 编辑页保留/移除提示 |
| 旧版本 bug（如 b10290 token 重复） | 体验差 | 解释器中的版本告警；日志检测计划中 |
| 安装目录不可写 | 无法写入数据 | 启动时原生错误弹窗并退出 |
| 非 ASCII 路径 | 命令行解析错误 | 全程 UTF-8；含空格路径加引号 |

## 11. 附录

### 11.1 已验证版本
- 验证构建：b10930，Windows x86_64，CUDA 12.4（RTX 3080 Ti，驱动 596.49）。
- 最新版本：https://github.com/ggml-org/llama.cpp/releases

### 11.2 GitHub 资产命名
```
llama-<tag>-bin-win-cuda-<ver>-x64.zip     # CUDA 主程序
llama-<tag>-bin-win-vulkan-x64.zip
llama-<tag>-bin-win-cpu-x64.zip
llama-<tag>-bin-win-avx2-x64.zip           # （较早版本）
cudart-llama-bin-win-cuda-<ver>-x64.zip    # CUDA 运行时（cudart/cublas）
```

### 11.3 常用启动参数

| Flag | 说明 |
|------|------|
| `-m` | 模型文件路径 |
| `-ngl` | 放入 GPU 的层数 |
| `-c` | 上下文大小 |
| `-t` | 线程数 |
| `--host` | 监听地址 |
| `--port` | 监听端口 |
| `-np` | 并行请求数 |
| `--no-mmap` | 禁用内存映射 |

### 11.4 常用命令
- 开发：`wails dev`
- 构建：`wails build`
- 测试：`go test ./...`
- 图标：`powershell -ExecutionPolicy Bypass -File tools\genicon.ps1`
