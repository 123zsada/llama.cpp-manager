package instance

import (
	"testing"

	"llama-manager/internal/config"
	"llama-manager/internal/runtime"
)

func TestRebindRuntime(t *testing.T) {
	store := &config.Store{BaseDir: t.TempDir()}
	reg, err := runtime.NewRegistry(store)
	if err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(store, reg)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Create(LlamaInstance{Name: "a", RuntimeID: "old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(LlamaInstance{Name: "b", RuntimeID: "other"}); err != nil {
		t.Fatal(err)
	}

	n, err := m.RebindRuntime("old", "new")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 rebound instance, got %d", n)
	}

	reloaded, err := NewManager(store, reg)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, inst := range reloaded.List() {
		if inst.RuntimeID == "new" {
			count++
		}
		if inst.Name == "b" && inst.RuntimeID != "other" {
			t.Errorf("instance b should keep runtimeId 'other', got %q", inst.RuntimeID)
		}
	}
	if count != 1 {
		t.Fatalf("expected persisted rebind, got %d", count)
	}
}

func TestReconcileBindings(t *testing.T) {
	store := &config.Store{BaseDir: t.TempDir()}
	reg, err := runtime.NewRegistry(store)
	if err != nil {
		t.Fatal(err)
	}
	// 同一构建号 + 后端只有一个版本 -> 唯一匹配
	if err := reg.Add(runtime.LlamaRuntime{ID: "b10930-cuda-12.4", BuildTag: "b10930", Backend: "cuda"}); err != nil {
		t.Fatal(err)
	}
	// 另一个构建号存在两个变体 -> 歧义，不应自动改绑
	if err := reg.Add(runtime.LlamaRuntime{ID: "b1000-cpu", BuildTag: "b1000", Backend: "cpu"}); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(runtime.LlamaRuntime{ID: "b1000-avx2", BuildTag: "b1000", Backend: "avx2"}); err != nil {
		t.Fatal(err)
	}

	m, err := NewManager(store, reg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(LlamaInstance{Name: "dangling", RuntimeID: "b10930-cuda"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(LlamaInstance{Name: "valid", RuntimeID: "b1000-cpu"}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(LlamaInstance{Name: "unknown", RuntimeID: "b99999-cpu"}); err != nil {
		t.Fatal(err)
	}

	n, err := m.ReconcileBindings()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reconciled instance, got %d", n)
	}

	got := map[string]string{}
	for _, inst := range m.List() {
		got[inst.Name] = inst.RuntimeID
	}
	if got["dangling"] != "b10930-cuda-12.4" {
		t.Errorf("dangling not rebound: %q", got["dangling"])
	}
	if got["valid"] != "b1000-cpu" {
		t.Errorf("valid instance changed: %q", got["valid"])
	}
	if got["unknown"] != "b99999-cpu" {
		t.Errorf("unknown instance changed: %q", got["unknown"])
	}
}
