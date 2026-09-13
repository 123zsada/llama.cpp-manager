package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreUsesExecutableDir(t *testing.T) {
	s, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() error: %v", err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error: %v", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	want := filepath.Dir(exe)
	if s.BaseDir != want {
		t.Fatalf("BaseDir = %q, want %q", s.BaseDir, want)
	}
}
