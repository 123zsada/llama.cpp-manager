package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Store 负责本地配置与数据的持久化。
// 所有数据统一存放在程序（可执行文件）所在目录中。
type Store struct {
	BaseDir string
}

// NewStore 以程序所在目录作为数据目录，并确保其可写。
func NewStore() (*Store, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("无法获取程序路径: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)

	// 写入探测：程序目录不可写（如安装在 Program Files）时直接报错。
	probe := filepath.Join(dir, ".llama-manager-write-test")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("程序目录不可写: %s: %w", dir, err)
	}
	f.Close()
	_ = os.Remove(probe)

	return &Store{BaseDir: dir}, nil
}

// Path 返回基础目录下指定文件的绝对路径。
func (s *Store) Path(name string) string {
	return filepath.Join(s.BaseDir, name)
}

// RuntimesDir 返回存放已安装版本的目录。
func (s *Store) RuntimesDir() string {
	return filepath.Join(s.BaseDir, "runtimes")
}

// LogsDir 返回存放日志的目录。
func (s *Store) LogsDir() string {
	return filepath.Join(s.BaseDir, "logs")
}

// ReadJSON 读取 JSON 文件，文件不存在时返回 nil。
func (s *Store) ReadJSON(name string, v interface{}) error {
	data, err := os.ReadFile(s.Path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// WriteJSON 原子写入 JSON 文件。
func (s *Store) WriteJSON(name string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path(name + ".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path(name))
}
