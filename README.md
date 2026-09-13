# llama.cpp Manager

图形化管理多个 [llama.cpp](https://github.com/ggml-org/llama.cpp) 实例的 Windows 桌面应用，基于 **Go + Wails v2 + Vue 3**。

支持多版本切换、可视化命令构建、进程生命周期管理与实时日志监控。

## 功能

### 版本管理
- 从 GitHub Releases 下载官方预构建二进制（Windows x86_64），或添加本地已有构建目录。
- 自动识别后端（CUDA / Vulkan / CPU / AVX2）与构建号。
- **CUDA 拆分资产支持**：自动同时下载主程序与独立的 cudart 运行库（cublas/cudart）。
- **驱动感知选择**：通过 `nvidia-smi` 选择与驱动兼容的 CUDA 变体。
- **运行库校验与修复**：检测缺失 DLL 并一键补全。
- **孤儿目录清理**：清理未注册的版本目录，回收磁盘。

### 实例管理
- 创建多个实例，每个绑定一个版本。
- 启动 / 停止 / 重启 / 删除，实时状态显示。
- 启动对账：自动修复指向已删除版本的实例。
- 启动前预检：缺失运行库或版本不支持的参数会明确报错。

### 可视化命令构建
- 依据所选版本的 `--help` 动态生成分组参数面板（模型 / GPU / 上下文 / 采样 / 服务器 / 性能）。
- 实时命令预览与逐参数解释。
- **从完整命令行导入**：粘贴 `llama-server ...` 命令自动解析。

### 日志与监控
- 实时捕获 `stdout` / `stderr`，按实例分离展示。
- 过滤 / 搜索 / 清空，启动前记录完整命令行。

## 下载

前往 [Releases](https://github.com/123zsada/llama.cpp-manager/releases) 下载 `llama-manager.exe`，直接运行（便携式，无需安装）。

## 系统要求

- Windows 10 / 11 x64
- Microsoft Edge WebView2 Runtime（Windows 11 及多数 Windows 10 已内置；缺失时请从微软官网安装）

## 数据目录

所有配置与下载的版本均保存在**程序所在目录**（便携布局）：

```
<程序目录>/
├── runtimes.json     # 已安装版本
├── instances.json    # 实例
├── runtimes/         # 下载的 llama.cpp 构建
└── logs/             # 预留日志目录
```

若程序目录不可写（例如安装在 `Program Files`），启动时会弹出错误提示并退出。

## 使用

1. **版本管理** → 添加版本（从 GitHub 下载或选择本地目录）。CUDA 会按驱动自动选择变体并下载运行库。
2. **实例** → 新建实例：选择绑定版本、模型文件，按分组配置参数，右侧实时预览完整命令。
3. 保存后启动；在**日志**页查看实时输出。

## 从源码构建

需要 Go 1.25+、Node.js、Wails v2 CLI。

```bash
wails dev      # 开发模式（热重载）
wails build    # 构建（含图标/资源，请勿加 -nopackage）
go test ./...  # 运行测试
```

重新生成图标：

```powershell
powershell -ExecutionPolicy Bypass -File tools\genicon.ps1
```

## 文档

- 开发文档：[doc/DEVELOPMENT.zh-CN.md](doc/DEVELOPMENT.zh-CN.md)

## 已知问题

- CUDA 版本需与显卡驱动匹配，驱动过旧会导致回退 CPU；应用会在启动前校验并提示。
- 部分旧版本（如 b10290）存在 token 重复 bug，命令解释器会给出提示。
