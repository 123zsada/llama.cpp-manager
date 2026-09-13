package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"llama-manager/internal/cmd"
	"llama-manager/internal/config"
	"llama-manager/internal/instance"
	llamaruntime "llama-manager/internal/runtime"
)

// App 是暴露给前端的应用对象。
type App struct {
	ctx    context.Context
	store  *config.Store
	reg    *llamaruntime.Registry
	mgrs   *instance.Manager
	params map[string][]llamaruntime.ParamDef
}

// NewApp 创建应用对象。
func NewApp(store *config.Store) *App {
	return &App{store: store, params: map[string][]llamaruntime.ParamDef{}}
}

// startup 在应用启动时调用，初始化各管理器。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	reg, err := llamaruntime.NewRegistry(a.store)
	if err != nil {
		showFatalError("llama.cpp Manager 启动失败", "加载版本列表失败: "+err.Error())
		os.Exit(1)
	}
	a.reg = reg

	mgr, err := instance.NewManager(a.store, reg)
	if err != nil {
		showFatalError("llama.cpp Manager 启动失败", "加载实例列表失败: "+err.Error())
		os.Exit(1)
	}
	a.mgrs = mgr
	mgr.SetEmitter(func(event string, data ...interface{}) {
		wruntime.EventsEmit(a.ctx, event, data...)
	})
	if n, err := mgr.ReconcileBindings(); err != nil {
		println("实例版本对账失败:", err.Error())
	} else if n > 0 {
		println("已自动重绑定实例数量:", n)
	}
	mgr.AutoStartAll()
}

func (a *App) shutdown(ctx context.Context) {
	if a.mgrs != nil {
		a.mgrs.StopAll()
	}
}

// ---------- 版本管理 ----------

// ListRuntimes 返回所有已安装版本。
func (a *App) ListRuntimes() []llamaruntime.LlamaRuntime {
	if a.reg == nil {
		return []llamaruntime.LlamaRuntime{}
	}
	return a.reg.List()
}

// GetRuntime 返回指定版本。
func (a *App) GetRuntime(id string) (llamaruntime.LlamaRuntime, error) {
	rt, ok := a.reg.Get(id)
	if !ok {
		return llamaruntime.LlamaRuntime{}, fmt.Errorf("版本不存在: %s", id)
	}
	return rt, nil
}

// FetchAvailableReleases 获取可下载的官方版本列表。
func (a *App) FetchAvailableReleases() ([]llamaruntime.Release, error) {
	return llamaruntime.FetchReleases()
}

// DownloadRuntime 下载并注册一个版本。
func (a *App) DownloadRuntime(tag, backend string) (*llamaruntime.LlamaRuntime, error) {
	rt, err := llamaruntime.DownloadRuntime(a.store, tag, backend, func(phase, name string, downloaded, total int64) {
		a.emitProgress(tag, backend, phase, name, downloaded, total)
	})
	if err != nil {
		return nil, err
	}
	if err := a.reg.Add(*rt); err != nil {
		return nil, err
	}
	return rt, nil
}

// RepairRuntime 为缺库的 CUDA 版本补全 cudart 运行库。
func (a *App) RepairRuntime(id string) (*llamaruntime.LlamaRuntime, error) {
	rt, ok := a.reg.Get(id)
	if !ok {
		return nil, fmt.Errorf("版本不存在: %s", id)
	}
	if rt.Backend != "cuda" {
		return nil, fmt.Errorf("仅 CUDA 版本需要补全运行库")
	}
	if len(llamaruntime.MissingRuntimeLibs(rt)) == 0 {
		return &rt, nil
	}

	// 旧记录没有 CUDA 版本信息，无法匹配 cudart，整包重新下载。
	if rt.Variant == "" {
		fresh, err := llamaruntime.DownloadRuntime(a.store, rt.BuildTag, "cuda", func(phase, name string, downloaded, total int64) {
			a.emitProgress(rt.BuildTag, "cuda", phase, name, downloaded, total)
		})
		if err != nil {
			return nil, err
		}
		if err := a.reg.Add(*fresh); err != nil {
			return nil, err
		}
		if fresh.ID != rt.ID {
			if _, err := a.mgrs.RebindRuntime(rt.ID, fresh.ID); err != nil {
				return nil, err
			}
			_ = a.reg.Remove(rt.ID)
		}
		return fresh, nil
	}

	err := llamaruntime.DownloadCudartFor(rt.BuildTag, rt.Variant, rt.WorkDir, func(phase, name string, downloaded, total int64) {
		a.emitProgress(rt.BuildTag, rt.Backend, phase, name, downloaded, total)
	})
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

// CheckRuntime 返回版本缺失的运行库列表。
func (a *App) CheckRuntime(id string) ([]string, error) {
	rt, ok := a.reg.Get(id)
	if !ok {
		return nil, fmt.Errorf("版本不存在: %s", id)
	}
	return llamaruntime.MissingRuntimeLibs(rt), nil
}

// ListOrphanRuntimes 返回未注册的版本目录。
func (a *App) ListOrphanRuntimes() ([]llamaruntime.OrphanDir, error) {
	return llamaruntime.ListOrphanDirs(a.store, a.reg)
}

// DeleteOrphanRuntime 删除未注册的版本目录。
func (a *App) DeleteOrphanRuntime(name string) error {
	return llamaruntime.DeleteOrphanDir(a.store, name)
}

func (a *App) emitProgress(tag, backend, phase, name string, downloaded, total int64) {
	percent := 0.0
	if total > 0 {
		percent = float64(downloaded) / float64(total) * 100
	}
	wruntime.EventsEmit(a.ctx, "runtime:progress", map[string]interface{}{
		"tag":        tag,
		"backend":    backend,
		"phase":      phase,
		"name":       name,
		"downloaded": downloaded,
		"total":      total,
		"percent":    percent,
	})
}

// AddLocalRuntime 注册一个本地已有的构建目录。
func (a *App) AddLocalRuntime(dir string) (*llamaruntime.LlamaRuntime, error) {
	rt, err := llamaruntime.ScanLocalDir(dir)
	if err != nil {
		return nil, err
	}
	if err := a.reg.Add(*rt); err != nil {
		return nil, err
	}
	return rt, nil
}

// RemoveRuntime 移除版本注册信息。
func (a *App) RemoveRuntime(id string) error {
	return a.reg.Remove(id)
}

// GetRuntimeParams 获取（并缓存）版本的参数表。
func (a *App) GetRuntimeParams(id string) ([]llamaruntime.ParamDef, error) {
	if _, ok := a.reg.Get(id); !ok {
		return nil, fmt.Errorf("版本不存在: %s", id)
	}
	defs, _ := a.paramsFor(id)
	return defs, nil
}

// paramsFor 返回指定版本的参数表与构建号，并缓存参数表。
func (a *App) paramsFor(runtimeID string) ([]llamaruntime.ParamDef, string) {
	rt, ok := a.reg.Get(runtimeID)
	if !ok {
		return llamaruntime.DefaultParams(), ""
	}
	if defs, ok := a.params[runtimeID]; ok {
		return defs, rt.BuildTag
	}
	defs := llamaruntime.GetParams(rt.Executable)
	a.params[runtimeID] = defs
	return defs, rt.BuildTag
}

// ---------- 实例管理 ----------

// ListInstances 返回所有实例。
func (a *App) ListInstances() []*instance.LlamaInstance {
	if a.mgrs == nil {
		return []*instance.LlamaInstance{}
	}
	return a.mgrs.List()
}

// CreateInstance 创建实例。
func (a *App) CreateInstance(inst instance.LlamaInstance) (*instance.LlamaInstance, error) {
	return a.mgrs.Create(inst)
}

// UpdateInstance 更新实例。
func (a *App) UpdateInstance(inst instance.LlamaInstance) error {
	return a.mgrs.Update(inst)
}

// DeleteInstance 删除实例。
func (a *App) DeleteInstance(id string) error {
	return a.mgrs.Delete(id)
}

// StartInstance 启动实例。
func (a *App) StartInstance(id string) error {
	return a.mgrs.Start(id)
}

// StopInstance 停止实例。
func (a *App) StopInstance(id string) error {
	return a.mgrs.Stop(id)
}

// RestartInstance 重启实例。
func (a *App) RestartInstance(id string) error {
	return a.mgrs.Restart(id)
}

// GetInstanceState 返回实例状态。
func (a *App) GetInstanceState(id string) instance.InstanceState {
	return a.mgrs.GetState(id)
}

// GetInstanceStates 返回所有实例状态。
func (a *App) GetInstanceStates() []instance.InstanceState {
	list := a.mgrs.List()
	out := make([]instance.InstanceState, 0, len(list))
	for _, inst := range list {
		out = append(out, a.mgrs.GetState(inst.ID))
	}
	return out
}

// ---------- 命令工具 ----------

// PreviewCommand 返回实例的完整命令行。
func (a *App) PreviewCommand(inst instance.LlamaInstance) string {
	exe := ""
	if rt, ok := a.reg.Get(inst.RuntimeID); ok {
		exe = rt.Executable
	}
	args := cmd.Build(inst.ModelPath, inst.Params, inst.ExtraArgs)
	if exe == "" {
		return cmd.Quote(args)
	}
	return cmd.Quote(append([]string{exe}, args...))
}

// ExplainCommand 逐参数解释实例命令。
func (a *App) ExplainCommand(inst instance.LlamaInstance) ([]cmd.Explanation, error) {
	defs, buildTag := a.paramsFor(inst.RuntimeID)
	args := cmd.Build(inst.ModelPath, inst.Params, inst.ExtraArgs)
	return cmd.Explain(args, defs, buildTag), nil
}

// ParseCommand 将完整命令行解析为结构化实例字段。
func (a *App) ParseCommand(command, runtimeID string) cmd.ParsedCommand {
	defs, _ := a.paramsFor(runtimeID)
	return cmd.Parse(command, defs)
}

// ---------- 路径选择 ----------

// SelectDirectory 打开目录选择对话框。
func (a *App) SelectDirectory() (string, error) {
	return wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "选择目录",
	})
}

// SelectFile 打开文件选择对话框。filters 格式为 "名称 (*.gguf)|*.gguf|..."。
func (a *App) SelectFile(filters string) (string, error) {
	opts := wruntime.OpenDialogOptions{Title: "选择文件"}
	opts.Filters = parseFilters(filters)
	if len(opts.Filters) == 0 {
		opts.Filters = []wruntime.FileFilter{{DisplayName: "GGUF 模型 (*.gguf)", Pattern: "*.gguf"}}
	}
	return wruntime.OpenFileDialog(a.ctx, opts)
}

func parseFilters(s string) []wruntime.FileFilter {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, "|")
	filters := []wruntime.FileFilter{}
	for i := 0; i+1 < len(parts); i += 2 {
		name := strings.TrimSpace(parts[i])
		pattern := strings.TrimSpace(parts[i+1])
		if name == "" || pattern == "" {
			continue
		}
		filters = append(filters, wruntime.FileFilter{DisplayName: name, Pattern: pattern})
	}
	return filters
}

// ---------- 日志 ----------

// GetLogs 返回实例日志。
func (a *App) GetLogs(instanceID string) []instance.LogLine {
	if a.mgrs == nil {
		return []instance.LogLine{}
	}
	return a.mgrs.GetLogs(instanceID)
}

// ClearLogs 清空实例日志。
func (a *App) ClearLogs(instanceID string) {
	if a.mgrs != nil {
		a.mgrs.ClearLogs(instanceID)
	}
}

// ---------- 工具 ----------

// BaseDir 返回数据目录，便于前端展示。
func (a *App) BaseDir() string {
	if a.store == nil {
		return ""
	}
	abs, _ := filepath.Abs(a.store.BaseDir)
	return abs
}
