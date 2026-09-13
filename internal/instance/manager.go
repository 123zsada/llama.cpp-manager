package instance

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"llama-manager/internal/config"
	"llama-manager/internal/runtime"
)

const maxLogLines = 3000

// Manager 负责实例的 CRUD、进程生命周期与日志缓冲。
type Manager struct {
	store *config.Store
	reg   *runtime.Registry

	emitMu sync.RWMutex
	emit   func(event string, data ...interface{})

	defsMu    sync.Mutex
	defsCache map[string][]runtime.ParamDef

	mu        sync.RWMutex
	instances map[string]*LlamaInstance
	states    map[string]*InstanceState
	logs      map[string][]LogLine
	stopping  map[string]bool
}

// NewManager 创建实例管理器并加载持久化数据。
func NewManager(store *config.Store, reg *runtime.Registry) (*Manager, error) {
	m := &Manager{
		store:     store,
		reg:       reg,
		defsCache: map[string][]runtime.ParamDef{},
		instances: map[string]*LlamaInstance{},
		states:    map[string]*InstanceState{},
		logs:      map[string][]LogLine{},
		stopping:  map[string]bool{},
	}

	var list []*LlamaInstance
	if err := store.ReadJSON("instances.json", &list); err != nil {
		return nil, err
	}
	for _, inst := range list {
		if inst.Params == nil {
			inst.Params = map[string]string{}
		}
		m.instances[inst.ID] = inst
		m.states[inst.ID] = &InstanceState{InstanceID: inst.ID, Status: StatusStopped}
	}
	return m, nil
}

// SetEmitter 注入事件发送函数。
func (m *Manager) SetEmitter(fn func(event string, data ...interface{})) {
	m.emitMu.Lock()
	m.emit = fn
	m.emitMu.Unlock()
}

func (m *Manager) emitEvent(event string, data ...interface{}) {
	m.emitMu.RLock()
	fn := m.emit
	m.emitMu.RUnlock()
	if fn != nil {
		fn(event, data...)
	}
}

// List 返回所有实例副本。
func (m *Manager) List() []*LlamaInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*LlamaInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		cp := *inst
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

// Get 返回实例副本。
func (m *Manager) Get(id string) (*LlamaInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[id]
	if !ok {
		return nil, false
	}
	cp := *inst
	return &cp, true
}

// Create 创建实例。
func (m *Manager) Create(inst LlamaInstance) (*LlamaInstance, error) {
	if inst.Name == "" {
		return nil, fmt.Errorf("实例名称不能为空")
	}
	m.mu.Lock()
	if inst.ID == "" {
		inst.ID = uuid.NewString()
	}
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = time.Now()
	}
	if inst.Params == nil {
		inst.Params = map[string]string{}
	}
	cp := inst
	m.instances[inst.ID] = &cp
	m.states[inst.ID] = &InstanceState{InstanceID: inst.ID, Status: StatusStopped}
	m.logs[inst.ID] = nil
	err := m.saveLocked()
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return &cp, nil
}

// Update 更新实例。
func (m *Manager) Update(inst LlamaInstance) error {
	m.mu.Lock()
	existing, ok := m.instances[inst.ID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("实例不存在: %s", inst.ID)
	}
	if inst.Params == nil {
		inst.Params = map[string]string{}
	}
	if inst.CreatedAt.IsZero() {
		inst.CreatedAt = existing.CreatedAt
	}
	cp := inst
	m.instances[inst.ID] = &cp
	err := m.saveLocked()
	m.mu.Unlock()
	return err
}

// RebindRuntime 将所有引用 oldID 的实例改绑到 newID，返回受影响数量。
func (m *Manager) RebindRuntime(oldID, newID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := 0
	for _, inst := range m.instances {
		if inst.RuntimeID == oldID {
			inst.RuntimeID = newID
			changed++
		}
	}
	if changed == 0 {
		return 0, nil
	}
	return changed, m.saveLocked()
}

var knownBackends = map[string]bool{"cuda": true, "vulkan": true, "cpu": true, "avx2": true}

// parseRuntimeID 从版本 ID 中解析构建号与后端，例如 b10930-cuda-12.4 -> b10930, cuda。
func parseRuntimeID(id string) (buildTag, backend string, ok bool) {
	parts := strings.Split(id, "-")
	for i := 1; i < len(parts); i++ {
		if knownBackends[parts[i]] {
			return strings.Join(parts[:i], "-"), parts[i], true
		}
	}
	return "", "", false
}

// ReconcileBindings 修复指向已不存在版本的实例：当同一构建号与后端只有一个
// 版本时自动改绑，返回改绑数量。
func (m *Manager) ReconcileBindings() (int, error) {
	byKey := map[string][]string{}
	valid := map[string]bool{}
	for _, rt := range m.reg.List() {
		valid[rt.ID] = true
		key := rt.BuildTag + "|" + rt.Backend
		byKey[key] = append(byKey[key], rt.ID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	changed := 0
	for _, inst := range m.instances {
		if valid[inst.RuntimeID] {
			continue
		}
		buildTag, backend, ok := parseRuntimeID(inst.RuntimeID)
		if !ok {
			continue
		}
		matches := byKey[buildTag+"|"+backend]
		if len(matches) == 1 {
			inst.RuntimeID = matches[0]
			changed++
		}
	}
	if changed == 0 {
		return 0, nil
	}
	return changed, m.saveLocked()
}

// runtimeDefs 返回（并缓存）版本的参数表。
func (m *Manager) runtimeDefs(rt runtime.LlamaRuntime) []runtime.ParamDef {
	m.defsMu.Lock()
	if defs, ok := m.defsCache[rt.ID]; ok {
		m.defsMu.Unlock()
		return defs
	}
	m.defsMu.Unlock()

	defs := runtime.GetParams(rt.Executable)
	m.defsMu.Lock()
	m.defsCache[rt.ID] = defs
	m.defsMu.Unlock()
	return defs
}

// UnknownParams 返回实例参数中不属于该版本的标志。
// 参数表过小时（解析失败回退）不做校验，避免误报。
func (m *Manager) UnknownParams(rt runtime.LlamaRuntime, params map[string]string) []string {
	defs := m.runtimeDefs(rt)
	if len(defs) < 20 {
		return nil
	}
	known := map[string]bool{}
	for _, d := range defs {
		known[strings.ToLower(d.Flag)] = true
		if d.Alias != "" {
			known[strings.ToLower(d.Alias)] = true
		}
	}
	var unknown []string
	for k := range params {
		if !known[strings.ToLower(k)] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	return unknown
}

// Delete 删除实例，若正在运行则先停止。
func (m *Manager) Delete(id string) error {
	m.mu.RLock()
	state, ok := m.states[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("实例不存在: %s", id)
	}
	if state.Status == StatusRunning || state.Status == StatusStarting {
		if err := m.Stop(id); err != nil {
			return err
		}
	}

	m.mu.Lock()
	delete(m.instances, id)
	delete(m.states, id)
	delete(m.logs, id)
	err := m.saveLocked()
	m.mu.Unlock()
	return err
}

// GetState 返回实例运行时状态。
func (m *Manager) GetState(id string) InstanceState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if st, ok := m.states[id]; ok {
		return InstanceState{
			InstanceID: st.InstanceID,
			Status:     st.Status,
			PID:        st.PID,
			StartedAt:  st.StartedAt,
			LastError:  st.LastError,
		}
	}
	return InstanceState{InstanceID: id, Status: StatusStopped}
}

// GetLogs 返回实例日志副本。
func (m *Manager) GetLogs(id string) []LogLine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	src := m.logs[id]
	out := make([]LogLine, len(src))
	copy(out, src)
	return out
}

// ClearLogs 清空实例日志。
func (m *Manager) ClearLogs(id string) {
	m.mu.Lock()
	m.logs[id] = nil
	m.mu.Unlock()
}

// AutoStartAll 启动所有标记为自启的实例。
func (m *Manager) AutoStartAll() {
	for _, inst := range m.List() {
		if inst.AutoStart {
			_ = m.Start(inst.ID)
		}
	}
}

// StopAll 停止所有实例，用于应用退出。
func (m *Manager) StopAll() {
	for _, inst := range m.List() {
		_ = m.Stop(inst.ID)
	}
}

func (m *Manager) saveLocked() error {
	list := make([]*LlamaInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		list = append(list, inst)
	}
	return m.store.WriteJSON("instances.json", list)
}

func (m *Manager) appendLog(id string, line LogLine) {
	m.mu.Lock()
	if line.Time.IsZero() {
		line.Time = time.Now()
	}
	buf := append(m.logs[id], line)
	if len(buf) > maxLogLines {
		buf = buf[len(buf)-maxLogLines:]
	}
	m.logs[id] = buf
	m.mu.Unlock()
	m.emitEvent("instance:log", LogEvent{ID: id, Line: line.Line, Stream: line.Stream})
}

func (m *Manager) updateState(id, status string, pid int, lastErr string) {
	m.mu.Lock()
	st, ok := m.states[id]
	if !ok {
		st = &InstanceState{InstanceID: id}
		m.states[id] = st
	}
	st.Status = status
	st.PID = pid
	st.LastError = lastErr
	if status == StatusRunning {
		st.StartedAt = time.Now()
	}
	m.mu.Unlock()
	m.emitEvent("instance:status", StatusEvent{ID: id, Status: status, PID: pid, Error: lastErr})
}

// LogsDir 确保日志目录存在并返回路径。
func (m *Manager) LogsDir() string {
	dir := m.store.LogsDir()
	_ = os.MkdirAll(dir, 0o755)
	return dir
}
