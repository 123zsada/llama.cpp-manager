package runtime

import (
	"sort"
	"strings"
	"sync"
	"time"

	"llama-manager/internal/config"
)

// LlamaRuntime 表示一个已安装的 llama.cpp 版本。
type LlamaRuntime struct {
	ID          string    `json:"id"`
	BuildTag    string    `json:"buildTag"`
	Backend     string    `json:"backend"`
	Variant     string    `json:"variant"`
	Arch        string    `json:"arch"`
	Executable  string    `json:"executable"`
	WorkDir     string    `json:"workDir"`
	Source      string    `json:"source"`
	InstalledAt time.Time `json:"installedAt"`
}

// Registry 管理所有已安装版本，并负责持久化。
type Registry struct {
	store *config.Store

	mu    sync.RWMutex
	items []LlamaRuntime
}

// NewRegistry 从磁盘加载已注册版本。
func NewRegistry(store *config.Store) (*Registry, error) {
	r := &Registry{store: store}
	if err := store.ReadJSON("runtimes.json", &r.items); err != nil {
		return nil, err
	}
	return r, nil
}

// List 返回所有版本的副本，按构建号倒序排列。
func (r *Registry) List() []LlamaRuntime {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LlamaRuntime, len(r.items))
	copy(out, r.items)
	sort.SliceStable(out, func(i, j int) bool {
		return buildNumber(out[i].BuildTag) > buildNumber(out[j].BuildTag)
	})
	return out
}

// Get 按 ID 查找版本。
func (r *Registry) Get(id string) (LlamaRuntime, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, it := range r.items {
		if it.ID == id {
			return it, true
		}
	}
	return LlamaRuntime{}, false
}

// Add 注册（或覆盖）一个版本。
func (r *Registry) Add(rt LlamaRuntime) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, it := range r.items {
		if it.ID == rt.ID {
			r.items[i] = rt
			return r.saveLocked()
		}
	}
	r.items = append(r.items, rt)
	return r.saveLocked()
}

// Remove 移除版本注册信息（不删除磁盘文件）。
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	filtered := r.items[:0]
	for _, it := range r.items {
		if it.ID != id {
			filtered = append(filtered, it)
		}
	}
	r.items = filtered
	return r.saveLocked()
}

func (r *Registry) saveLocked() error {
	items := make([]LlamaRuntime, len(r.items))
	copy(items, r.items)
	return r.store.WriteJSON("runtimes.json", items)
}

// buildNumber 从 "b10290" 中提取 10290，用于排序。
func buildNumber(tag string) int {
	tag = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(tag)), "b")
	n := 0
	for _, c := range tag {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
