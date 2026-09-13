package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"llama-manager/internal/config"
)

// OrphanDir 表示一个位于 runtimes 目录下但未在注册表中的版本目录。
type OrphanDir struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListOrphanDirs 返回 runtimes 目录下未注册的子目录。
func ListOrphanDirs(store *config.Store, reg *Registry) ([]OrphanDir, error) {
	registered := map[string]bool{}
	if reg != nil {
		for _, rt := range reg.List() {
			registered[rt.ID] = true
		}
	}

	base := store.RuntimesDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []OrphanDir{}, nil
		}
		return nil, err
	}

	out := []OrphanDir{}
	for _, e := range entries {
		if !e.IsDir() || registered[e.Name()] {
			continue
		}
		out = append(out, OrphanDir{Name: e.Name(), Path: filepath.Join(base, e.Name())})
	}
	return out, nil
}

// DeleteOrphanDir 删除 runtimes 目录下指定的直接子目录。
func DeleteOrphanDir(store *config.Store, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("非法的目录名: %s", name)
	}
	base := filepath.Clean(store.RuntimesDir())
	target := filepath.Join(base, name)
	rel, err := filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("非法的目录路径: %s", name)
	}
	return os.RemoveAll(target)
}
